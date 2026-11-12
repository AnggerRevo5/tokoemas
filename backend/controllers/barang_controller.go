package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

func BarangJual(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// penjualan barang stok oleh toko (barang di toko sendiri) atau owner/admin; PJ menjual lewat menu Lebur
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		var req models.JualReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.Items) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
		// harga jual wajib diisi per barang (manual); 0 = gratis, dicek per jenis setelah barang dikunci
		harga := map[int]float64{}
		idsJual := make([]int, 0, len(req.Items))
		for _, it := range req.Items {
			if it.Harga < 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "harga jual tidak boleh negatif"})
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
		metode, okMetode := services.NormalMetode(req.Metode)
		if !okMetode {
			c.JSON(http.StatusBadRequest, gin.H{"error": "metode pembayaran tidak dikenal"})
			return
		}
		var totalHarga float64
		for _, h := range harga {
			totalHarga += h
		}
		if err := services.ValidasiBiaya(metode, req.Biaya, totalHarga); err != nil {
			balasKesalahan(c, err)
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
			balasKesalahan(c, err)
			return
		}
		for _, it := range items {
			if harga[it.ID] == 0 && !services.BolehGratis(it.Jenis) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("harga jual %s wajib diisi (lebih dari 0)", it.Name)})
				return
			}
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
			// tunai dan transfer: kas lokasi yang menjual. QRIS/Debit: kas back office; untuk toko cabang diberi kategori
			// tersendiri dan asal cabangnya dicatat agar omzet cabang tetap bisa dilacak.
			kasLokasi, kategoriKas := lokasi, "omzet"
			if lokasi == boID {
				kategoriKas = "jual_barang"
			}
			catatanKas := fmt.Sprintf("Penjualan %d barang", len(kj.IDs))
			var asal sql.NullInt64
			if metode == services.MetodeQRIS {
				catatanKas += " (" + services.LabelMetode(metode) + ")"
				if lokasi != boID {
					var namaAsal string
					tx.QueryRow("SELECT name FROM locations WHERE id = ?", lokasi).Scan(&namaAsal)
					kasLokasi, kategoriKas = boID, services.KategoriJualQRIS
					asal = sql.NullInt64{Int64: int64(lokasi), Valid: true}
					catatanKas += " dari " + namaAsal
				}
			} else if metode == services.MetodeTransfer {
				catatanKas += " (" + services.LabelMetode(metode) + ")"
			}
			if req.Note != "" {
				catatanKas += ": " + req.Note
			}
			res, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by, metode, asal_location_id)
				 VALUES (?, 'in', ?, ?, ?, ?, ?, ?, ?)`,
				kasLokasi, kategoriKas, kj.Total, catatanKas, tgl, cl.UserID, metode, asal)
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
			services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", kasID, gin.H{"penjualan": len(kj.IDs), "total": kj.Total, "metode": metode})
		}
		if req.Biaya > 0 {
			catatBiaya := "Potongan " + services.LabelMetode(metode) + fmt.Sprintf(" penjualan %d barang", len(items))
			res, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by, metode)
				 VALUES (?, 'out', ?, ?, ?, ?, ?, ?)`,
				boID, services.KategoriBiayaQRIS, req.Biaya, catatBiaya, tgl, cl.UserID, metode)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			biayaID, _ := res.LastInsertId()
			for _, itemID := range idsJual {
				if _, err := tx.Exec("UPDATE items SET jual_biaya_cash_id = ? WHERE id = ?", biayaID, itemID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", biayaID, gin.H{"biaya": req.Biaya, "metode": metode})
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "jual", "items", 0, keadaanBarang(items),
			gin.H{"status": "terjual", "harga_jual": harga, "request": req})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		if metode == services.MetodeQRIS {
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang terjual (%s), kas back office bertambah %s", len(items), services.LabelMetode(metode), utils.RupiahTeks(totalSemua-req.Biaya))})
			return
		}
		if metode == services.MetodeTransfer {
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang terjual (%s), kas toko bertambah %s", len(items), services.LabelMetode(metode), utils.RupiahTeks(totalSemua))})
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
		totalBarang := 0
		for i := range req.Items {
			it := &req.Items[i]
			baris := i + 1
			it.Note = strings.TrimSpace(it.Note)
			if it.Note == "" {
				it.Note = catatanUmum
			}
			it.Brand = strings.TrimSpace(it.Brand)
			it.ItemName = strings.TrimSpace(it.ItemName)
			bahanReq := strings.TrimSpace(it.Bahan)
			it.Bahan = "emas"
			if strings.EqualFold(bahanReq, "perak") {
				it.Bahan = "perak"
			}

			if it.Jumlah == 0 {
				it.Jumlah = 1
			}
			if it.Jumlah > 1 && !services.TanpaGram(it.Jenis) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: jumlah hanya untuk susuk, tempat, dan kotak perhiasan", baris)})
				return
			}
			totalBarang += it.Jumlah
			// berat susuk, tempat, dan kotak perhiasan opsional
			if it.WeightGram <= 0 && !services.TanpaGram(it.Jenis) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: berat wajib diisi", baris)})
				return
			}
			switch it.Jenis {
			case "susuk", "tempat_perhiasan", "kotak_perhiasan":
				if it.ItemName == "" {
					it.ItemName = services.NamaBawaan(it.Jenis)
				}
				it.Brand = ""
				// kadar opsional, hanya untuk susuk
				if it.Jenis == "susuk" && strings.TrimSpace(it.Karat) != "" {
					k, ok := utils.NormalKarat(it.Karat)
					if !ok {
						c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: kadar harus 1K sampai 24K", baris)})
						return
					}
					it.Karat = k
				} else {
					it.Karat = ""
				}
			case "logam_mulia":
				if it.Brand == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: merek logam mulia wajib diisi", baris)})
					return
				}
				it.Karat = "24K"
				it.ItemName = "Logam mulia " + it.Brand
				if it.Bahan == "perak" {
					it.Karat = "999"
					it.ItemName = "Logam mulia perak " + it.Brand
				}
			case "perhiasan":
				if it.ItemName == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: nama perhiasan wajib diisi", baris)})
					return
				}
				if it.Bahan == "perak" {
					it.Karat = "925" // perhiasan perak hanya berkadar 925
				} else {
					k, ok := utils.NormalKarat(it.Karat)
					if !ok {
						c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: kadar harus 1K sampai 24K", baris)})
						return
					}
					it.Karat = k
				}
				it.Brand = ""
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: jenis barang tidak valid", baris)})
				return
			}
			if !services.BolehBaki(it.Jenis) {
				it.TrayID = 0 // baki untuk perhiasan, logam mulia (emas atau perak), dan susuk
			}
			if it.Jenis != "logam_mulia" && it.Jenis != "perhiasan" {
				it.Bahan = "emas"
			}
			if it.HargaBeli < 0 || (beliCustomer && it.HargaBeli <= 0) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: harga beli wajib diisi", baris)})
				return
			}
			totalBeli += it.HargaBeli * float64(it.Jumlah)
		}
		if totalBarang > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali input"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// baki harus ada dan milik lokasi barang; isi satu baki tidak dibatasi
		cekBaki := map[int]bool{}
		for _, it := range req.Items {
			if it.TrayID > 0 && !cekBaki[it.TrayID] {
				cekBaki[it.TrayID] = true
				if _, err := services.IsiBaki(tx, it.TrayID, locID); err != nil {
					balasKesalahan(c, err)
					return
				}
			}
		}

		var cashID interface{}
		kasKeluar := 0.0
		if beliCustomer {
			kasKeluar = totalBeli
			note := fmt.Sprintf("Beli dari customer: %d barang", totalBarang)
			if catatanUmum != "" {
				note += " - " + catatanUmum
			}
			res, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
				 VALUES (?,?,?,?,?,?,?)`,
				locID, "out", "pembelian_customer", totalBeli, note, tgl, cl.UserID,
			)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			id, _ := res.LastInsertId()
			cashID = id
			services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", id, gin.H{"pembelian_customer": totalBarang, "total": totalBeli})
		}

		var lastID int64
		for _, it := range req.Items {
			for n := 0; n < it.Jumlah; n++ {
				res, err := tx.Exec(
					`INSERT INTO items
					 (item_name, karat, weight_gram, origin_location_id, location_id, source_category, entry_date, note, created_by,
					  jenis, brand, kondisi, harga_beli, purchase_cash_id, tray_id, bahan)
					 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
					it.ItemName, it.Karat, it.WeightGram, locID, locID, req.Category, tgl, it.Note, cl.UserID,
					it.Jenis, sql.NullString{String: it.Brand, Valid: it.Brand != ""}, kondisi, it.HargaBeli, cashID,
					sql.NullInt64{Int64: int64(it.TrayID), Valid: it.TrayID > 0}, it.Bahan,
				)
				if err != nil {
					utils.Gagal(c, err)
					return
				}
				id, _ := res.LastInsertId()
				lastID = id
				tx.Exec("UPDATE items SET item_no = ? WHERE id = ?", fmt.Sprintf("%s-%05d", services.AwalanKode(it.Jenis), id), id)
				if kirimKeBO {
					tx.Exec("UPDATE items SET status = 'dikirim', kirim_tgl = ? WHERE id = ?", tgl, id)
				}
				if _, err := tx.Exec(services.SQLStokItem, locID, "in", req.Category, it.ItemName, it.WeightGram, it.Karat,
					it.Note, tgl, cl.UserID, id); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
		}
		services.CatatAudit(tx, middleware.Meta(c), "create", "items", lastID, gin.H{"jumlah_barang": totalBarang, "request": req})

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		pesan := fmt.Sprintf("%d barang tersimpan", totalBarang)
		if kirimKeBO {
			pesan += ", diantar ke back office (menunggu konfirmasi)"
		}
		c.JSON(http.StatusCreated, gin.H{"message": pesan, "jumlah": totalBarang, "kas_keluar": kasKeluar})
	}
}

// ---------- DAFTAR BARANG ----------

func BarangVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := middleware.HanyaSuperAdmin(c)
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
		services.CatatAudit(tx, middleware.Meta(c), "void", "items", int64(id), gin.H{"reason": reason})

		if cashID.Valid && harga > 0 {
			var amt float64
			if err := tx.QueryRow("SELECT amount FROM cash_transactions WHERE id = ? FOR UPDATE", cashID.Int64).Scan(&amt); err == nil {
				baru := amt - harga
				if baru < 0.005 {
					tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, cashID.Int64)
				} else {
					tx.Exec("UPDATE cash_transactions SET amount = ? WHERE id = ?", baru, cashID.Int64)
				}
				services.CatatAudit(tx, middleware.Meta(c), "koreksi_kas", "cash_transactions", cashID.Int64,
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

		kolom := `SELECT i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), i.weight_gram,
		             i.jenis, IFNULL(i.brand,''), i.kondisi, i.harga_beli,
		             i.location_id, l.name, i.origin_location_id, o.name, i.status, i.entry_date, IFNULL(i.note,''),
		             i.kirim_tgl, i.belum_sampai, IFNULL(i.belum_sampai_note,''),
		             IF(i.status = 'cuci', IFNULL(w.washer_name,''), ''),
		             IF(i.status = 'cuci', w.start_date, NULL),
		             IFNULL(i.kirim_ke_location_id,0), IFNULL(k.name,''),
		             i.harga_jual, i.tanggal_jual, IFNULL(t.id,0), IFNULL(t.no,0), IFNULL(t.name,''), i.purchase_cash_id IS NOT NULL,
		             IFNULL(jc.metode,''), i.bahan`
		dari := `FROM items i
		      JOIN locations l ON l.id = i.location_id
		      JOIN locations o ON o.id = i.origin_location_id
		      LEFT JOIN locations k ON k.id = i.kirim_ke_location_id
		      LEFT JOIN washing_jobs w ON w.id = i.washing_job_id
		      LEFT JOIN trays t ON t.id = i.tray_id AND t.location_id = i.location_id
		      LEFT JOIN cash_transactions jc ON jc.id = i.jual_cash_id
		      WHERE i.is_void = FALSE`
		args := []interface{}{}

		if cl.Role == "cabang" {
			dari += " AND (i.location_id = ? OR i.kirim_ke_location_id = ?)"
			args = append(args, cl.LocationID, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" && loc != "0" {
			dari += " AND i.location_id = ?"
			args = append(args, loc)
		}
		if s := c.Query("status"); s == "dikirim" {
			dari += " AND i.status IN ('dikirim','kembali')"
		} else if s != "" && s != "semua" {
			dari += " AND i.status = ?"
			args = append(args, s)
		}
		switch j := c.Query("jenis"); j {
		case "hasil_lebur", "susuk", "tempat_perhiasan", "kotak_perhiasan":
			dari += " AND i.jenis = ?"
			args = append(args, j)
		case "perhiasan": // perhiasan emas
			dari += " AND i.jenis = 'perhiasan' AND i.bahan = 'emas'"
		case "perhiasan_perak":
			dari += " AND i.jenis = 'perhiasan' AND i.bahan = 'perak'"
		case "logam_mulia": // logam mulia emas
			dari += " AND i.jenis = 'logam_mulia' AND i.bahan = 'emas'"
		case "logam_mulia_perak":
			dari += " AND i.jenis = 'logam_mulia' AND i.bahan = 'perak'"
		}
		if v := c.Query("item_id"); v != "" {
			dari += " AND i.id = ?"
			args = append(args, v)
		}
		// barang yang membentuk satu catatan kas penjualan (dipakai menu Kas untuk mengubah penjualan)
		if k := c.Query("kas_id"); k != "" {
			dari += " AND (i.jual_cash_id = ? OR i.jual_biaya_cash_id = ?)"
			args = append(args, k, k)
		}
		if b := c.Query("baki_id"); b != "" && b != "0" {
			dari += " AND i.tray_id = ?"
			args = append(args, b)
		}
		if s := strings.TrimSpace(c.Query("q")); s != "" {
			dari += " AND (i.item_name LIKE ? OR IFNULL(i.note,'') LIKE ? OR (i.status = 'cuci' AND IFNULL(w.washer_name,'') LIKE ?))"
			args = append(args, "%"+s+"%", "%"+s+"%", "%"+s+"%")
		}

		// jumlah barang dan total berat/harga dihitung dari seluruh barang yang cocok, bukan hanya baris yang tampil
		var jumlah int
		var totalGram, totalPerak, totalBeli, totalJual float64
		if err := db.QueryRow(
			"SELECT COUNT(*), IFNULL(SUM(IF(i.bahan = 'perak' OR i.jenis IN ("+services.SQLJenisTanpaGram+"), 0, i.weight_gram)),0),"+
				" IFNULL(SUM(IF(i.bahan = 'perak', i.weight_gram, 0)),0),"+
				" IFNULL(SUM(i.harga_beli),0), IFNULL(SUM(i.harga_jual),0) "+dari, args...,
		).Scan(&jumlah, &totalGram, &totalPerak, &totalBeli, &totalJual); err != nil {
			utils.Gagal(c, err)
			return
		}
		// halaman: 200 barang per muat, ?limit= untuk memuat lebih banyak (maksimal 2000)
		batas := 200
		if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
			batas = v
		}
		if batas > 2000 {
			batas = 2000
		}

		rows, err := db.Query(kolom+" "+dari+" ORDER BY i.id DESC LIMIT ?", append(args, batas)...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.BarangListRow{}
		for rows.Next() {
			var r models.BarangListRow
			var kirim, cuciTgl, tglJual sql.NullTime
			var hargaJual sql.NullFloat64
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram,
				&r.Jenis, &r.Brand, &r.Kondisi, &r.HargaBeli,
				&r.LocationID, &r.Lokasi, &r.OriginID, &r.Asal, &r.Status, &r.EntryDate, &r.Note,
				&kirim, &r.BelumSampai, &r.BelumNote, &r.Pencuci, &cuciTgl, &r.TujuanID, &r.Tujuan,
				&hargaJual, &tglJual, &r.BakiID, &r.BakiNo, &r.BakiNama, &r.DariCustomer, &r.Metode, &r.Bahan); err != nil {
				utils.Gagal(c, err)
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
			}
			if tglJual.Valid {
				t := tglJual.Time
				r.TanggalJual = &t
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "jumlah": jumlah, "ditampilkan": len(out), "ada_lagi": len(out) < jumlah, "batas": batas,
			"total_gram": totalGram, "total_gram_perak": totalPerak, "total_beli": totalBeli, "total_jual": totalJual})
	}
}
