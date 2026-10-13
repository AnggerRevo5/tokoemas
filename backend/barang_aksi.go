package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func gagal(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func bacaAlasan(c *gin.Context) (string, bool) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alasan pembatalan wajib diisi"})
		return "", false
	}
	return req.Reason, true
}

func ringkas(items []itemRow) (string, float64, int, string) {
	nama, karat := "", ""
	var berat float64
	samaNama, samaKarat := true, true
	for i, it := range items {
		berat += it.Weight
		if i == 0 {
			nama, karat = it.Name, it.Karat
			continue
		}
		if it.Name != nama {
			samaNama = false
		}
		if it.Karat != karat {
			samaKarat = false
		}
	}
	if !samaNama {
		nama = "Aneka barang"
	}
	if !samaKarat {
		karat = ""
	}
	return nama, berat, len(items), karat
}

func kunci(m map[int]itemRow) []int {
	out := make([]int, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	return out
}

func daftar(m map[int]itemRow) []itemRow {
	out := make([]itemRow, 0, len(m))
	for _, it := range m {
		out = append(out, it)
	}
	return out
}

func ambilIDs(tx *sql.Tx, q string, args ...interface{}) ([]int, error) {
	rows, err := tx.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func ambilAsal(tx *sql.Tx, ids []int) (map[int]int, error) {
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := tx.Query("SELECT id, origin_location_id FROM items WHERE id IN ("+placeholders(len(args))+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]int{}
	for rows.Next() {
		var id, o int
		if err := rows.Scan(&id, &o); err != nil {
			return nil, err
		}
		out[id] = o
	}
	return out, nil
}

// semua barang harus tersedia dan berada di back office
func barangDiBO(c *gin.Context, tx *sql.Tx, ids []int, boID int) (map[int]itemRow, bool) {
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
		return nil, false
	}
	if len(ids) > 300 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
		return nil, false
	}
	items, err := ambilBarang(tx, ids)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return nil, false
	}
	for _, it := range items {
		if it.Status != "tersedia" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sedang tidak tersedia (%s)", it.Name, it.Status)})
			return nil, false
		}
		if it.Location != boID {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak berada di back office", it.Name)})
			return nil, false
		}
	}
		cekArgs := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		cekArgs = append(cekArgs, id)
	}
	var lm int
	tx.QueryRow("SELECT COUNT(*) FROM items WHERE jenis = 'logam_mulia' AND id IN ("+placeholders(len(cekArgs))+")", cekArgs...).Scan(&lm)
	if lm > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "logam mulia tidak bisa dicuci atau dilebur, langsung dijual saja"})
		return nil, false
	}
	return items, true
}

// ---------- CUCI ----------

type cuciMulaiReq struct {
	ItemIDs    []int  `json:"item_ids"`
	WasherName string `json:"washer_name"`
	StartDate  string `json:"start_date"`
	Note       string `json:"note"`
}

func cuciMulai(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req cuciMulaiReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		washer := strings.TrimSpace(req.WasherName)
		if washer == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama pencuci wajib diisi"})
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

		jobs := 0
		for origin, g := range groups {
			nama, berat, qty, karat := ringkas(g)
			res, err := tx.Exec(
				`INSERT INTO washing_jobs (origin_location_id, item_name, weight_gram, karat, qty, washer_name, status, start_date, note, created_by)
				 VALUES (?,?,?,?,?,?,'proses',?,?,?)`,
				origin, nama, berat, karat, qty, washer, tgl, req.Note, cl.UserID,
			)
			if err != nil {
				gagal(c, err)
				return
			}
			jobID, _ := res.LastInsertId()
			for _, it := range g {
				if _, err := tx.Exec(sqlStokItem, boID, "out", "cuci", it.Name, it.Weight, it.Karat,
					"Cuci oleh "+washer, tgl, cl.UserID, it.ID); err != nil {
					gagal(c, err)
					return
				}
				if _, err := tx.Exec("UPDATE items SET status = 'cuci', washing_job_id = ? WHERE id = ?", jobID, it.ID); err != nil {
					gagal(c, err)
					return
				}
			}
			catatAudit(tx, cl.UserID, "cuci_mulai", "washing_jobs", jobID, gin.H{"jumlah": len(g), "pencuci": washer})
			jobs++
		}

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("%d barang masuk proses cuci (%d pekerjaan)", len(items), jobs)})
	}
}

func cuciSelesaiItems(db *sql.DB) gin.HandlerFunc {
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

		type brg struct {
			ID     int
			Name   string
			Karat  string
			Weight float64
			Origin int
		}
		rows, err := tx.Query(
			`SELECT id, item_name, IFNULL(karat,''), weight_gram, origin_location_id
			 FROM items WHERE washing_job_id = ? AND status = 'cuci' FOR UPDATE`, id)
		if err != nil {
			gagal(c, err)
			return
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight, &b.Origin); err != nil {
				rows.Close()
				gagal(c, err)
				return
			}
			list = append(list, b)
		}
		rows.Close()
		if len(list) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam proses cuci"})
			return
		}

		for _, b := range list {
			cat := "barang_masuk"
			if b.Origin == boID {
				cat = "cucian"
			}
			if _, err := tx.Exec(sqlStokItem, b.Origin, "in", cat, b.Name, b.Weight, b.Karat,
				"Selesai cuci", tgl, cl.UserID, b.ID); err != nil {
				gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'tersedia', location_id = ? WHERE id = ?", b.Origin, b.ID); err != nil {
				gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec("UPDATE washing_jobs SET status = 'selesai', finish_date = ? WHERE id = ?", tgl, id); err != nil {
			gagal(c, err)
			return
		}
		catatAudit(tx, cl.UserID, "cuci_selesai", "washing_jobs", int64(id), gin.H{"jumlah": len(list)})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("cuci selesai, %d barang kembali ke stok", len(list))})
	}
}

func cuciVoidItems(db *sql.DB) gin.HandlerFunc {
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
		if status != "proses" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cuci sudah selesai, tidak bisa dibatalkan"})
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

// ---------- LEBUR ----------

type leburBaruReq struct {
	RecordDate string  `json:"record_date"`
	ItemIDs    []int   `json:"item_ids"`
	MeltTotal  float64 `json:"melt_total"`
	SaleTotal  float64 `json:"sale_total"`
	Note       string  `json:"note"`
}

func leburBaru(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req leburBaruReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		if req.MeltTotal <= 0 || req.SaleTotal <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "total lebur dan total harga jual harus lebih dari 0"})
			return
		}
		tgl, ok := parseTanggal(c, req.RecordDate)
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
		origin := 0
		for _, o := range asal {
			if origin == 0 {
				origin = o
			} else if o != origin {
				c.JSON(http.StatusBadRequest, gin.H{"error": "pilih barang dari satu cabang asal saja untuk tiap lebur"})
				return
			}
		}
		var namaAsal string
		tx.QueryRow("SELECT name FROM locations WHERE id = ?", origin).Scan(&namaAsal)

		list := daftar(items)
		nama, berat, qty, karat := ringkas(list)

		res, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			boID, "in", "jual_barang", req.SaleTotal,
			fmt.Sprintf("Hasil lebur %d barang - asal %s", qty, namaAsal), tgl, cl.UserID,
		)
		if err != nil {
			gagal(c, err)
			return
		}
		cashID, _ := res.LastInsertId()

		res2, err := tx.Exec(
			`INSERT INTO melt_records
			 (record_date, origin_location_id, item_name, karat, weight_gram, qty, melt_total, sale_total, note, cash_id, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			tgl, origin, nama, karat, berat, qty, req.MeltTotal, req.SaleTotal, req.Note, cashID, cl.UserID,
		)
		if err != nil {
			gagal(c, err)
			return
		}
		meltID, _ := res2.LastInsertId()

		for _, it := range list {
			if _, err := tx.Exec(sqlStokItem, boID, "out", "lebur", it.Name, it.Weight, it.Karat,
				"Lebur - asal "+namaAsal, tgl, cl.UserID, it.ID); err != nil {
				gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'lebur', melt_id = ? WHERE id = ?", meltID, it.ID); err != nil {
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "create", "melt_records", meltID, req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":      meltID,
			"message": "lebur dicatat, kas back office bertambah",
			"selisih": req.SaleTotal - req.MeltTotal,
		})
	}
}

func leburVoidItems(db *sql.DB) gin.HandlerFunc {
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

		var isVoid bool
		var cashID sql.NullInt64
		err = tx.QueryRow("SELECT is_void, cash_id FROM melt_records WHERE id = ? FOR UPDATE", id).Scan(&isVoid, &cashID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "catatan lebur tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}

		ids, err := ambilIDs(tx, "SELECT id FROM items WHERE melt_id = ? AND status = 'lebur' FOR UPDATE", id)
		if err != nil {
			gagal(c, err)
			return
		}
		for _, itemID := range ids {
			tx.Exec(`UPDATE stock_movements SET is_void = TRUE, void_reason = ?
			         WHERE item_id = ? AND category = 'lebur' AND direction = 'out' AND is_void = FALSE
			         ORDER BY id DESC LIMIT 1`, reason, itemID)
			tx.Exec("UPDATE items SET status = 'tersedia', melt_id = NULL WHERE id = ?", itemID)
		}
		if cashID.Valid {
			tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, cashID.Int64)
			catatAudit(tx, cl.UserID, "void", "cash_transactions", cashID.Int64, gin.H{"reason": reason})
		}
		tx.Exec("UPDATE melt_records SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		catatAudit(tx, cl.UserID, "void", "melt_records", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "lebur dibatalkan, barang dan kas dikembalikan"})
	}
}

// ---------- JUAL ----------

type jualReq struct {
	ItemIDs []int  `json:"item_ids"`
	Date    string `json:"date"`
	Note    string `json:"note"`
}

func barangJual(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		var req jualReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
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
			if it.Status != "tersedia" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sedang tidak tersedia (%s)", it.Name, it.Status)})
				return
			}
			if cl.Role == "cabang" && it.Location != cl.LocationID {
				c.JSON(http.StatusForbidden, gin.H{"error": "bukan barang di lokasi kamu"})
				return
			}
		}

		note := req.Note
		if note == "" {
			note = "Terjual"
		}
		for _, it := range items {
			cat := "laku"
			if it.Location == boID {
				cat = "jual"
			}
			if _, err := tx.Exec(sqlStokItem, it.Location, "out", cat, it.Name, it.Weight, it.Karat, note, tgl, cl.UserID, it.ID); err != nil {
				gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'terjual' WHERE id = ?", it.ID); err != nil {
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "jual", "items", 0, req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang terjual (omzet dicatat di menu Kas)", len(items))})
	}
}

// ---------- BATALKAN BARANG MASUK ----------

func barangVoid(db *sql.DB) gin.HandlerFunc {
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
		err = tx.QueryRow("SELECT status, is_void FROM items WHERE id = ? FOR UPDATE", id).Scan(&status, &isVoid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "barang tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}
		if status != "tersedia" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang sedang tidak tersedia"})
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

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang dibatalkan"})
	}
}