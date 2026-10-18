package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type beratItem struct {
	ItemID int     `json:"item_id"`
	Weight float64 `json:"weight"`
}

// ---------- PJ: KIRIM KE PENCUCI DENGAN BERAT AWAL ----------

func cuciKirimV2(db *sql.DB) gin.HandlerFunc {
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
			WasherName string      `json:"washer_name"`
			Date       string      `json:"date"`
			Weights    []beratItem `json:"weights"`
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

		rows, err := tx.Query(
			"SELECT id, item_name, weight_gram FROM items WHERE washing_job_id = ? AND status = 'cuci' AND is_void = FALSE FOR UPDATE", id)
		if err != nil {
			gagal(c, err)
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
				gagal(c, err)
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
				gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			"UPDATE washing_jobs SET status = 'proses', washer_name = ?, washer_sent_date = ?, sent_by = ? WHERE id = ?",
			nama, tgl, cl.UserID, id,
		); err != nil {
			gagal(c, err)
			return
		}
		catatAudit(tx, cl.UserID, "cuci_kirim", "washing_jobs", int64(id), gin.H{"pencuci": nama, "berat_awal": timbang})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("dikirim ke pencuci %s, berat awal %d barang tercatat", nama, len(items))})
	}
}

// ---------- PJ: SELESAI CUCI DENGAN BERAT SESUDAH ----------

func cuciSelesaiBarangV3(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaPJ(c)
		if !ok {
			return
		}
		var req struct {
			Items []struct {
				ItemID int     `json:"item_id"`
				Weight float64 `json:"weight_after"`
			} `json:"items"`
			FinishDate string `json:"finish_date"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.Items) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
		tgl, ok := parseTanggal(c, req.FinishDate)
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
			gagal(c, err)
			return
		}
		defer tx.Rollback()

		rows, err := tx.Query(
			`SELECT i.id, i.item_name, i.status, IFNULL(i.washing_job_id,0), IFNULL(w.status,''), i.weight_gram,
			        IFNULL(x.weight_before, i.weight_gram)
			 FROM items i
			 LEFT JOIN washing_jobs w ON w.id = i.washing_job_id
			 LEFT JOIN washing_job_items x ON x.job_id = i.washing_job_id AND x.item_id = i.id
			 WHERE i.is_void = FALSE AND i.id IN (`+placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			gagal(c, err)
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
				gagal(c, err)
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
				gagal(c, err)
				return
			}
			if _, err := tx.Exec(
				`UPDATE items SET weight_gram = ?, status = 'kembali', kembali_jenis = 'cuci_selesai',
				        belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?`, after, b.ID,
			); err != nil {
				gagal(c, err)
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
			catatAudit(tx, cl.UserID, "cuci_selesai", "washing_jobs", int64(job), gin.H{"sisa": sisa, "susut": totalSusut})
		}

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("%d barang selesai dicuci (susut %.3f g), dikembalikan ke back office dan menunggu konfirmasi owner", len(list), totalSusut),
		})
	}
}

// ---------- PJ: BATAL CUCI / LEBUR, BARANG MENUNGGU KONFIRMASI BACK OFFICE ----------

func cuciKembaliV2(db *sql.DB) gin.HandlerFunc {
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
			Reason string `json:"reason"`
			Date   string `json:"date"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan wajib diisi"})
			return
		}
		alasan := strings.TrimSpace(req.Reason)
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
		if isVoid || (status != "menunggu" && status != "proses") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pekerjaan cuci sudah selesai atau dibatalkan"})
			return
		}

		ids, err := ambilIDs(tx, "SELECT id FROM items WHERE washing_job_id = ? AND status = 'cuci' AND is_void = FALSE FOR UPDATE", id)
		if err != nil {
			gagal(c, err)
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
				gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			`UPDATE washing_jobs SET status = 'batal', cancel_reason = ?, cancelled_by = ?, cancelled_date = ?, cancel_returned = ?
			 WHERE id = ?`, batasTeks(alasan, 250), cl.UserID, tgl, len(ids), id,
		); err != nil {
			gagal(c, err)
			return
		}
		catatAudit(tx, cl.UserID, "cuci_batal_kembali", "washing_jobs", int64(id), gin.H{"alasan": alasan, "dikembalikan": len(ids)})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("cuci #%d dibatalkan, %d barang dikembalikan ke back office (menunggu konfirmasi owner)", id, len(ids)),
		})
	}
}

func leburKembaliV2(db *sql.DB) gin.HandlerFunc {
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
			Reason string `json:"reason"`
			Date   string `json:"date"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan wajib diisi"})
			return
		}
		alasan := strings.TrimSpace(req.Reason)
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
		err = tx.QueryRow("SELECT status, is_void FROM melt_batches WHERE id = ? FOR UPDATE", id).Scan(&status, &isVoid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "lebur tidak ditemukan"})
			return
		}
		if isVoid || status == "batal" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lebur sudah dibatalkan"})
			return
		}
		if status != "proses" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang sudah dilebur, tidak bisa dikembalikan. Kalau salah input, minta Super Admin membatalkan langkahnya"})
			return
		}

		ids, err := ambilIDs(tx, "SELECT id FROM items WHERE melt_id = ? AND jenis <> 'hasil_lebur' AND status = 'lebur' AND is_void = FALSE FOR UPDATE", id)
		if err != nil {
			gagal(c, err)
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam proses lebur"})
			return
		}
		for _, itemID := range ids {
			if _, err := tx.Exec(
				`UPDATE items SET status = 'kembali', kembali_jenis = 'lebur_batal', belum_sampai = FALSE, belum_sampai_note = NULL
				 WHERE id = ?`, itemID,
			); err != nil {
				gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			"UPDATE melt_batches SET status = 'batal', cancel_reason = ?, cancelled_by = ?, cancelled_date = ? WHERE id = ?",
			batasTeks(alasan, 250), cl.UserID, tgl, id,
		); err != nil {
			gagal(c, err)
			return
		}
		catatAudit(tx, cl.UserID, "lebur_batal_kembali", "melt_batches", int64(id), gin.H{"alasan": alasan, "dikembalikan": len(ids)})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("lebur #%d dibatalkan, %d barang dikembalikan ke back office (menunggu konfirmasi owner)", id, len(ids)),
		})
	}
}

// ---------- OWNER: KONFIRMASI BARANG KEMBALI DARI PJ ----------

func barangTerimaKembali(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req struct {
			ItemIDs []int  `json:"item_ids"`
			Result  string `json:"result"`
			Date    string `json:"date"`
			Note    string `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if req.Result != "sampai" && req.Result != "belum" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih sudah diterima atau belum diterima"})
			return
		}
		if len(req.ItemIDs) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
		tgl, ok := parseTanggal(c, req.Date)
		if !ok {
			return
		}
		note := strings.TrimSpace(req.Note)
		boID, err := idBackOffice(db)
		if err != nil {
			gagal(c, err)
			return
		}

		uniq := map[int]bool{}
		args := []interface{}{}
		for _, id := range req.ItemIDs {
			if !uniq[id] {
				uniq[id] = true
				args = append(args, id)
			}
		}

		tx, err := db.Begin()
		if err != nil {
			gagal(c, err)
			return
		}
		defer tx.Rollback()

		rows, err := tx.Query(
			`SELECT id, item_name, IFNULL(karat,''), weight_gram, status, IFNULL(kembali_jenis,'')
			 FROM items WHERE is_void = FALSE AND id IN (`+placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			gagal(c, err)
			return
		}
		type brg struct {
			ID     int
			Name   string
			Karat  string
			Weight float64
			Status string
			Jenis  string
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight, &b.Status, &b.Jenis); err != nil {
				rows.Close()
				gagal(c, err)
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
			if b.Status != "kembali" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak sedang dikembalikan ke back office", b.Name)})
				return
			}
		}

		if req.Result == "belum" {
			for _, b := range list {
				if _, err := tx.Exec("UPDATE items SET belum_sampai = TRUE, belum_sampai_note = ? WHERE id = ?", strNull(note), b.ID); err != nil {
					gagal(c, err)
					return
				}
			}
			catatAudit(tx, cl.UserID, "belum_diterima_kembali", "items", 0, req)
			if err := tx.Commit(); err != nil {
				gagal(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang ditandai belum diterima", len(list))})
			return
		}

		for _, b := range list {
			kategori, catatan := "batal_cuci", "Cuci dibatalkan, diterima back office"
			switch b.Jenis {
			case "cuci_selesai":
				kategori, catatan = "cucian", "Selesai cuci, diterima back office"
			case "lebur_batal":
				kategori, catatan = "batal_lebur", "Lebur dibatalkan, diterima back office"
			}
			if _, err := tx.Exec(sqlStokItem, boID, "in", kategori, b.Name, b.Weight, b.Karat, catatan, tgl, cl.UserID, b.ID); err != nil {
				gagal(c, err)
				return
			}
			if _, err := tx.Exec(
				`UPDATE items SET status = 'tersedia', kondisi = IF(? = 'cuci_selesai', 'bersih', kondisi),
				        kembali_jenis = NULL, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?`, b.Jenis, b.ID,
			); err != nil {
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "terima_kembali", "items", 0, req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang diterima di back office", len(list))})
	}
}

// ---------- DAFTAR BARANG DI PENCUCI (dengan berat awal) ----------

func cucianBarangV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := bolehLihatLebur(c); !ok {
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
			BeratAwal  float64    `json:"berat_awal"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var sent sql.NullTime
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram, &r.Asal,
				&r.JobID, &r.JobStatus, &r.Pencuci, &r.StartDate, &sent, &r.BeratAwal); err != nil {
				gagal(c, err)
				return
			}
			r.SentDate = ptrTime(sent)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- DAFTAR CUCI (dengan susut) ----------

func cuciListV4(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
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
			gagal(c, err)
			return
		}
		defer rows.Close()

		type row struct {
			ID             int        `json:"id"`
			LocationID     int        `json:"origin_location_id"`
			Cabang         string     `json:"cabang"`
			ItemName       string     `json:"item_name"`
			Karat          string     `json:"karat"`
			WeightGram     float64    `json:"weight_gram"`
			Qty            int        `json:"qty"`
			Washer         string     `json:"washer_name"`
			Status         string     `json:"status"`
			StartDate      time.Time  `json:"start_date"`
			FinishDate     *time.Time `json:"finish_date"`
			Note           string     `json:"note"`
			Sisa           int        `json:"sisa"`
			CancelReason   string     `json:"cancel_reason"`
			CancelReturned int        `json:"cancel_returned"`
			BeratAwalSelesai float64  `json:"berat_awal_selesai"`
			BeratSesudah   float64    `json:"berat_sesudah"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var fin sql.NullTime
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Cabang, &r.ItemName, &r.Karat, &r.WeightGram, &r.Qty,
				&r.Washer, &r.Status, &r.StartDate, &fin, &r.Note, &r.Sisa, &r.CancelReason, &r.CancelReturned,
				&r.BeratAwalSelesai, &r.BeratSesudah); err != nil {
				gagal(c, err)
				return
			}
			r.FinishDate = ptrTime(fin)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}