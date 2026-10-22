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

type barangInput struct {
	ItemName   string  `json:"item_name"`
	Karat      string  `json:"karat"`
	WeightGram float64 `json:"weight_gram"`
	Qty        int     `json:"qty"`
}

type barangReq struct {
	LocationID int           `json:"location_id"`
	Category   string        `json:"category"`
	EntryDate  string        `json:"entry_date"`
	Note       string        `json:"note"`
	Items      []barangInput `json:"items"`
}

func LegacyBarangCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req barangReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if !stokKategori[tipe+"|in"][req.Category] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kategori barang masuk tidak valid untuk lokasi ini"})
			return
		}
		if len(req.Items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi minimal satu barang"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.EntryDate)
		if !ok {
			return
		}

		total := 0
		for i := range req.Items {
			it := &req.Items[i]
			it.ItemName = strings.TrimSpace(it.ItemName)
			if it.Qty < 1 {
				it.Qty = 1
			}
			if it.ItemName == "" || it.WeightGram <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baris %d: nama dan berat wajib diisi", i+1)})
				return
			}
			total += it.Qty
		}
		if total > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali input"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var lastID int64
		for _, it := range req.Items {
			for n := 0; n < it.Qty; n++ {
				res, err := tx.Exec(
					`INSERT INTO items (item_name, karat, weight_gram, origin_location_id, location_id, source_category, entry_date, note, created_by)
					 VALUES (?,?,?,?,?,?,?,?,?)`,
					it.ItemName, it.Karat, it.WeightGram, locID, locID, req.Category, tgl, req.Note, cl.UserID,
				)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				id, _ := res.LastInsertId()
				lastID = id
				tx.Exec("UPDATE items SET item_no = ? WHERE id = ?", fmt.Sprintf("BRG-%05d", id), id)
				if _, err := tx.Exec(services.SQLStokItem, locID, "in", req.Category, it.ItemName, it.WeightGram, it.Karat,
					req.Note, tgl, cl.UserID, id); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
		services.CatatAudit(tx, cl.UserID, "create", "items", lastID, gin.H{"jumlah_barang": total, "request": req})

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("%d barang tersimpan", total), "jumlah": total})
	}
}

// ---------- DAFTAR BARANG ----------

func LegacyBarangList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)

		q := `SELECT i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), i.weight_gram,
		             i.location_id, l.name, i.origin_location_id, o.name, i.status, i.entry_date, IFNULL(i.note,'')
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
			ID         int       `json:"id"`
			ItemNo     string    `json:"item_no"`
			ItemName   string    `json:"item_name"`
			Karat      string    `json:"karat"`
			WeightGram float64   `json:"weight_gram"`
			LocationID int       `json:"location_id"`
			Lokasi     string    `json:"lokasi"`
			OriginID   int       `json:"origin_location_id"`
			Asal       string    `json:"asal"`
			Status     string    `json:"status"`
			EntryDate  time.Time `json:"entry_date"`
			Note       string    `json:"note"`
		}
		out := []row{}
		var totalGram float64
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram,
				&r.LocationID, &r.Lokasi, &r.OriginID, &r.Asal, &r.Status, &r.EntryDate, &r.Note); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			totalGram += r.WeightGram
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "jumlah": len(out), "total_gram": totalGram})
	}
}

// ---------- PINDAH BARANG (kirim ke toko / kembalikan ke back office) ----------

type pindahReq struct {
	ItemIDs          []int  `json:"item_ids"`
	Action           string `json:"action"`
	TargetLocationID int    `json:"target_location_id"`
	Date             string `json:"date"`
}

func LegacyBarangPindah(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req pindahReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.ItemIDs) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
		if req.Action != "kirim_toko" && req.Action != "kembali_bo" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "aksi tidak dikenal"})
			return
		}
		if req.Action == "kirim_toko" && cl.Role == "cabang" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		boID, err := services.IDBackOffice(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
			return
		}
		if req.Action == "kirim_toko" {
			if _, ok := services.CekCabang(db, req.TargetLocationID); !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "pilih toko tujuan (Kepanjen atau Mall Dinoyo)"})
				return
			}
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		items, err := services.AmbilBarang(tx, req.ItemIDs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// validasi semua barang dulu, baru proses
		for _, it := range items {
			if it.Status != "tersedia" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sedang tidak tersedia (%s)", it.Name, it.Status)})
				return
			}
			if req.Action == "kirim_toko" && it.Location != boID {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak berada di back office", it.Name)})
				return
			}
			if req.Action == "kembali_bo" {
				if it.Location == boID {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sudah di back office", it.Name)})
					return
				}
				if cl.Role == "cabang" && it.Location != cl.LocationID {
					c.JSON(http.StatusForbidden, gin.H{"error": "bukan barang di lokasi kamu"})
					return
				}
			}
		}

		for _, it := range items {
			var src, dst int
			var outCat, inCat, outNote, inNote string
			if req.Action == "kirim_toko" {
				src, dst = boID, req.TargetLocationID
				outCat, inCat = "ke_toko", "barang_masuk"
				outNote, inNote = "Kirim ke toko", "Kiriman dari Back Office"
			} else {
				src, dst = it.Location, boID
				outCat, inCat = "kembali_bo", "kembali_cabang"
				outNote, inNote = "Dikembalikan ke Back Office", "Dikembalikan dari cabang"
			}

			r1, err := tx.Exec(services.SQLStokItem, src, "out", outCat, it.Name, it.Weight, it.Karat, outNote, tgl, cl.UserID, it.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			r2, err := tx.Exec(services.SQLStokItem, dst, "in", inCat, it.Name, it.Weight, it.Karat, inNote, tgl, cl.UserID, it.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			outID, _ := r1.LastInsertId()
			inID, _ := r2.LastInsertId()
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", inID, outID)
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", outID, inID)

			if _, err := tx.Exec("UPDATE items SET location_id = ? WHERE id = ?", dst, it.ID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		services.CatatAudit(tx, cl.UserID, req.Action, "items", 0, req)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang dipindahkan", len(items))})
	}
}

func daftar(m map[int]models.ItemRow) []models.ItemRow {
	out := make([]models.ItemRow, 0, len(m))
	for _, it := range m {
		out = append(out, it)
	}
	return out
}

type cuciMulaiReq struct {
	ItemIDs    []int  `json:"item_ids"`
	WasherName string `json:"washer_name"`
	StartDate  string `json:"start_date"`
	Note       string `json:"note"`
}

func LegacyCuciMulai(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
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

		jobs := 0
		for origin, g := range groups {
			nama, berat, qty, karat := services.Ringkas(g)
			res, err := tx.Exec(
				`INSERT INTO washing_jobs (origin_location_id, item_name, weight_gram, karat, qty, washer_name, status, start_date, note, created_by)
				 VALUES (?,?,?,?,?,?,'proses',?,?,?)`,
				origin, nama, berat, karat, qty, washer, tgl, req.Note, cl.UserID,
			)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			jobID, _ := res.LastInsertId()
			for _, it := range g {
				if _, err := tx.Exec(services.SQLStokItem, boID, "out", "cuci", it.Name, it.Weight, it.Karat,
					"Cuci oleh "+washer, tgl, cl.UserID, it.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
				if _, err := tx.Exec("UPDATE items SET status = 'cuci', washing_job_id = ? WHERE id = ?", jobID, it.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			services.CatatAudit(tx, cl.UserID, "cuci_mulai", "washing_jobs", jobID, gin.H{"jumlah": len(g), "pencuci": washer})
			jobs++
		}

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("%d barang masuk proses cuci (%d pekerjaan)", len(items), jobs)})
	}
}

func LegacyCuciSelesaiItems(db *sql.DB) gin.HandlerFunc {
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
			utils.Gagal(c, err)
			return
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight, &b.Origin); err != nil {
				rows.Close()
				utils.Gagal(c, err)
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
			if _, err := tx.Exec(services.SQLStokItem, b.Origin, "in", cat, b.Name, b.Weight, b.Karat,
				"Selesai cuci", tgl, cl.UserID, b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'tersedia', location_id = ? WHERE id = ?", b.Origin, b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec("UPDATE washing_jobs SET status = 'selesai', finish_date = ? WHERE id = ?", tgl, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "cuci_selesai", "washing_jobs", int64(id), gin.H{"jumlah": len(list)})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("cuci selesai, %d barang kembali ke stok", len(list))})
	}
}

func LegacyCuciVoidItems(db *sql.DB) gin.HandlerFunc {
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
		if status != "proses" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cuci sudah selesai, tidak bisa dibatalkan"})
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

// ---------- LEBUR ----------

type leburBaruReq struct {
	RecordDate string  `json:"record_date"`
	ItemIDs    []int   `json:"item_ids"`
	MeltTotal  float64 `json:"melt_total"`
	SaleTotal  float64 `json:"sale_total"`
	Note       string  `json:"note"`
}

func LegacyLeburBaru(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
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
		tgl, ok := utils.ParseTanggal(c, req.RecordDate)
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
		nama, berat, qty, karat := services.Ringkas(list)

		res, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			boID, "in", "jual_barang", req.SaleTotal,
			fmt.Sprintf("Hasil lebur %d barang - asal %s", qty, namaAsal), tgl, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
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
			utils.Gagal(c, err)
			return
		}
		meltID, _ := res2.LastInsertId()

		for _, it := range list {
			if _, err := tx.Exec(services.SQLStokItem, boID, "out", "lebur", it.Name, it.Weight, it.Karat,
				"Lebur - asal "+namaAsal, tgl, cl.UserID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'lebur', melt_id = ? WHERE id = ?", meltID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatAudit(tx, cl.UserID, "create", "melt_records", meltID, req)

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":      meltID,
			"message": "lebur dicatat, kas back office bertambah",
			"selisih": req.SaleTotal - req.MeltTotal,
		})
	}
}

func LegacyLeburVoidItems(db *sql.DB) gin.HandlerFunc {
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

		ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE melt_id = ? AND status = 'lebur' FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
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
			services.CatatAudit(tx, cl.UserID, "void", "cash_transactions", cashID.Int64, gin.H{"reason": reason})
		}
		tx.Exec("UPDATE melt_records SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		services.CatatAudit(tx, cl.UserID, "void", "melt_records", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "lebur dibatalkan, barang dan kas dikembalikan"})
	}
}

// ---------- JUAL ----------

func LegacyBarangVoid(db *sql.DB) gin.HandlerFunc {
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
		services.CatatAudit(tx, cl.UserID, "void", "items", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang dibatalkan"})
	}
}

func LegacyBarangListV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)

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

// jumlah barang yang menunggu konfirmasi (untuk notifikasi)
func LegacyBarangNotif(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		q := "SELECT COUNT(*), IFNULL(SUM(belum_sampai), 0) FROM items WHERE is_void = FALSE AND status = 'dikirim'"
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND location_id = ?"
			args = append(args, cl.LocationID)
		}
		var menunggu, belum int
		if err := db.QueryRow(q, args...).Scan(&menunggu, &belum); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"menunggu": menunggu, "belum_sampai": belum})
	}
}
