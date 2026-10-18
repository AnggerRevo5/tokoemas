package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func batasTeks(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// ---------- PJ: CUCI TIDAK JADI, BARANG KEMBALI KE BACK OFFICE ----------

func cuciKembali(db *sql.DB) gin.HandlerFunc {
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
		if isVoid || (status != "menunggu" && status != "proses") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pekerjaan cuci sudah selesai atau dibatalkan"})
			return
		}

		rows, err := tx.Query(
			`SELECT id, item_name, IFNULL(karat,''), weight_gram FROM items
			 WHERE washing_job_id = ? AND status = 'cuci' AND is_void = FALSE FOR UPDATE`, id)
		if err != nil {
			gagal(c, err)
			return
		}
		type brg struct {
			ID     int
			Name   string
			Karat  string
			Weight float64
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight); err != nil {
				rows.Close()
				gagal(c, err)
				return
			}
			list = append(list, b)
		}
		rows.Close()
		if len(list) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang yang masih di pencuci"})
			return
		}

		catatan := "Cuci dibatalkan: " + batasTeks(alasan, 200)
		for _, b := range list {
			if _, err := tx.Exec(sqlStokItem, boID, "in", "batal_cuci", b.Name, b.Weight, b.Karat,
				catatan, tgl, cl.UserID, b.ID); err != nil {
				gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'tersedia' WHERE id = ?", b.ID); err != nil {
				gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			`UPDATE washing_jobs SET status = 'batal', cancel_reason = ?, cancelled_by = ?, cancelled_date = ?, cancel_returned = ?
			 WHERE id = ?`, batasTeks(alasan, 250), cl.UserID, tgl, len(list), id,
		); err != nil {
			gagal(c, err)
			return
		}
		catatAudit(tx, cl.UserID, "cuci_batal_kembali", "washing_jobs", int64(id), gin.H{"alasan": alasan, "dikembalikan": len(list)})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("cuci #%d dibatalkan, %d barang kembali tersedia di back office", id, len(list))})
	}
}

// daftar cuci dengan keterangan pembatalan
func cuciListV3(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		q := `SELECT w.id, w.origin_location_id, l.name, w.item_name, IFNULL(w.karat,''), w.weight_gram, w.qty,
		             w.washer_name, w.status, w.start_date, w.finish_date, IFNULL(w.note,''),
		             (SELECT COUNT(*) FROM items i WHERE i.washing_job_id = w.id AND i.status = 'cuci' AND i.is_void = FALSE),
		             IFNULL(w.cancel_reason,''), IFNULL(w.cancel_returned,0)
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
		}
		out := []row{}
		for rows.Next() {
			var r row
			var fin sql.NullTime
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Cabang, &r.ItemName, &r.Karat, &r.WeightGram, &r.Qty,
				&r.Washer, &r.Status, &r.StartDate, &fin, &r.Note, &r.Sisa, &r.CancelReason, &r.CancelReturned); err != nil {
				gagal(c, err)
				return
			}
			r.FinishDate = ptrTime(fin)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- PJ: LEBUR TIDAK JADI, BARANG KEMBALI KE BACK OFFICE ----------

func leburKembali(db *sql.DB) gin.HandlerFunc {
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

		rows, err := tx.Query(
			`SELECT id, item_name, IFNULL(karat,''), weight_gram FROM items
			 WHERE melt_id = ? AND jenis <> 'hasil_lebur' AND status = 'lebur' AND is_void = FALSE FOR UPDATE`, id)
		if err != nil {
			gagal(c, err)
			return
		}
		type brg struct {
			ID     int
			Name   string
			Karat  string
			Weight float64
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight); err != nil {
				rows.Close()
				gagal(c, err)
				return
			}
			list = append(list, b)
		}
		rows.Close()
		if len(list) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam proses lebur"})
			return
		}

		catatan := "Lebur dibatalkan: " + batasTeks(alasan, 200)
		for _, b := range list {
			if _, err := tx.Exec(sqlStokItem, boID, "in", "batal_lebur", b.Name, b.Weight, b.Karat,
				catatan, tgl, cl.UserID, b.ID); err != nil {
				gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'tersedia' WHERE id = ?", b.ID); err != nil {
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
		catatAudit(tx, cl.UserID, "lebur_batal_kembali", "melt_batches", int64(id), gin.H{"alasan": alasan, "dikembalikan": len(list)})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("lebur #%d dibatalkan, %d barang kembali tersedia di back office", id, len(list))})
	}
}

// ---------- PJ: BATALKAN PENJUALAN (24 jam), SUPER ADMIN TANPA BATAS ----------

func penjualanBatal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		if cl.Role != "pj" && cl.Role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
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

		var isVoid, recent bool
		var cashID sql.NullInt64
		err = tx.QueryRow(
			"SELECT is_void, cash_id, created_at >= NOW() - INTERVAL 1 DAY FROM penadah_sales WHERE id = ? FOR UPDATE", id,
		).Scan(&isVoid, &cashID, &recent)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "penjualan tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}
		if cl.Role == "pj" && !recent {
			c.JSON(http.StatusForbidden, gin.H{"error": "lebih dari 24 jam, minta Super Admin yang membatalkan"})
			return
		}

		ids, err := ambilIDs(tx, "SELECT id FROM items WHERE sale_id = ? FOR UPDATE", id)
		if err != nil {
			gagal(c, err)
			return
		}
		for _, itemID := range ids {
			var jenis string
			tx.QueryRow("SELECT jenis FROM items WHERE id = ?", itemID).Scan(&jenis)
			st := "tersedia"
			if jenis == "hasil_lebur" {
				st = "hasil"
			}
			tx.Exec("UPDATE items SET status = ?, sale_id = NULL WHERE id = ?", st, itemID)
			tx.Exec(`UPDATE stock_movements SET is_void = TRUE, void_reason = ?
			         WHERE item_id = ? AND category = 'jual_penadah' AND direction = 'out' AND is_void = FALSE
			         ORDER BY id DESC LIMIT 1`, reason, itemID)
			if jenis == "hasil_lebur" {
				tx.Exec("UPDATE melt_batches SET status = 'cukim' WHERE cukim_item_id = ? AND is_void = FALSE", itemID)
			}
		}
		if cashID.Valid {
			tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, cashID.Int64)
			catatAudit(tx, cl.UserID, "void", "cash_transactions", cashID.Int64, gin.H{"reason": reason})
		}
		tx.Exec("UPDATE penadah_sales SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		catatAudit(tx, cl.UserID, "penjualan_batal", "penadah_sales", int64(id), gin.H{"reason": reason, "oleh": cl.Role})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "penjualan dibatalkan, barang kembali siap dijual dan kas dikembalikan"})
	}
}