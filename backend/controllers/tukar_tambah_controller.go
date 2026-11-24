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

const kategoriTukarTambah = "tukar_tambah"

// siapkanBarangLama merapikan satu barang lama (hanya perhiasan dan logam mulia, emas atau perak) dan
// membalas 400 bila tidak valid. baris = nomor baris untuk pesan.
func siapkanBarangLama(c *gin.Context, it *models.TukarLama, baris int) bool {
	tolak := func(pesan string) bool {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("barang lama baris %d: %s", baris, pesan)})
		return false
	}
	it.ItemName, it.Brand, it.Note = strings.TrimSpace(it.ItemName), strings.TrimSpace(it.Brand), strings.TrimSpace(it.Note)
	if strings.EqualFold(strings.TrimSpace(it.Bahan), "perak") {
		it.Bahan = "perak"
	} else {
		it.Bahan = "emas"
	}
	if it.Kondisi != "bersih" {
		it.Kondisi = "kotor"
	}
	switch it.Jenis {
	case "logam_mulia":
		if it.Brand == "" {
			return tolak("merek logam mulia wajib diisi")
		}
		it.Karat, it.ItemName = "24K", "Logam mulia "+it.Brand
		if it.Bahan == "perak" {
			it.Karat, it.ItemName = "999", "Logam mulia perak "+it.Brand
		}
	case "perhiasan":
		if it.ItemName == "" {
			return tolak("nama perhiasan wajib diisi")
		}
		it.Brand = ""
		if it.Bahan == "perak" {
			it.Karat = "925"
		} else if k, ok := utils.NormalKarat(it.Karat); ok {
			it.Karat = k
		} else {
			return tolak("kadar harus 1K sampai 24K")
		}
	default:
		return tolak("jenis harus perhiasan atau logam mulia")
	}
	return true
}

// ---------- SIMPAN TUKAR TAMBAH ----------

func TukarTambahCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req models.TukarTambahReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if tipe != "cabang" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tukar tambah hanya di toko cabang"})
			return
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
		nilai := make([]float64, len(req.Lama))
		for i := range req.Lama {
			if !siapkanBarangLama(c, &req.Lama[i], i+1) {
				return
			}
			nilai[i] = req.Lama[i].Nilai
		}
		harga := map[int]float64{}
		ids := make([]int, 0, len(req.Baru))
		hargaBaru := make([]float64, 0, len(req.Baru))
		for _, b := range req.Baru {
			if _, dobel := harga[b.ItemID]; dobel {
				c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang baru yang dipilih dua kali"})
				return
			}
			harga[b.ItemID] = b.Harga
			ids = append(ids, b.ItemID)
			hargaBaru = append(hargaBaru, b.Harga)
		}
		selisih, err := services.HitungTukarTambah(nilai, hargaBaru, metode, req.Biaya)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		var totalLama, totalBaru float64
		for _, n := range nilai {
			totalLama += n
		}
		for _, h := range hargaBaru {
			totalBaru += h
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		baru, err := services.AmbilBarang(tx, ids)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		for _, it := range baru {
			switch {
			case it.Status != "tersedia":
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sedang tidak tersedia (%s)", it.Name, it.Status)})
				return
			case it.Location != locID:
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s bukan barang toko ini", it.Name)})
				return
			case harga[it.ID] == 0 && !services.BolehGratis(it.Jenis):
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("harga %s wajib diisi (lebih dari 0)", it.Name)})
				return
			}
		}
		cekBaki := map[int]bool{}
		for _, l := range req.Lama {
			if l.TrayID > 0 && !cekBaki[l.TrayID] {
				cekBaki[l.TrayID] = true
				if _, err := services.IsiBaki(tx, l.TrayID, locID); err != nil {
					balasKesalahan(c, err)
					return
				}
			}
		}

		catatan := "Tukar tambah"
		if n := strings.TrimSpace(req.Note); n != "" {
			catatan += ": " + n
		}
		res, err := tx.Exec(
			`INSERT INTO tukar_tambah (location_id, total_lama, total_baru, selisih, metode, biaya, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			locID, totalLama, totalBaru, selisih, metode, req.Biaya, strings.TrimSpace(req.Note), tgl, cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		ttID, _ := res.LastInsertId()

		for _, l := range req.Lama {
			note := catatan
			if l.Note != "" {
				note = l.Note
			}
			r, err := tx.Exec(
				`INSERT INTO items (item_name, karat, weight_gram, origin_location_id, location_id, source_category, entry_date, note,
				  created_by, jenis, brand, kondisi, harga_beli, tray_id, bahan, tukar_tambah_id)
				 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				l.ItemName, l.Karat, l.WeightGram, locID, locID, kategoriTukarTambah, tgl, note, cl.UserID,
				l.Jenis, sql.NullString{String: l.Brand, Valid: l.Brand != ""}, l.Kondisi, l.Nilai,
				sql.NullInt64{Int64: int64(l.TrayID), Valid: l.TrayID > 0}, l.Bahan, ttID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			id, _ := r.LastInsertId()
			if _, err := tx.Exec("UPDATE items SET item_no = ? WHERE id = ?", fmt.Sprintf("%s-%05d", services.AwalanKode(l.Jenis), id), id); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec(services.SQLStokItem, locID, "in", kategoriTukarTambah, l.ItemName, l.WeightGram, l.Karat, note, tgl, cl.UserID, id); err != nil {
				utils.Gagal(c, err)
				return
			}
		}

		// jual_cash_id sengaja tidak diisi: pembatalan per barang lewat menu jual harus ditolak, batal hanya lewat tukar tambah.
		for _, it := range baru {
			if _, err := tx.Exec(services.SQLStokItem, locID, "out", "laku", it.Name, it.Weight, it.Karat, catatan, tgl, cl.UserID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec(
				"UPDATE items SET status = 'terjual', harga_jual = ?, tanggal_jual = ?, tukar_tambah_id = ? WHERE id = ?",
				harga[it.ID], tgl, ttID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}

		if selisih > 0.005 || selisih < -0.005 {
			// dua kategori kas supaya laporan laba rugi (yang mengelompokkan per kategori) tahu arahnya
			arah, kat, jumlah := "in", "tukar_tambah_masuk", selisih
			if selisih < 0 {
				arah, kat, jumlah = "out", "tukar_tambah_keluar", -selisih
			}
			ket := catatan + " (" + services.LabelMetode(metode) + ")"
			r, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by, metode)
				 VALUES (?,?,?,?,?,?,?,?)`, locID, arah, kat, jumlah, ket, tgl, cl.UserID, metode)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			kasID, _ := r.LastInsertId()
			if _, err := tx.Exec("UPDATE tukar_tambah SET kas_id = ? WHERE id = ?", kasID, ttID); err != nil {
				utils.Gagal(c, err)
				return
			}
			services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", kasID, gin.H{"tukar_tambah": ttID, "selisih": selisih, "metode": metode})
		}
		if req.Biaya > 0 {
			r, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by, metode)
				 VALUES (?,?,?,?,?,?,?,?)`, locID, "out", services.KategoriBiayaQRIS, req.Biaya,
				"Potongan "+services.LabelMetode(metode)+" tukar tambah", tgl, cl.UserID, metode)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			biayaID, _ := r.LastInsertId()
			if _, err := tx.Exec("UPDATE tukar_tambah SET biaya_kas_id = ? WHERE id = ?", biayaID, ttID); err != nil {
				utils.Gagal(c, err)
				return
			}
			services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", biayaID, gin.H{"tukar_tambah": ttID, "biaya": req.Biaya})
		}
		services.CatatAudit(tx, middleware.Meta(c), "create", "tukar_tambah", ttID, gin.H{"lama": len(req.Lama), "baru": len(req.Baru), "selisih": selisih})

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		pesan := "Tukar tambah tersimpan, tidak ada selisih kas"
		switch {
		case selisih > 0.005:
			pesan = fmt.Sprintf("Tukar tambah tersimpan, kas toko bertambah %s", utils.RupiahTeks(selisih))
		case selisih < -0.005:
			pesan = fmt.Sprintf("Tukar tambah tersimpan, kas toko berkurang %s", utils.RupiahTeks(-selisih))
		}
		c.JSON(http.StatusCreated, gin.H{"message": pesan, "id": ttID, "selisih": selisih})
	}
}

// ---------- BATALKAN TUKAR TAMBAH ----------

func TukarTambahBatal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaSuperAdmin(c); !ok {
			return
		}
		cl := c.MustGet("claims").(*models.Claims)
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		alasan, ok := utils.BacaAlasan(c)
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var lokasi int
		var batal, baru bool
		var kasID, biayaID sql.NullInt64
		err = tx.QueryRow(
			`SELECT location_id, is_void, transaction_date >= DATE_SUB(CURDATE(), INTERVAL 7 DAY), kas_id, biaya_kas_id
			 FROM tukar_tambah WHERE id = ? FOR UPDATE`, id).Scan(&lokasi, &batal, &baru, &kasID, &biayaID)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "tukar tambah tidak ditemukan"})
			return
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if batal {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tukar tambah ini sudah dibatalkan"})
			return
		}
		if !bolehUbahTransaksi(cl, lokasi, baru) {
			c.JSON(http.StatusForbidden, gin.H{"error": "hanya Super Admin yang bisa membatalkan tukar tambah yang sudah tercatat"})
			return
		}

		// barang lama harus masih utuh di toko; barang baru harus masih berstatus terjual
		rows, err := tx.Query(
			`SELECT id, item_name, status, location_id, source_category = 'tukar_tambah'
			 FROM items WHERE tukar_tambah_id = ? AND is_void = FALSE FOR UPDATE`, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		var lamaIDs, baruIDs []int
		for rows.Next() {
			var iid, loc int
			var nama, status string
			var lama bool
			if err := rows.Scan(&iid, &nama, &status, &loc, &lama); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			switch {
			case lama && (status != "tersedia" || loc != lokasi):
				rows.Close()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s (barang lama) sudah diproses (%s), tukar tambah tidak bisa dibatalkan", nama, status)})
				return
			case !lama && status != "terjual":
				rows.Close()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s (barang baru) berstatus %s, tukar tambah tidak bisa dibatalkan", nama, status)})
				return
			case lama:
				lamaIDs = append(lamaIDs, iid)
			default:
				baruIDs = append(baruIDs, iid)
			}
		}
		rows.Close()

		for _, iid := range lamaIDs {
			if _, err := tx.Exec("UPDATE items SET is_void = TRUE, void_reason = ? WHERE id = ?", alasan, iid); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec(
				"UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE item_id = ? AND direction = 'in' AND category = ? AND is_void = FALSE",
				alasan, iid, kategoriTukarTambah); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		for _, iid := range baruIDs {
			if _, err := tx.Exec(
				`UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE id =
				 (SELECT id FROM (SELECT MAX(id) AS id FROM stock_movements WHERE item_id = ? AND direction = 'out' AND category = 'laku' AND is_void = FALSE) x)`,
				alasan, iid); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec(
				"UPDATE items SET status = 'tersedia', harga_jual = NULL, tanggal_jual = NULL, tukar_tambah_id = NULL WHERE id = ?", iid); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		for _, k := range []sql.NullInt64{kasID, biayaID} {
			if !k.Valid {
				continue
			}
			if _, err := tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", alasan, k.Int64); err != nil {
				utils.Gagal(c, err)
				return
			}
			services.CatatAudit(tx, middleware.Meta(c), "koreksi_kas", "cash_transactions", k.Int64, gin.H{"batal_tukar_tambah": id, "alasan": alasan})
		}
		if _, err := tx.Exec("UPDATE tukar_tambah SET is_void = TRUE, void_reason = ? WHERE id = ?", alasan, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "void", "tukar_tambah", int64(id), gin.H{"alasan": alasan})
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "tukar tambah dibatalkan, stok dan kas dikembalikan"})
	}
}

// ---------- DAFTAR TUKAR TAMBAH (7 HARI TERAKHIR) ----------

func TukarTambahList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if cl.Role != "cabang" && cl.Role != "owner" && cl.Role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
			return
		}
		where, args := "t.is_void = FALSE AND t.transaction_date >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)", []interface{}{}
		if cl.Role == "cabang" {
			where += " AND t.location_id = ?"
			args = append(args, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" && loc != "0" {
			where += " AND t.location_id = ?"
			args = append(args, loc)
		}
		rows, err := db.Query(
			`SELECT t.id, t.location_id, l.name, DATE_FORMAT(t.transaction_date, '%Y-%m-%d'), t.total_lama, t.total_baru, t.selisih,
			        t.metode, t.biaya, IFNULL(t.note,''),
			        IFNULL((SELECT GROUP_CONCAT(CONCAT(i.item_name, ' ', IFNULL(i.karat,''), ' ', i.weight_gram, 'g') SEPARATOR '; ')
			                FROM items i WHERE i.tukar_tambah_id = t.id AND i.source_category = 'tukar_tambah'), ''),
			        IFNULL((SELECT GROUP_CONCAT(CONCAT(i.item_name, ' ', IFNULL(i.karat,''), ' ', i.weight_gram, 'g') SEPARATOR '; ')
			                FROM items i WHERE i.tukar_tambah_id = t.id AND i.source_category <> 'tukar_tambah'), '')
			 FROM tukar_tambah t JOIN locations l ON l.id = t.location_id
			 WHERE `+where+` ORDER BY t.id DESC LIMIT 200`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var id, loc int
			var nama, tgl, metode, note, lama, baru string
			var tl, tb, sel, biaya float64
			if err := rows.Scan(&id, &loc, &nama, &tgl, &tl, &tb, &sel, &metode, &biaya, &note, &lama, &baru); err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, gin.H{
				"id": id, "location_id": loc, "location_name": nama, "tanggal": tgl,
				"total_lama": tl, "total_baru": tb, "selisih": sel, "metode": metode, "biaya": biaya, "note": note,
				"barang_lama": lama, "barang_baru": baru, "bisa_batal": bolehUbahTransaksi(cl, loc, true),
			})
		}
		c.JSON(http.StatusOK, out)
	}
}
