package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// barang selesai dicuci: kembali ke stok back office, kondisi bersih
func selesaiCuci(tx *sql.Tx, ids []int, tgl string, userID int, boID int) error {
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := tx.Query(
		`SELECT i.id, i.item_name, IFNULL(i.karat,''), i.weight_gram, IFNULL(i.washing_job_id,0), IFNULL(w.washer_name,'')
		 FROM items i LEFT JOIN washing_jobs w ON w.id = i.washing_job_id
		 WHERE i.id IN (`+placeholders(len(args))+`)`, args...)
	if err != nil {
		return err
	}
	type brg struct {
		ID      int
		Name    string
		Karat   string
		Weight  float64
		Job     int
		Washer  string
	}
	list := []brg{}
	for rows.Next() {
		var b brg
		if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight, &b.Job, &b.Washer); err != nil {
			rows.Close()
			return err
		}
		list = append(list, b)
	}
	rows.Close()

	jobs := map[int]int{}
	for _, b := range list {
		if _, err := tx.Exec(sqlStokItem, boID, "in", "cucian", b.Name, b.Weight, b.Karat,
			"Selesai cuci oleh "+b.Washer, tgl, userID, b.ID); err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE items SET status = 'tersedia', kondisi = 'bersih' WHERE id = ?", b.ID); err != nil {
			return err
		}
		if b.Job > 0 {
			jobs[b.Job]++
		}
	}
	for job, n := range jobs {
		var sisa int
		tx.QueryRow("SELECT COUNT(*) FROM items WHERE washing_job_id = ? AND status = 'cuci'", job).Scan(&sisa)
		if sisa == 0 {
			tx.Exec("UPDATE washing_jobs SET status = 'selesai', finish_date = ? WHERE id = ?", tgl, job)
		}
		catatAudit(tx, userID, "cuci_selesai", "washing_jobs", int64(job), gin.H{"jumlah": n, "sisa": sisa})
	}
	return nil
}

// selesai untuk barang yang dicentang
func cuciSelesaiBarang(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req struct {
			ItemIDs    []int  `json:"item_ids"`
			FinishDate string `json:"finish_date"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.ItemIDs) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
		tgl, ok := parseTanggal(c, req.FinishDate)
		if !ok {
			return
		}
		boID, err := idBackOffice(db)
		if err != nil {
			gagal(c, err)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			gagal(c, err)
			return
		}
		defer tx.Rollback()

		items, err := ambilBarang(tx, req.ItemIDs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		for _, it := range items {
			if it.Status != "cuci" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak sedang dicuci", it.Name)})
				return
			}
		}
		if err := selesaiCuci(tx, kunci(items), tgl, cl.UserID, boID); err != nil {
			gagal(c, err)
			return
		}
		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang selesai dicuci, kembali ke stok back office (bersih)", len(items))})
	}
}

// selesai untuk semua barang yang tersisa di satu pekerjaan cuci
func cuciSelesaiJob(db *sql.DB) gin.HandlerFunc {
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
		c.ShouldBindJSON(&req)
		tgl, ok := parseTanggal(c, req.FinishDate)
		if !ok {
			return
		}
		boID, err := idBackOffice(db)
		if err != nil {
			gagal(c, err)
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
		err = tx.QueryRow("SELECT status, is_void FROM washing_jobs WHERE id = ? FOR UPDATE", id).Scan(&status, &isVoid)
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
		ids, err := ambilIDs(tx, "SELECT id FROM items WHERE washing_job_id = ? AND status = 'cuci' FOR UPDATE", id)
		if err != nil {
			gagal(c, err)
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam proses cuci"})
			return
		}
		if err := selesaiCuci(tx, ids, tgl, cl.UserID, boID); err != nil {
			gagal(c, err)
			return
		}
		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("cuci selesai, %d barang kembali ke stok back office (bersih)", len(ids))})
	}
}

// batalkan pekerjaan cuci (hanya kalau belum ada barang yang selesai)
func cuciVoidV2(db *sql.DB) gin.HandlerFunc {
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
		err = tx.QueryRow("SELECT status, is_void FROM washing_jobs WHERE id = ? FOR UPDATE", id).Scan(&status, &isVoid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pekerjaan cuci tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}
		if status == "selesai" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cuci sudah selesai, tidak bisa dibatalkan"})
			return
		}
		var selesaiN int
		tx.QueryRow("SELECT COUNT(*) FROM items WHERE washing_job_id = ? AND status <> 'cuci' AND is_void = FALSE", id).Scan(&selesaiN)
		if selesaiN > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sebagian barang sudah selesai dicuci, tidak bisa dibatalkan"})
			return
		}

		ids, err := ambilIDs(tx, "SELECT id FROM items WHERE washing_job_id = ? AND status = 'cuci' FOR UPDATE", id)
		if err != nil {
			gagal(c, err)
			return
		}
		for _, itemID := range ids {
			tx.Exec(`UPDATE stock_movements SET is_void = TRUE, void_reason = ?
			         WHERE item_id = ? AND category = 'cuci' AND direction = 'out' AND is_void = FALSE
			         ORDER BY id DESC LIMIT 1`, reason, itemID)
			tx.Exec("UPDATE items SET status = 'tersedia', washing_job_id = NULL WHERE id = ?", itemID)
		}
		tx.Exec("UPDATE washing_jobs SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		catatAudit(tx, cl.UserID, "void", "washing_jobs", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cuci dibatalkan, barang kembali tersedia di back office"})
	}
}

// daftar pekerjaan cuci beserta progres
func cuciListV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		q := `SELECT w.id, w.origin_location_id, l.name, w.item_name, IFNULL(w.karat,''), w.weight_gram, w.qty,
		             w.washer_name, w.status, w.start_date, w.finish_date, IFNULL(w.note,''),
		             (SELECT COUNT(*) FROM items i WHERE i.washing_job_id = w.id AND i.status = 'cuci' AND i.is_void = FALSE)
		      FROM washing_jobs w JOIN locations l ON l.id = w.origin_location_id
		      WHERE w.is_void = FALSE`
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND w.origin_location_id = ?"
			args = append(args, cl.LocationID)
		}
		switch s := c.Query("status"); s {
		case "menunggu", "proses", "selesai":
			q += " AND w.status = ?"
			args = append(args, s)
		case "aktif":
			q += " AND w.status <> 'selesai'"
		}
		q += " ORDER BY w.status ASC, w.id DESC LIMIT 200"

		rows, err := db.Query(q, args...)
		if err != nil {
			gagal(c, err)
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
			Sisa       int        `json:"sisa"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var fin sql.NullTime
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Cabang, &r.ItemName, &r.Karat, &r.WeightGram, &r.Qty,
				&r.Washer, &r.Status, &r.StartDate, &fin, &r.Note, &r.Sisa); err != nil {
				gagal(c, err)
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