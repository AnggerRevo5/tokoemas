package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ---------- OWNER: MULAI CUCI (tanpa nama pencuci) ----------

func cuciMulaiV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req struct {
			ItemIDs   []int  `json:"item_ids"`
			StartDate string `json:"start_date"`
			Note      string `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		tgl, ok := parseTanggal(c, req.StartDate)
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

		items, ok := barangDiBO(c, tx, req.ItemIDs, boID)
		if !ok {
			return
		}
		asal, err := ambilAsal(tx, kunci(items))
		if err != nil {
			gagal(c, err)
			return
		}
		groups := map[int][]itemRow{}
		for id, it := range items {
			groups[asal[id]] = append(groups[asal[id]], it)
		}

		note := strings.TrimSpace(req.Note)
		jobs := 0
		for origin, g := range groups {
			nama, berat, qty, karat := ringkas(g)
			res, err := tx.Exec(
				`INSERT INTO washing_jobs (origin_location_id, item_name, weight_gram, karat, qty, washer_name, status, start_date, note, created_by)
				 VALUES (?,?,?,?,?,'','menunggu',?,?,?)`,
				origin, nama, berat, karat, qty, tgl, note, cl.UserID,
			)
			if err != nil {
				gagal(c, err)
				return
			}
			jobID, _ := res.LastInsertId()
			for _, it := range g {
				if _, err := tx.Exec(sqlStokItem, boID, "out", "cuci", it.Name, it.Weight, it.Karat,
					"Masuk antrean cuci", tgl, cl.UserID, it.ID); err != nil {
					gagal(c, err)
					return
				}
				if _, err := tx.Exec("UPDATE items SET status = 'cuci', washing_job_id = ? WHERE id = ?", jobID, it.ID); err != nil {
					gagal(c, err)
					return
				}
			}
			catatAudit(tx, cl.UserID, "cuci_mulai", "washing_jobs", jobID, gin.H{"jumlah": len(g)})
			jobs++
		}

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message": fmt.Sprintf("%d barang masuk antrean cuci (%d pekerjaan), menunggu PJ mengirim ke pencuci", len(items), jobs),
		})
	}
}

// ---------- PJ: KIRIM KE PENCUCI ----------

func cuciKirim(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaPJ(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		var req struct {
			WasherName string `json:"washer_name"`
			Date       string `json:"date"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		nama := strings.TrimSpace(req.WasherName)
		if nama == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama pencuci wajib diisi"})
			return
		}
		tgl, ok := parseTanggal(c, req.Date)
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
		if isVoid || status != "menunggu" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pekerjaan cuci ini sudah dikirim atau dibatalkan"})
			return
		}
		if _, err := tx.Exec(
			"UPDATE washing_jobs SET status = 'proses', washer_name = ?, washer_sent_date = ?, sent_by = ? WHERE id = ?",
			nama, tgl, cl.UserID, id,
		); err != nil {
			gagal(c, err)
			return
		}
		catatAudit(tx, cl.UserID, "cuci_kirim", "washing_jobs", int64(id), gin.H{"pencuci": nama})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("dikirim ke pencuci %s", nama)})
	}
}

// ---------- PJ: SELESAI CUCI ----------

func cuciSelesaiBarangV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaPJ(c)
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
		ids := kunci(items)
		args := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		var belum int
		tx.QueryRow(
			`SELECT COUNT(*) FROM items i JOIN washing_jobs w ON w.id = i.washing_job_id
			 WHERE i.id IN (`+placeholders(len(args))+`) AND w.status <> 'proses'`, args...,
		).Scan(&belum)
		if belum > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang yang belum dikirim ke pencuci, kirim dulu"})
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
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang selesai dicuci, kembali ke stok back office (bersih)", len(items))})
	}
}

func cuciSelesaiJobV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaPJ(c)
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
		if status != "proses" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cuci belum dikirim ke pencuci, atau sudah selesai"})
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

// ---------- DAFTAR BARANG DI PENCUCI (untuk PJ) ----------

func cucianBarang(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := bolehLihatLebur(c); !ok {
			return
		}
		rows, err := db.Query(`
			SELECT i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), i.weight_gram, o.name,
			       w.id, w.status, IFNULL(w.washer_name,''), w.start_date, w.washer_sent_date
			FROM items i
			JOIN washing_jobs w ON w.id = i.washing_job_id
			JOIN locations o ON o.id = i.origin_location_id
			WHERE i.is_void = FALSE AND i.status = 'cuci' AND w.is_void = FALSE
			ORDER BY w.id DESC, i.id`)
		if err != nil {
			gagal(c, err)
			return
		}
		defer rows.Close()

		type row struct {
			ID         int        `json:"id"`
			ItemNo     string     `json:"item_no"`
			ItemName   string     `json:"item_name"`
			Karat      string     `json:"karat"`
			WeightGram float64    `json:"weight_gram"`
			Asal       string     `json:"asal"`
			JobID      int        `json:"job_id"`
			JobStatus  string     `json:"job_status"`
			Pencuci    string     `json:"washer_name"`
			StartDate  time.Time  `json:"start_date"`
			SentDate   *time.Time `json:"sent_date"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var sent sql.NullTime
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram, &r.Asal,
				&r.JobID, &r.JobStatus, &r.Pencuci, &r.StartDate, &sent); err != nil {
				gagal(c, err)
				return
			}
			r.SentDate = ptrTime(sent)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- ANGKA MERAH UNTUK PJ ----------

func cucianNotif(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		n := 0
		if cl.Role == "pj" || cl.Role == "super_admin" {
			db.QueryRow("SELECT COUNT(*) FROM washing_jobs WHERE is_void = FALSE AND status = 'menunggu'").Scan(&n)
		}
		c.JSON(http.StatusOK, gin.H{"badge": n})
	}
}