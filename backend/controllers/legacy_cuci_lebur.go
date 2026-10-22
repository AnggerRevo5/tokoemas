// File ini berisi kode lama yang sudah tidak terdaftar di routes.
// Dibiarkan dulu sampai ada konfirmasi boleh dihapus.

package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

func LegacyCuciKembali(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
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
		tgl, ok := utils.ParseTanggal(c, req.Date)
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
			utils.Gagal(c, err)
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
				utils.Gagal(c, err)
				return
			}
			list = append(list, b)
		}
		rows.Close()
		if len(list) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang yang masih di pencuci"})
			return
		}

		catatan := "Cuci dibatalkan: " + utils.BatasTeks(alasan, 200)
		for _, b := range list {
			if _, err := tx.Exec(services.SQLStokItem, boID, "in", "batal_cuci", b.Name, b.Weight, b.Karat,
				catatan, tgl, cl.UserID, b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'tersedia' WHERE id = ?", b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			`UPDATE washing_jobs SET status = 'batal', cancel_reason = ?, cancelled_by = ?, cancelled_date = ?, cancel_returned = ?
			 WHERE id = ?`, utils.BatasTeks(alasan, 250), cl.UserID, tgl, len(list), id,
		); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "cuci_batal_kembali", "washing_jobs", int64(id), gin.H{"alasan": alasan, "dikembalikan": len(list)})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("cuci #%d dibatalkan, %d barang kembali tersedia di back office", id, len(list))})
	}
}

// daftar cuci dengan keterangan pembatalan
func LegacyCuciListV3(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
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
			utils.Gagal(c, err)
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
				utils.Gagal(c, err)
				return
			}
			r.FinishDate = utils.PtrTime(fin)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- PJ: LEBUR TIDAK JADI, BARANG KEMBALI KE BACK OFFICE ----------

func LegacyLeburKembali(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
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
		tgl, ok := utils.ParseTanggal(c, req.Date)
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
			utils.Gagal(c, err)
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
				utils.Gagal(c, err)
				return
			}
			list = append(list, b)
		}
		rows.Close()
		if len(list) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam proses lebur"})
			return
		}

		catatan := "Lebur dibatalkan: " + utils.BatasTeks(alasan, 200)
		for _, b := range list {
			if _, err := tx.Exec(services.SQLStokItem, boID, "in", "batal_lebur", b.Name, b.Weight, b.Karat,
				catatan, tgl, cl.UserID, b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'tersedia' WHERE id = ?", b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			"UPDATE melt_batches SET status = 'batal', cancel_reason = ?, cancelled_by = ?, cancelled_date = ? WHERE id = ?",
			utils.BatasTeks(alasan, 250), cl.UserID, tgl, id,
		); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "lebur_batal_kembali", "melt_batches", int64(id), gin.H{"alasan": alasan, "dikembalikan": len(list)})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("lebur #%d dibatalkan, %d barang kembali tersedia di back office", id, len(list))})
	}
}

// ---------- PJ: BATALKAN PENJUALAN (24 jam), SUPER ADMIN TANPA BATAS ----------

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

func LegacyCuciCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req cuciReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		namaCabang, ok := services.CekCabang(db, req.OriginLocationID)
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
		tgl, ok := utils.ParseTanggal(c, req.StartDate)
		if !ok {
			return
		}
		boID, err := services.IDBackOffice(db)
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
		services.CatatAudit(tx, cl.UserID, "create", "washing_jobs", id, req)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "cuci dicatat, status proses"})
	}
}

func LegacyCuciSelesai(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req struct {
			FinishDate string `json:"finish_date"`
		}
		c.ShouldBindJSON(&req) // body boleh kosong
		tgl, ok := utils.ParseTanggal(c, req.FinishDate)
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
		services.CatatAudit(tx, cl.UserID, "cuci_selesai", "washing_jobs", int64(id), gin.H{"finish_date": tgl, "in_stock_id": inID})

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cuci selesai, barang masuk ke stok toko"})
	}
}

func LegacyCuciList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
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

func LegacyCuciVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
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
		services.CatatAudit(tx, cl.UserID, "void", "washing_jobs", int64(id), gin.H{"reason": req.Reason})
		for _, s := range []sql.NullInt64{outID, inID} {
			if s.Valid {
				tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, s.Int64)
				services.CatatAudit(tx, cl.UserID, "void", "stock_movements", s.Int64, gin.H{"reason": req.Reason})
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

func LegacyLeburCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req leburReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		namaCabang, ok := services.CekCabang(db, req.OriginLocationID)
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
		tgl, ok := utils.ParseTanggal(c, req.RecordDate)
		if !ok {
			return
		}
		boID, err := services.IDBackOffice(db)
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
		services.CatatAudit(tx, cl.UserID, "create", "melt_records", id, req)

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

func LegacyLeburList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
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

func LegacyLeburVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
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
		services.CatatAudit(tx, cl.UserID, "void", "melt_records", int64(id), gin.H{"reason": req.Reason})
		if stockID.Valid {
			tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, stockID.Int64)
			services.CatatAudit(tx, cl.UserID, "void", "stock_movements", stockID.Int64, gin.H{"reason": req.Reason})
		}
		if cashID.Valid {
			tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, cashID.Int64)
			services.CatatAudit(tx, cl.UserID, "void", "cash_transactions", cashID.Int64, gin.H{"reason": req.Reason})
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "lebur dibatalkan, stok dan kas dikembalikan"})
	}
}

func LegacyCuciKirim(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
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
		if _, err := tx.Exec(
			"UPDATE washing_jobs SET status = 'proses', washer_name = ?, washer_sent_date = ?, sent_by = ? WHERE id = ?",
			nama, tgl, cl.UserID, id,
		); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "cuci_kirim", "washing_jobs", int64(id), gin.H{"pencuci": nama})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("dikirim ke pencuci %s", nama)})
	}
}

// ---------- PJ: SELESAI CUCI ----------

func LegacyCuciSelesaiJobV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req struct {
			FinishDate string `json:"finish_date"`
		}
		c.ShouldBindJSON(&req)
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
		ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE washing_job_id = ? AND status = 'cuci' FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam proses cuci"})
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
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("cuci selesai, %d barang kembali ke stok back office (bersih)", len(ids))})
	}
}

// ---------- DAFTAR BARANG DI PENCUCI (untuk PJ) ----------

func LegacyCucianBarang(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.BolehLihatLebur(c); !ok {
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
			utils.Gagal(c, err)
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
				utils.Gagal(c, err)
				return
			}
			r.SentDate = utils.PtrTime(sent)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- ANGKA MERAH UNTUK PJ ----------

// selesai untuk barang yang dicentang
func LegacyCuciSelesaiBarang(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
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
		if err := services.SelesaiCuci(tx, services.Kunci(items), tgl, cl.UserID, boID); err != nil {
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

// selesai untuk semua barang yang tersisa di satu pekerjaan cuci
func LegacyCuciSelesaiJob(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req struct {
			FinishDate string `json:"finish_date"`
		}
		c.ShouldBindJSON(&req)
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
		ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE washing_job_id = ? AND status = 'cuci' FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam proses cuci"})
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
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("cuci selesai, %d barang kembali ke stok back office (bersih)", len(ids))})
	}
}

// daftar pekerjaan cuci beserta progres
func LegacyCuciListV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
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
			utils.Gagal(c, err)
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
				utils.Gagal(c, err)
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
