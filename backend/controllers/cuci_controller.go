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

func CuciMulai(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req models.CuciMulaiRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.StartDate)
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

		items, ok := barangDiBO(c, tx, req.ItemIDs, boID)
		if !ok {
			return
		}
		asal, err := services.AmbilAsal(tx, services.Kunci(items))
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		groups := map[int][]models.ItemRow{}
		for id, it := range items {
			groups[asal[id]] = append(groups[asal[id]], it)
		}

		note := strings.TrimSpace(req.Note)
		jobs := 0
		for origin, g := range groups {
			nama, berat, qty, karat := services.Ringkas(g)
			res, err := tx.Exec(
				`INSERT INTO washing_jobs (origin_location_id, item_name, weight_gram, karat, qty, washer_name, status, start_date, note, created_by)
				 VALUES (?,?,?,?,?,'','menunggu',?,?,?)`,
				origin, nama, berat, karat, qty, tgl, note, cl.UserID,
			)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			jobID, _ := res.LastInsertId()
			for _, it := range g {
				if _, err := tx.Exec(services.SQLStokItem, boID, "out", "cuci", it.Name, it.Weight, it.Karat,
					"Masuk antrean cuci", tgl, cl.UserID, it.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
				if _, err := tx.Exec("UPDATE items SET status = 'cuci', washing_job_id = ? WHERE id = ?", jobID, it.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			services.CatatAudit(tx, cl.UserID, "cuci_mulai", "washing_jobs", jobID, gin.H{"jumlah": len(g)})
			jobs++
		}

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message": fmt.Sprintf("%d barang masuk antrean cuci (%d pekerjaan), menunggu PJ mengirim ke pencuci", len(items), jobs),
		})
	}
}

// ---------- PJ: KIRIM KE PENCUCI ----------

func CuciSelesaiBarangLangsung(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		var req models.CuciSelesaiBarangLangsungRequest
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.ItemIDs) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.FinishDate)
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

		items, err := services.AmbilBarang(tx, req.ItemIDs)
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
		ids := services.Kunci(items)
		args := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		var belum int
		tx.QueryRow(
			`SELECT COUNT(*) FROM items i JOIN washing_jobs w ON w.id = i.washing_job_id
			 WHERE i.id IN (`+utils.Placeholders(len(args))+`) AND w.status <> 'proses'`, args...,
		).Scan(&belum)
		if belum > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang yang belum dikirim ke pencuci, kirim dulu"})
			return
		}

		if err := services.SelesaiCuci(tx, ids, tgl, cl.UserID, boID); err != nil {
			utils.Gagal(c, err)
			return
		}
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang selesai dicuci, kembali ke stok back office (bersih)", len(items))})
	}
}

func CucianNotif(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		n := 0
		if cl.Role == "pj" || cl.Role == "super_admin" {
			db.QueryRow("SELECT COUNT(*) FROM washing_jobs WHERE is_void = FALSE AND status = 'menunggu'").Scan(&n)
		}
		c.JSON(http.StatusOK, gin.H{"badge": n})
	}
}

// batalkan pekerjaan cuci (hanya kalau belum ada barang yang selesai)
func CuciVoid(db *sql.DB) gin.HandlerFunc {
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

		ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE washing_job_id = ? AND status = 'cuci' FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		for _, itemID := range ids {
			tx.Exec(`UPDATE stock_movements SET is_void = TRUE, void_reason = ?
			         WHERE item_id = ? AND category = 'cuci' AND direction = 'out' AND is_void = FALSE
			         ORDER BY id DESC LIMIT 1`, reason, itemID)
			tx.Exec("UPDATE items SET status = 'tersedia', washing_job_id = NULL WHERE id = ?", itemID)
		}
		tx.Exec("UPDATE washing_jobs SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		services.CatatAudit(tx, cl.UserID, "void", "washing_jobs", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cuci dibatalkan, barang kembali tersedia di back office"})
	}
}

// ---------- PJ: KIRIM KE PENCUCI DENGAN BERAT AWAL ----------

func CuciKirim(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.CuciKirimRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		nama := strings.TrimSpace(req.WasherName)
		if nama == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama pencuci wajib diisi"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.Date)
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
		err = tx.QueryRow("SELECT status, is_void FROM washing_jobs WHERE id = ? FOR UPDATE", id).Scan(&status, &isVoid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pekerjaan cuci tidak ditemukan"})
			return
		}
		if isVoid || status != "menunggu" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pekerjaan cuci ini sudah dikirim atau dibatalkan"})
			return
		}

		rows, err := tx.Query(
			"SELECT id, item_name, weight_gram FROM items WHERE washing_job_id = ? AND status = 'cuci' AND is_void = FALSE FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		type brg struct {
			Name string
			W    float64
		}
		items := map[int]brg{}
		for rows.Next() {
			var iid int
			var b brg
			if err := rows.Scan(&iid, &b.Name, &b.W); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			items[iid] = b
		}
		rows.Close()
		if len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam pekerjaan cuci ini"})
			return
		}

		timbang := map[int]float64{}
		for _, w := range req.Weights {
			if w.Weight <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "berat awal harus lebih dari 0"})
				return
			}
			timbang[w.ItemID] = w.Weight
		}
		for iid, it := range items {
			if _, ada := timbang[iid]; !ada {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("berat awal %s belum diisi", it.Name)})
				return
			}
		}
		for iid, it := range items {
			if _, err := tx.Exec(
				`INSERT INTO washing_job_items (job_id, item_id, weight_system, weight_before) VALUES (?,?,?,?)
				 ON DUPLICATE KEY UPDATE weight_before = VALUES(weight_before)`,
				id, iid, it.W, timbang[iid],
			); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			"UPDATE washing_jobs SET status = 'proses', washer_name = ?, washer_sent_date = ?, sent_by = ? WHERE id = ?",
			nama, tgl, cl.UserID, id,
		); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "cuci_kirim", "washing_jobs", int64(id), gin.H{"pencuci": nama, "berat_awal": timbang})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("dikirim ke pencuci %s, berat awal %d barang tercatat", nama, len(items))})
	}
}

// ---------- PJ: SELESAI CUCI DENGAN BERAT SESUDAH ----------

func CuciSelesaiBarang(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		var req models.CuciSelesaiBarangRequest
		if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.Items) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.FinishDate)
		if !ok {
			return
		}
		sesudah := map[int]float64{}
		args := []interface{}{}
		for _, it := range req.Items {
			if it.Weight <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "berat sesudah harus lebih dari 0"})
				return
			}
			if _, ada := sesudah[it.ItemID]; !ada {
				args = append(args, it.ItemID)
			}
			sesudah[it.ItemID] = it.Weight
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		rows, err := tx.Query(
			`SELECT i.id, i.item_name, i.status, IFNULL(i.washing_job_id,0), IFNULL(w.status,''), i.weight_gram,
			        IFNULL(x.weight_before, i.weight_gram)
			 FROM items i
			 LEFT JOIN washing_jobs w ON w.id = i.washing_job_id
			 LEFT JOIN washing_job_items x ON x.job_id = i.washing_job_id AND x.item_id = i.id
			 WHERE i.is_void = FALSE AND i.id IN (`+utils.Placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		type brg struct {
			ID        int
			Name      string
			Status    string
			Job       int
			JobStatus string
			Sistem    float64
			Awal      float64
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Status, &b.Job, &b.JobStatus, &b.Sistem, &b.Awal); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			list = append(list, b)
		}
		rows.Close()
		if len(list) != len(args) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang yang tidak ditemukan"})
			return
		}

		for _, b := range list {
			if b.Status != "cuci" || b.JobStatus != "proses" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s belum dikirim ke pencuci atau tidak sedang dicuci", b.Name)})
				return
			}
			if sesudah[b.ID] > b.Awal+0.0005 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("berat sesudah %s tidak boleh lebih dari berat awal (%.3f g)", b.Name, b.Awal)})
				return
			}
		}

		jobs := map[int]bool{}
		var totalSusut float64
		for _, b := range list {
			after := sesudah[b.ID]
			if _, err := tx.Exec(
				`INSERT INTO washing_job_items (job_id, item_id, weight_system, weight_before, weight_after) VALUES (?,?,?,?,?)
				 ON DUPLICATE KEY UPDATE weight_after = VALUES(weight_after)`,
				b.Job, b.ID, b.Sistem, b.Awal, after,
			); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec(
				`UPDATE items SET weight_gram = ?, status = 'kembali', kembali_jenis = 'cuci_selesai',
				        belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?`, after, b.ID,
			); err != nil {
				utils.Gagal(c, err)
				return
			}
			totalSusut += b.Awal - after
			jobs[b.Job] = true
		}
		for job := range jobs {
			var sisa int
			tx.QueryRow("SELECT COUNT(*) FROM items WHERE washing_job_id = ? AND status = 'cuci' AND is_void = FALSE", job).Scan(&sisa)
			if sisa == 0 {
				tx.Exec("UPDATE washing_jobs SET status = 'selesai', finish_date = ? WHERE id = ?", tgl, job)
			}
			services.CatatAudit(tx, cl.UserID, "cuci_selesai", "washing_jobs", int64(job), gin.H{"sisa": sisa, "susut": totalSusut})
		}

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("%d barang selesai dicuci (susut %.3f g), dikembalikan ke back office dan menunggu konfirmasi owner", len(list), totalSusut),
		})
	}
}

// ---------- PJ: BATAL CUCI / LEBUR, BARANG MENUNGGU KONFIRMASI BACK OFFICE ----------

func CuciKembali(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.CuciKembaliRequest
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan wajib diisi"})
			return
		}
		alasan := strings.TrimSpace(req.Reason)
		tgl, ok := utils.ParseTanggal(c, req.Date)
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
		err = tx.QueryRow("SELECT status, is_void FROM washing_jobs WHERE id = ? FOR UPDATE", id).Scan(&status, &isVoid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pekerjaan cuci tidak ditemukan"})
			return
		}
		if isVoid || (status != "menunggu" && status != "proses") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pekerjaan cuci sudah selesai atau dibatalkan"})
			return
		}

		ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE washing_job_id = ? AND status = 'cuci' AND is_void = FALSE FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang yang masih di pencuci"})
			return
		}
		for _, itemID := range ids {
			if _, err := tx.Exec(
				`UPDATE items SET status = 'kembali', kembali_jenis = 'cuci_batal', belum_sampai = FALSE, belum_sampai_note = NULL
				 WHERE id = ?`, itemID,
			); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			`UPDATE washing_jobs SET status = 'batal', cancel_reason = ?, cancelled_by = ?, cancelled_date = ?, cancel_returned = ?
			 WHERE id = ?`, utils.BatasTeks(alasan, 250), cl.UserID, tgl, len(ids), id,
		); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "cuci_batal_kembali", "washing_jobs", int64(id), gin.H{"alasan": alasan, "dikembalikan": len(ids)})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("cuci #%d dibatalkan, %d barang dikembalikan ke back office (menunggu konfirmasi owner)", id, len(ids)),
		})
	}
}

func CucianBarang(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.BolehLihatLebur(c); !ok {
			return
		}
		rows, err := db.Query(`
			SELECT i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), i.weight_gram, o.name,
			       w.id, w.status, IFNULL(w.washer_name,''), w.start_date, w.washer_sent_date,
			       IFNULL(x.weight_before, i.weight_gram)
			FROM items i
			JOIN washing_jobs w ON w.id = i.washing_job_id
			JOIN locations o ON o.id = i.origin_location_id
			LEFT JOIN washing_job_items x ON x.job_id = w.id AND x.item_id = i.id
			WHERE i.is_void = FALSE AND i.status = 'cuci' AND w.is_void = FALSE
			ORDER BY w.id DESC, i.id`)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.CucianBarangRow{}
		for rows.Next() {
			var r models.CucianBarangRow
			var sent sql.NullTime
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram, &r.Asal,
				&r.JobID, &r.JobStatus, &r.Pencuci, &r.StartDate, &sent, &r.BeratAwal); err != nil {
				utils.Gagal(c, err)
				return
			}
			r.SentDate = utils.PtrTime(sent)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- DAFTAR CUCI (dengan susut) ----------

func CuciList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		q := `SELECT w.id, w.origin_location_id, l.name, w.item_name, IFNULL(w.karat,''), w.weight_gram, w.qty,
		             w.washer_name, w.status, w.start_date, w.finish_date, IFNULL(w.note,''),
		             (SELECT COUNT(*) FROM items i WHERE i.washing_job_id = w.id AND i.status = 'cuci' AND i.is_void = FALSE),
		             IFNULL(w.cancel_reason,''), IFNULL(w.cancel_returned,0),
		             IFNULL((SELECT SUM(x.weight_before) FROM washing_job_items x WHERE x.job_id = w.id AND x.weight_after IS NOT NULL),0),
		             IFNULL((SELECT SUM(x.weight_after) FROM washing_job_items x WHERE x.job_id = w.id),0)
		      FROM washing_jobs w JOIN locations l ON l.id = w.origin_location_id
		      WHERE w.is_void = FALSE`
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND w.origin_location_id = ?"
			args = append(args, cl.LocationID)
		}
		switch s := c.Query("status"); s {
		case "menunggu", "proses", "selesai", "batal":
			q += " AND w.status = ?"
			args = append(args, s)
		case "aktif":
			q += " AND w.status NOT IN ('selesai','batal')"
		}
		q += " ORDER BY w.status ASC, w.id DESC LIMIT 200"

		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.CuciListRow{}
		for rows.Next() {
			var r models.CuciListRow
			var fin sql.NullTime
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Cabang, &r.ItemName, &r.Karat, &r.WeightGram, &r.Qty,
				&r.Washer, &r.Status, &r.StartDate, &fin, &r.Note, &r.Sisa, &r.CancelReason, &r.CancelReturned,
				&r.BeratAwalSelesai, &r.BeratSesudah); err != nil {
				utils.Gagal(c, err)
				return
			}
			r.FinishDate = utils.PtrTime(fin)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}
