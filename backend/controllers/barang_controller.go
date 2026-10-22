package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

func BarangJual(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req models.JualReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.Items) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
		// harga jual wajib diisi per barang (manual)
		harga := map[int]float64{}
		idsJual := make([]int, 0, len(req.Items))
		for _, it := range req.Items {
			if it.Harga <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "harga jual setiap barang wajib diisi (lebih dari 0)"})
				return
			}
			if _, dobel := harga[it.ItemID]; dobel {
				c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang yang dipilih dua kali"})
				return
			}
			harga[it.ItemID] = it.Harga
			idsJual = append(idsJual, it.ItemID)
		}
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		boID, err := services.IDBackOffice(db)
		if err != nil {
			utils.Gagal(c, err)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		items, err := services.AmbilBarang(tx, idsJual)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for _, it := range items {
			if it.Status != "tersedia" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sedang tidak tersedia (%s)", it.Name, it.Status)})
				return
			}
			if cl.Role == "cabang" && it.Location != cl.LocationID {
				c.JSON(http.StatusForbidden, gin.H{"error": "bukan barang di lokasi kamu"})
				return
			}
		}

		note := req.Note
		if note == "" {
			note = "Terjual"
		}
		// hasil jual masuk ke kas lokasi tempat barang dijual (satu catatan kas per lokasi)
		type kasJual struct {
			Total float64
			IDs   []int
		}
		perLokasi := map[int]*kasJual{}
		urutLokasi := []int{}
		var totalSemua float64
		for _, it := range items {
			cat := "laku"
			if it.Location == boID {
				cat = "jual"
			}
			if _, err := tx.Exec(services.SQLStokItem, it.Location, "out", cat, it.Name, it.Weight, it.Karat, note, tgl, cl.UserID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'terjual', harga_jual = ?, tanggal_jual = ? WHERE id = ?", harga[it.ID], tgl, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			kj := perLokasi[it.Location]
			if kj == nil {
				kj = &kasJual{}
				perLokasi[it.Location] = kj
				urutLokasi = append(urutLokasi, it.Location)
			}
			kj.Total += harga[it.ID]
			kj.IDs = append(kj.IDs, it.ID)
			totalSemua += harga[it.ID]
		}
		for _, lokasi := range urutLokasi {
			kj := perLokasi[lokasi]
			kategoriKas := "omzet"
			if lokasi == boID {
				kategoriKas = "jual_barang"
			}
			catatanKas := fmt.Sprintf("Penjualan %d barang", len(kj.IDs))
			if req.Note != "" {
				catatanKas += ": " + req.Note
			}
			res, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
				 VALUES (?, 'in', ?, ?, ?, ?, ?)`,
				lokasi, kategoriKas, kj.Total, catatanKas, tgl, cl.UserID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			kasID, _ := res.LastInsertId()
			for _, itemID := range kj.IDs {
				if _, err := tx.Exec("UPDATE items SET jual_cash_id = ? WHERE id = ?", kasID, itemID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			services.CatatAudit(tx, cl.UserID, "create_auto", "cash_transactions", kasID, gin.H{"penjualan": len(kj.IDs), "total": kj.Total})
		}
		services.CatatAudit(tx, cl.UserID, "jual", "items", 0, req)

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang terjual, kas bertambah %s", len(items), utils.RupiahTeks(totalSemua))})
	}
}

// ---------- BATALKAN BARANG MASUK ----------

var kategoriMasuk = map[string]map[string]bool{
	"cabang":      {"beli_customer": true, "stok_awal": true},
	"back_office": {"pabrik": true, "konsinyasi": true, "stok_awal": true},
}

// ---------- INPUT BARANG MASUK ----------

func BarangCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req models.BarangMasukReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if !kategoriMasuk[tipe][req.Category] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "asal barang tidak valid untuk lokasi ini"})
			return
		}
		if len(req.Items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi minimal satu barang"})
			return
		}
		if len(req.Items) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali input"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.EntryDate)
		if !ok {
			return
		}

		beliCustomer := req.Category == "beli_customer"
		kirimKeBO := req.Posisi == "bo"
		if kirimKeBO && !(tipe == "cabang" && beliCustomer) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hanya barang dari customer yang bisa langsung diantar ke back office"})
			return
		}
		kondisi := "bersih"
		if beliCustomer {
			kondisi = "kotor"
		}
		catatanUmum := strings.TrimSpace(req.Note)

		var totalBeli float64
		for i := range req.Items {
			it := &req.Items[i]
			baris := i + 1
			it.Note = strings.TrimSpace(it.Note)
			if it.Note == "" {
				it.Note = catatanUmum
			}
			it.Brand = strings.TrimSpace(it.Brand)
			it.ItemName = strings.TrimSpace(it.ItemName)

			if it.WeightGram <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: berat wajib diisi", baris)})
				return
			}
			switch it.Jenis {
			case "logam_mulia":
				if it.Brand == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: merek logam mulia wajib diisi", baris)})
					return
				}
				it.Karat = "24K"
				it.ItemName = "Logam mulia " + it.Brand
			case "perhiasan":
				if it.ItemName == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: nama perhiasan wajib diisi", baris)})
					return
				}
				k, ok := utils.NormalKarat(it.Karat)
				if !ok {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: kadar harus 1K sampai 24K", baris)})
					return
				}
				it.Karat = k
				it.Brand = ""
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: jenis barang tidak valid", baris)})
				return
			}
			if it.HargaBeli < 0 || (beliCustomer && it.HargaBeli <= 0) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: harga beli wajib diisi", baris)})
				return
			}
			totalBeli += it.HargaBeli
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var cashID interface{}
		kasKeluar := 0.0
		if beliCustomer {
			kasKeluar = totalBeli
			note := fmt.Sprintf("Beli dari customer: %d barang", len(req.Items))
			if catatanUmum != "" {
				note += " - " + catatanUmum
			}
			res, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
				 VALUES (?,?,?,?,?,?,?)`,
				locID, "out", "pembelian_customer", totalBeli, note, tgl, cl.UserID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			id, _ := res.LastInsertId()
			cashID = id
			services.CatatAudit(tx, cl.UserID, "create_auto", "cash_transactions", id, gin.H{"pembelian_customer": len(req.Items), "total": totalBeli})
		}

		var lastID int64
		for _, it := range req.Items {
			res, err := tx.Exec(
				`INSERT INTO items
				 (item_name, karat, weight_gram, origin_location_id, location_id, source_category, entry_date, note, created_by,
				  jenis, brand, kondisi, harga_beli, purchase_cash_id)
				 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				it.ItemName, it.Karat, it.WeightGram, locID, locID, req.Category, tgl, it.Note, cl.UserID,
				it.Jenis, sql.NullString{String: it.Brand, Valid: it.Brand != ""}, kondisi, it.HargaBeli, cashID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			id, _ := res.LastInsertId()
			lastID = id
			tx.Exec("UPDATE items SET item_no = ? WHERE id = ?", fmt.Sprintf("BRG-%05d", id), id)
			if kirimKeBO {
				tx.Exec("UPDATE items SET status = 'dikirim', kirim_tgl = ? WHERE id = ?", tgl, id)
			}
			if _, err := tx.Exec(services.SQLStokItem, locID, "in", req.Category, it.ItemName, it.WeightGram, it.Karat,
				it.Note, tgl, cl.UserID, id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		services.CatatAudit(tx, cl.UserID, "create", "items", lastID, gin.H{"jumlah_barang": len(req.Items), "request": req})

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		pesan := fmt.Sprintf("%d barang tersimpan", len(req.Items))
		if kirimKeBO {
			pesan += ", diantar ke back office (menunggu konfirmasi)"
		}
		c.JSON(http.StatusCreated, gin.H{"message": pesan, "jumlah": len(req.Items), "kas_keluar": kasKeluar})
	}
}

// ---------- DAFTAR BARANG ----------

func BarangVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		reason, ok := utils.BacaAlasan(c)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var status string
		var isVoid bool
		var harga float64
		var cashID sql.NullInt64
		err = tx.QueryRow("SELECT status, is_void, harga_beli, purchase_cash_id FROM items WHERE id = ? FOR UPDATE", id).
			Scan(&status, &isVoid, &harga, &cashID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "barang tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}
		if status != "tersedia" && status != "dikirim" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang sedang diproses, tidak bisa dibatalkan"})
			return
		}
		var n int
		tx.QueryRow("SELECT COUNT(*) FROM stock_movements WHERE item_id = ? AND is_void = FALSE", id).Scan(&n)
		if n != 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang sudah pernah dipindah atau diproses, tidak bisa dibatalkan"})
			return
		}

		tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE item_id = ?", reason, id)
		tx.Exec("UPDATE items SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		services.CatatAudit(tx, cl.UserID, "void", "items", int64(id), gin.H{"reason": reason})

		if cashID.Valid && harga > 0 {
			var amt float64
			if err := tx.QueryRow("SELECT amount FROM cash_transactions WHERE id = ? FOR UPDATE", cashID.Int64).Scan(&amt); err == nil {
				baru := amt - harga
				if baru < 0.005 {
					tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, cashID.Int64)
				} else {
					tx.Exec("UPDATE cash_transactions SET amount = ? WHERE id = ?", baru, cashID.Int64)
				}
				services.CatatAudit(tx, cl.UserID, "koreksi_kas", "cash_transactions", cashID.Int64,
					gin.H{"dari": amt, "menjadi": baru, "alasan": reason})
			}
		}

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang dibatalkan"})
	}
}

func BarangList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)

		q := `SELECT i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), i.weight_gram,
		             i.jenis, IFNULL(i.brand,''), i.kondisi, i.harga_beli,
		             i.location_id, l.name, i.origin_location_id, o.name, i.status, i.entry_date, IFNULL(i.note,''),
		             i.kirim_tgl, i.belum_sampai, IFNULL(i.belum_sampai_note,''),
		             IF(i.status = 'cuci', IFNULL(w.washer_name,''), ''),
		             IF(i.status = 'cuci', w.start_date, NULL),
		             IFNULL(i.kirim_ke_location_id,0), IFNULL(k.name,''),
		             i.harga_jual, i.tanggal_jual
		      FROM items i
		      JOIN locations l ON l.id = i.location_id
		      JOIN locations o ON o.id = i.origin_location_id
		      LEFT JOIN locations k ON k.id = i.kirim_ke_location_id
		      LEFT JOIN washing_jobs w ON w.id = i.washing_job_id
		      WHERE i.is_void = FALSE`
		args := []interface{}{}

		if cl.Role == "cabang" {
			q += " AND (i.location_id = ? OR i.kirim_ke_location_id = ?)"
			args = append(args, cl.LocationID, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" && loc != "0" {
			q += " AND i.location_id = ?"
			args = append(args, loc)
		}
		if s := c.Query("status"); s == "dikirim" {
			q += " AND i.status IN ('dikirim','kembali')"
		} else if s != "" && s != "semua" {
			q += " AND i.status = ?"
			args = append(args, s)
		}
		if j := c.Query("jenis"); j == "perhiasan" || j == "logam_mulia" || j == "hasil_lebur" {
			q += " AND i.jenis = ?"
			args = append(args, j)
		}
		if s := strings.TrimSpace(c.Query("q")); s != "" {
			q += " AND (i.item_name LIKE ? OR IFNULL(i.note,'') LIKE ? OR (i.status = 'cuci' AND IFNULL(w.washer_name,'') LIKE ?))"
			args = append(args, "%"+s+"%", "%"+s+"%", "%"+s+"%")
		}
		q += " ORDER BY i.id DESC LIMIT 500"

		rows, err := db.Query(q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		out := []models.BarangListRow{}
		var totalGram, totalBeli, totalJual float64
		for rows.Next() {
			var r models.BarangListRow
			var kirim, cuciTgl, tglJual sql.NullTime
			var hargaJual sql.NullFloat64
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram,
				&r.Jenis, &r.Brand, &r.Kondisi, &r.HargaBeli,
				&r.LocationID, &r.Lokasi, &r.OriginID, &r.Asal, &r.Status, &r.EntryDate, &r.Note,
				&kirim, &r.BelumSampai, &r.BelumNote, &r.Pencuci, &cuciTgl, &r.TujuanID, &r.Tujuan,
				&hargaJual, &tglJual); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if kirim.Valid {
				t := kirim.Time
				r.KirimTgl = &t
			}
			if cuciTgl.Valid {
				t := cuciTgl.Time
				r.CuciTgl = &t
			}
			if hargaJual.Valid {
				h := hargaJual.Float64
				r.HargaJual = &h
				totalJual += h
			}
			if tglJual.Valid {
				t := tglJual.Time
				r.TanggalJual = &t
			}
			totalGram += r.WeightGram
			totalBeli += r.HargaBeli
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "jumlah": len(out), "total_gram": totalGram, "total_beli": totalBeli, "total_jual": totalJual})
	}
}
