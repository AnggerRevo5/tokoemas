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

func hanyaSuperAdmin(c *gin.Context) (*Claims, bool) {
	cl := c.MustGet("claims").(*Claims)
	if cl.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "hanya Super Admin yang bisa membatalkan"})
		return nil, false
	}
	return cl, true
}

func idParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return 0, false
	}
	return id, true
}

func cekCabang(db *sql.DB, id int) (string, bool) {
	var nama, tipe string
	if err := db.QueryRow("SELECT name, type FROM locations WHERE id = ?", id).Scan(&nama, &tipe); err != nil || tipe != "cabang" {
		return "", false
	}
	return nama, true
}

func idBackOffice(db *sql.DB) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM locations WHERE type = 'back_office' LIMIT 1").Scan(&id)
	return id, err
}

const sqlStokBaru = `INSERT INTO stock_movements
	(location_id, direction, category, item_name, weight_gram, karat, qty, note, transaction_date, created_by)
	VALUES (?,?,?,?,?,?,?,?,?,?)`

// ---------- CUCI ----------

type cuciReq struct {
	OriginLocationID int     `json:"origin_location_id"`
	ItemName         string  `json:"item_name"`
	WeightGram       float64 `json:"weight_gram"`
	Karat            string  `json:"karat"`
	Qty              int     `json:"qty"`
	WasherName       string  `json:"washer_name"`
	StartDate        string  `json:"start_date"`
	Note             string  `json:"note"`
}

func cuciCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req cuciReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		namaCabang, ok := cekCabang(db, req.OriginLocationID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih toko asal (Kepanjen atau Mall Dinoyo)"})
			return
		}
		if strings.TrimSpace(req.ItemName) == "" || req.WeightGram <= 0 || strings.TrimSpace(req.WasherName) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama barang, berat, dan nama pencuci wajib diisi"})
			return
		}
		if req.Qty < 1 {
			req.Qty = 1
		}
		tgl, ok := parseTanggal(c, req.StartDate)
		if !ok {
			return
		}
		boID, err := idBackOffice(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(sqlStokBaru, boID, "out", "cuci", req.ItemName, req.WeightGram, req.Karat, req.Qty,
			fmt.Sprintf("Cuci oleh %s - asal %s", req.WasherName, namaCabang), tgl, cl.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		outID, _ := res.LastInsertId()

		res2, err := tx.Exec(
			`INSERT INTO washing_jobs
			 (origin_location_id, item_name, weight_gram, karat, qty, washer_name, status, start_date, note, out_stock_id, created_by)
			 VALUES (?,?,?,?,?,?,'proses',?,?,?,?)`,
			req.OriginLocationID, req.ItemName, req.WeightGram, req.Karat, req.Qty, req.WasherName, tgl, req.Note, outID, cl.UserID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res2.LastInsertId()
		catatAudit(tx, cl.UserID, "create", "washing_jobs", id, req)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "cuci dicatat, status proses"})
	}
}

func cuciSelesai(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		var req struct {
			FinishDate string `json:"finish_date"`
		}
		c.ShouldBindJSON(&req) // body boleh kosong
		tgl, ok := parseTanggal(c, req.FinishDate)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var origin, qty int
		var outID int64
		var item, karat, washer, status string
		var weight float64
		var isVoid bool
		err = tx.QueryRow(
			`SELECT origin_location_id, item_name, IFNULL(karat,''), weight_gram, qty, washer_name, status, is_void, out_stock_id
			 FROM washing_jobs WHERE id = ? FOR UPDATE`, id,
		).Scan(&origin, &item, &karat, &weight, &qty, &washer, &status, &isVoid, &outID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pekerjaan cuci tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pekerjaan cuci sudah dibatalkan"})
			return
		}
		if status == "selesai" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cuci sudah selesai"})
			return
		}

		res, err := tx.Exec(sqlStokBaru, origin, "in", "barang_masuk", item, weight, karat, qty,
			fmt.Sprintf("Selesai cuci oleh %s", washer), tgl, cl.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		inID, _ := res.LastInsertId()
		tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", inID, outID)
		tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", outID, inID)
		if _, err := tx.Exec(
			"UPDATE washing_jobs SET status = 'selesai', finish_date = ?, in_stock_id = ? WHERE id = ?", tgl, inID, id,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		catatAudit(tx, cl.UserID, "cuci_selesai", "washing_jobs", int64(id), gin.H{"finish_date": tgl, "in_stock_id": inID})

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cuci selesai, barang masuk ke stok toko"})
	}
}

func cuciList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		q := `SELECT w.id, w.origin_location_id, l.name, w.item_name, IFNULL(w.karat,''), w.weight_gram, w.qty,
		             w.washer_name, w.status, w.start_date, w.finish_date, IFNULL(w.note,'')
		      FROM washing_jobs w JOIN locations l ON l.id = w.origin_location_id
		      WHERE w.is_void = FALSE`
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND w.origin_location_id = ?"
			args = append(args, cl.LocationID)
		}
		if s := c.Query("status"); s == "proses" || s == "selesai" {
			q += " AND w.status = ?"
			args = append(args, s)
		}
		q += " ORDER BY w.status ASC, w.id DESC LIMIT 200"

		rows, err := db.Query(q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type row struct {
			ID         int        `json:"id"`
			LocationID int        `json:"origin_location_id"`
			Cabang     string     `json:"cabang"`
			ItemName   string     `json:"item_name"`
			Karat      string     `json:"karat"`
			WeightGram float64    `json:"weight_gram"`
			Qty        int        `json:"qty"`
			Washer     string     `json:"washer_name"`
			Status     string     `json:"status"`
			StartDate  time.Time  `json:"start_date"`
			FinishDate *time.Time `json:"finish_date"`
			Note       string     `json:"note"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var fin sql.NullTime
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Cabang, &r.ItemName, &r.Karat, &r.WeightGram, &r.Qty,
				&r.Washer, &r.Status, &r.StartDate, &fin, &r.Note); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if fin.Valid {
				t := fin.Time
				r.FinishDate = &t
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

func cuciVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan pembatalan wajib diisi"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var isVoid bool
		var outID, inID sql.NullInt64
		err = tx.QueryRow("SELECT is_void, out_stock_id, in_stock_id FROM washing_jobs WHERE id = ? FOR UPDATE", id).
			Scan(&isVoid, &outID, &inID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pekerjaan cuci tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}

		tx.Exec("UPDATE washing_jobs SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, id)
		catatAudit(tx, cl.UserID, "void", "washing_jobs", int64(id), gin.H{"reason": req.Reason})
		for _, s := range []sql.NullInt64{outID, inID} {
			if s.Valid {
				tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, s.Int64)
				catatAudit(tx, cl.UserID, "void", "stock_movements", s.Int64, gin.H{"reason": req.Reason})
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cuci dibatalkan"})
	}
}

// ---------- LEBUR ----------

type leburReq struct {
	RecordDate       string  `json:"record_date"`
	OriginLocationID int     `json:"origin_location_id"`
	ItemName         string  `json:"item_name"`
	Karat            string  `json:"karat"`
	WeightGram       float64 `json:"weight_gram"`
	Qty              int     `json:"qty"`
	MeltTotal        float64 `json:"melt_total"`
	SaleTotal        float64 `json:"sale_total"`
	Note             string  `json:"note"`
}

func leburCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req leburReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		namaCabang, ok := cekCabang(db, req.OriginLocationID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih cabang asal (Kepanjen atau Mall Dinoyo)"})
			return
		}
		if strings.TrimSpace(req.ItemName) == "" || req.WeightGram <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama barang dan berat wajib diisi"})
			return
		}
		if req.MeltTotal <= 0 || req.SaleTotal <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "total lebur dan total harga jual harus lebih dari 0"})
			return
		}
		if req.Qty < 1 {
			req.Qty = 1
		}
		tgl, ok := parseTanggal(c, req.RecordDate)
		if !ok {
			return
		}
		boID, err := idBackOffice(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(sqlStokBaru, boID, "out", "lebur", req.ItemName, req.WeightGram, req.Karat, req.Qty,
			fmt.Sprintf("Lebur - asal %s", namaCabang), tgl, cl.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		stockID, _ := res.LastInsertId()

		res2, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			boID, "in", "jual_barang", req.SaleTotal,
			fmt.Sprintf("Hasil lebur %s - asal %s", req.ItemName, namaCabang), tgl, cl.UserID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cashID, _ := res2.LastInsertId()

		res3, err := tx.Exec(
			`INSERT INTO melt_records
			 (record_date, origin_location_id, item_name, karat, weight_gram, qty, melt_total, sale_total, note, stock_id, cash_id, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			tgl, req.OriginLocationID, req.ItemName, req.Karat, req.WeightGram, req.Qty,
			req.MeltTotal, req.SaleTotal, req.Note, stockID, cashID, cl.UserID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res3.LastInsertId()
		catatAudit(tx, cl.UserID, "create", "melt_records", id, req)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":      id,
			"message": "lebur dicatat, kas back office bertambah",
			"selisih": req.SaleTotal - req.MeltTotal,
		})
	}
}

func leburList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := hanyaAdminOwner(c); !ok {
			return
		}
		q := `SELECT m.id, m.record_date, l.name, m.item_name, IFNULL(m.karat,''), m.weight_gram, m.qty,
		             m.melt_total, m.sale_total, IFNULL(m.note,'')
		      FROM melt_records m JOIN locations l ON l.id = m.origin_location_id
		      WHERE m.is_void = FALSE`
		args := []interface{}{}
		if v := c.Query("from"); v != "" {
			q += " AND m.record_date >= ?"
			args = append(args, v)
		}
		if v := c.Query("to"); v != "" {
			q += " AND m.record_date <= ?"
			args = append(args, v)
		}
		q += " ORDER BY m.record_date DESC, m.id DESC LIMIT 200"

		rows, err := db.Query(q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type row struct {
			ID         int       `json:"id"`
			Date       time.Time `json:"record_date"`
			Cabang     string    `json:"cabang"`
			ItemName   string    `json:"item_name"`
			Karat      string    `json:"karat"`
			WeightGram float64   `json:"weight_gram"`
			Qty        int       `json:"qty"`
			MeltTotal  float64   `json:"melt_total"`
			SaleTotal  float64   `json:"sale_total"`
			Selisih    float64   `json:"selisih"`
			Note       string    `json:"note"`
		}
		out := []row{}
		var totalLebur, totalJual float64
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.Date, &r.Cabang, &r.ItemName, &r.Karat, &r.WeightGram, &r.Qty,
				&r.MeltTotal, &r.SaleTotal, &r.Note); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			r.Selisih = r.SaleTotal - r.MeltTotal
			totalLebur += r.MeltTotal
			totalJual += r.SaleTotal
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{
			"rows":        out,
			"total_lebur": totalLebur,
			"total_jual":  totalJual,
			"selisih":     totalJual - totalLebur,
		})
	}
}

func leburVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan pembatalan wajib diisi"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var isVoid bool
		var stockID, cashID sql.NullInt64
		err = tx.QueryRow("SELECT is_void, stock_id, cash_id FROM melt_records WHERE id = ? FOR UPDATE", id).
			Scan(&isVoid, &stockID, &cashID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "catatan lebur tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}

		tx.Exec("UPDATE melt_records SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, id)
		catatAudit(tx, cl.UserID, "void", "melt_records", int64(id), gin.H{"reason": req.Reason})
		if stockID.Valid {
			tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, stockID.Int64)
			catatAudit(tx, cl.UserID, "void", "stock_movements", stockID.Int64, gin.H{"reason": req.Reason})
		}
		if cashID.Valid {
			tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, cashID.Int64)
			catatAudit(tx, cl.UserID, "void", "cash_transactions", cashID.Int64, gin.H{"reason": req.Reason})
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "lebur dibatalkan, stok dan kas dikembalikan"})
	}
}

