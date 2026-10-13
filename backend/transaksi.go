package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// kunci: tipeLokasi|tabel|arah
var kategoriValid = map[string]map[string]bool{
	"cabang|kas|in":  {"omzet": true, "tambah_kas": true},
	"cabang|kas|out": {"pengeluaran_toko": true, "pembelian_customer": true, "transfer_bo": true},
	"cabang|stok|in": {"barang_masuk": true},
	"cabang|stok|out": {"laku": true, "pegadaian": true, "cuci": true, "lebur": true, "jual": true},

	"back_office|kas|in":  {"dari_toko": true, "jual_barang": true, "investor": true, "pinjaman": true, "jual_aset": true, "kembali_cabang": true},
	"back_office|kas|out": {"kas_toko": true, "bayar_hutang": true, "bulanan_investor": true, "operasional": true, "kembali_cabang": true},
	"back_office|stok|in":  {"pabrik": true, "konsinyasi": true, "cucian": true, "barang_toko": true, "kembali_cabang": true},
	"back_office|stok|out": {"ke_toko": true, "lebur": true, "jual": true, "cuci": true},
}

func resolveLocation(c *gin.Context, db *sql.DB, requested int) (int, string, bool) {
	cl := c.MustGet("claims").(*Claims)
	id := requested
	if cl.Role == "cabang" {
		id = cl.LocationID
	} else if cl.Role != "super_admin" && cl.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
		return 0, "", false
	}
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "location_id wajib diisi"})
		return 0, "", false
	}
	var tipe string
	if err := db.QueryRow("SELECT type FROM locations WHERE id = ?", id).Scan(&tipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lokasi tidak ditemukan"})
		return 0, "", false
	}
	return id, tipe, true
}

func parseTanggal(c *gin.Context, s string) (string, bool) {
	if s == "" {
		return time.Now().Format("2006-01-02"), true
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format tanggal harus YYYY-MM-DD"})
		return "", false
	}
	return s, true
}

func catatAudit(tx *sql.Tx, userID int, action, tabel string, recordID int64, data interface{}) error {
	j, _ := json.Marshal(data)
	_, err := tx.Exec(
		"INSERT INTO audit_logs (user_id, action, table_name, record_id, new_value) VALUES (?,?,?,?,?)",
		userID, action, tabel, recordID, string(j),
	)
	return err
}

// ---------- KAS ----------

type kasReq struct {
	LocationID int     `json:"location_id"`
	Direction  string  `json:"direction"`
	Category   string  `json:"category"`
	Amount     float64 `json:"amount"`
	Note       string  `json:"note"`
	Date       string  `json:"transaction_date"`
}

func kasCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		var req kasReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if !kategoriValid[tipe+"|kas|"+req.Direction][req.Category] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "arah atau kategori tidak valid untuk lokasi ini"})
			return
		}
				if req.Category == "bayar_hutang" || req.Category == "investor" || req.Category == "pinjaman" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dana dari investor atau pinjaman dan pembayaran hutang diinput lewat menu Hutang"})
			return
		}
		if req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nominal harus lebih dari 0"})
			return
		}
		tgl, ok := parseTanggal(c, req.Date)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			locID, req.Direction, req.Category, req.Amount, req.Note, tgl, cl.UserID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		catatAudit(tx, cl.UserID, "create", "cash_transactions", id, req)

		// Otomatis: transfer kas cabang -> pemasukan "dari_toko" di back office
		if req.Category == "transfer_bo" {
			var boID int
			if err := tx.QueryRow("SELECT id FROM locations WHERE type='back_office' LIMIT 1").Scan(&boID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
				return
			}
			res2, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, related_id, created_by)
				 VALUES (?,?,?,?,?,?,?,?)`,
				boID, "in", "dari_toko", req.Amount, req.Note, tgl, id, cl.UserID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			id2, _ := res2.LastInsertId()
			tx.Exec("UPDATE cash_transactions SET related_id = ? WHERE id = ?", id2, id)
			catatAudit(tx, cl.UserID, "create_auto", "cash_transactions", id2, gin.H{"dari_transfer_id": id})
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "kas tersimpan"})
	}
}

func kasList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		q := "SELECT id, location_id, direction, category, amount, IFNULL(note,''), transaction_date, created_at FROM cash_transactions WHERE is_void = FALSE"
		args := []interface{}{}

		if cl.Role == "cabang" {
			q += " AND location_id = ?"
			args = append(args, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" {
			q += " AND location_id = ?"
			args = append(args, loc)
		}
		if v := c.Query("from"); v != "" {
			q += " AND transaction_date >= ?"
			args = append(args, v)
		}
		if v := c.Query("to"); v != "" {
			q += " AND transaction_date <= ?"
			args = append(args, v)
		}
		q += " ORDER BY transaction_date DESC, id DESC LIMIT 200"

		rows, err := db.Query(q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type row struct {
			ID         int       `json:"id"`
			LocationID int       `json:"location_id"`
			Direction  string    `json:"direction"`
			Category   string    `json:"category"`
			Amount     float64   `json:"amount"`
			Note       string    `json:"note"`
			Date       time.Time `json:"transaction_date"`
			CreatedAt  time.Time `json:"created_at"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Direction, &r.Category, &r.Amount, &r.Note, &r.Date, &r.CreatedAt); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- STOK ----------

type stokReq struct {
	LocationID int     `json:"location_id"`
	Direction  string  `json:"direction"`
	Category   string  `json:"category"`
	ItemName   string  `json:"item_name"`
	WeightGram float64 `json:"weight_gram"`
	Karat      string  `json:"karat"`
	Qty        int     `json:"qty"`
	Note       string  `json:"note"`
	Date       string  `json:"transaction_date"`
}

func stokCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		var req stokReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if !kategoriValid[tipe+"|stok|"+req.Direction][req.Category] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "arah atau kategori tidak valid untuk lokasi ini"})
			return
		}
		if req.ItemName == "" || req.WeightGram <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama barang dan berat (gram) wajib diisi"})
			return
		}
		if req.Qty < 1 {
			req.Qty = 1
		}
		tgl, ok := parseTanggal(c, req.Date)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(
			`INSERT INTO stock_movements (location_id, direction, category, item_name, weight_gram, karat, qty, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?,?)`,
			locID, req.Direction, req.Category, req.ItemName, req.WeightGram, req.Karat, req.Qty, req.Note, tgl, cl.UserID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		catatAudit(tx, cl.UserID, "create", "stock_movements", id, req)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "stok tersimpan"})
	}
}

func stokList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		q := `SELECT id, location_id, direction, category, item_name, weight_gram, IFNULL(karat,''), qty, IFNULL(note,''), transaction_date
		      FROM stock_movements WHERE is_void = FALSE`
		args := []interface{}{}

		if cl.Role == "cabang" {
			q += " AND location_id = ?"
			args = append(args, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" {
			q += " AND location_id = ?"
			args = append(args, loc)
		}
		if v := c.Query("from"); v != "" {
			q += " AND transaction_date >= ?"
			args = append(args, v)
		}
		if v := c.Query("to"); v != "" {
			q += " AND transaction_date <= ?"
			args = append(args, v)
		}
		q += " ORDER BY transaction_date DESC, id DESC LIMIT 200"

		rows, err := db.Query(q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type row struct {
			ID         int       `json:"id"`
			LocationID int       `json:"location_id"`
			Direction  string    `json:"direction"`
			Category   string    `json:"category"`
			ItemName   string    `json:"item_name"`
			WeightGram float64   `json:"weight_gram"`
			Karat      string    `json:"karat"`
			Qty        int       `json:"qty"`
			Note       string    `json:"note"`
			Date       time.Time `json:"transaction_date"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Direction, &r.Category, &r.ItemName, &r.WeightGram, &r.Karat, &r.Qty, &r.Note, &r.Date); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}