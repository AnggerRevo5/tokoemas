package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var kategoriMasuk = map[string]map[string]bool{
	"cabang":      {"beli_customer": true, "stok_awal": true},
	"back_office": {"pabrik": true, "konsinyasi": true, "stok_awal": true},
}

func normalKarat(s string) (string, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimSpace(strings.TrimSuffix(s, "K"))
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 24 {
		return "", false
	}
	return fmt.Sprintf("%dK", n), true
}

type barangSatu struct {
	Jenis      string  `json:"jenis"`
	ItemName   string  `json:"item_name"`
	Brand      string  `json:"brand"`
	Karat      string  `json:"karat"`
	WeightGram float64 `json:"weight_gram"`
	HargaBeli  float64 `json:"harga_beli"`
	Note       string  `json:"note"`
}

type barangMasukReq struct {
	LocationID int          `json:"location_id"`
	Category   string       `json:"category"`
	EntryDate  string       `json:"entry_date"`
	Note       string       `json:"note"`
	Posisi     string       `json:"posisi"`
	Items      []barangSatu `json:"items"`
}

// ---------- INPUT BARANG MASUK ----------

func barangCreateV3(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		var req barangMasukReq
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
		tgl, ok := parseTanggal(c, req.EntryDate)
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
				k, ok := normalKarat(it.Karat)
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
			catatAudit(tx, cl.UserID, "create_auto", "cash_transactions", id, gin.H{"pembelian_customer": len(req.Items), "total": totalBeli})
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
			if _, err := tx.Exec(sqlStokItem, locID, "in", req.Category, it.ItemName, it.WeightGram, it.Karat,
				it.Note, tgl, cl.UserID, id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		catatAudit(tx, cl.UserID, "create", "items", lastID, gin.H{"jumlah_barang": len(req.Items), "request": req})

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

func barangListV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)

		q := `SELECT i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), i.weight_gram,
		             i.jenis, IFNULL(i.brand,''), i.kondisi, i.harga_beli,
		             i.location_id, l.name, i.origin_location_id, o.name, i.status, i.entry_date, IFNULL(i.note,''),
		             i.kirim_tgl, i.belum_sampai, IFNULL(i.belum_sampai_note,'')
		      FROM items i
		      JOIN locations l ON l.id = i.location_id
		      JOIN locations o ON o.id = i.origin_location_id
		      WHERE i.is_void = FALSE`
		args := []interface{}{}

		if cl.Role == "cabang" {
			q += " AND i.location_id = ?"
			args = append(args, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" && loc != "0" {
			q += " AND i.location_id = ?"
			args = append(args, loc)
		}
		if s := c.Query("status"); s != "" && s != "semua" {
			q += " AND i.status = ?"
			args = append(args, s)
		}
		if j := c.Query("jenis"); j == "perhiasan" || j == "logam_mulia" || j == "hasil_lebur" {
			q += " AND i.jenis = ?"
			args = append(args, j)
		}
		if s := strings.TrimSpace(c.Query("q")); s != "" {
			q += " AND (i.item_name LIKE ? OR IFNULL(i.note,'') LIKE ?)"
			args = append(args, "%"+s+"%", "%"+s+"%")
		}
		q += " ORDER BY i.id DESC LIMIT 500"

		rows, err := db.Query(q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type row struct {
			ID          int        `json:"id"`
			ItemNo      string     `json:"item_no"`
			ItemName    string     `json:"item_name"`
			Karat       string     `json:"karat"`
			WeightGram  float64    `json:"weight_gram"`
			Jenis       string     `json:"jenis"`
			Brand       string     `json:"brand"`
			Kondisi     string     `json:"kondisi"`
			HargaBeli   float64    `json:"harga_beli"`
			LocationID  int        `json:"location_id"`
			Lokasi      string     `json:"lokasi"`
			OriginID    int        `json:"origin_location_id"`
			Asal        string     `json:"asal"`
			Status      string     `json:"status"`
			EntryDate   time.Time  `json:"entry_date"`
			Note        string     `json:"note"`
			KirimTgl    *time.Time `json:"kirim_tgl"`
			BelumSampai bool       `json:"belum_sampai"`
			BelumNote   string     `json:"belum_sampai_note"`
		}
		out := []row{}
		var totalGram, totalBeli float64
		for rows.Next() {
			var r row
			var kirim sql.NullTime
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram,
				&r.Jenis, &r.Brand, &r.Kondisi, &r.HargaBeli,
				&r.LocationID, &r.Lokasi, &r.OriginID, &r.Asal, &r.Status, &r.EntryDate, &r.Note,
				&kirim, &r.BelumSampai, &r.BelumNote); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if kirim.Valid {
				t := kirim.Time
				r.KirimTgl = &t
			}
			totalGram += r.WeightGram
			totalBeli += r.HargaBeli
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "jumlah": len(out), "total_gram": totalGram, "total_beli": totalBeli})
	}
}

// ---------- BATALKAN BARANG MASUK ----------

func barangVoidV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		reason, ok := bacaAlasan(c)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			gagal(c, err)
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
		catatAudit(tx, cl.UserID, "void", "items", int64(id), gin.H{"reason": reason})

		if cashID.Valid && harga > 0 {
			var amt float64
			if err := tx.QueryRow("SELECT amount FROM cash_transactions WHERE id = ? FOR UPDATE", cashID.Int64).Scan(&amt); err == nil {
				baru := amt - harga
				if baru < 0.005 {
					tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, cashID.Int64)
				} else {
					tx.Exec("UPDATE cash_transactions SET amount = ? WHERE id = ?", baru, cashID.Int64)
				}
				catatAudit(tx, cl.UserID, "koreksi_kas", "cash_transactions", cashID.Int64,
					gin.H{"dari": amt, "menjadi": baru, "alasan": reason})
			}
		}

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang dibatalkan"})
	}
}