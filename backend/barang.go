package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

const sqlStokItem = `INSERT INTO stock_movements
	(location_id, direction, category, item_name, weight_gram, karat, qty, note, transaction_date, created_by, item_id)
	VALUES (?,?,?,?,?,?,1,?,?,?,?)`

// ---------- INPUT BARANG MASUK (banyak sekaligus) ----------

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

func barangCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
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
		tgl, ok := parseTanggal(c, req.EntryDate)
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
				if _, err := tx.Exec(sqlStokItem, locID, "in", req.Category, it.ItemName, it.WeightGram, it.Karat,
					req.Note, tgl, cl.UserID, id); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}
		catatAudit(tx, cl.UserID, "create", "items", lastID, gin.H{"jumlah_barang": total, "request": req})

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("%d barang tersimpan", total), "jumlah": total})
	}
}

// ---------- DAFTAR BARANG ----------

func barangList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)

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
			ID           int       `json:"id"`
			ItemNo       string    `json:"item_no"`
			ItemName     string    `json:"item_name"`
			Karat        string    `json:"karat"`
			WeightGram   float64   `json:"weight_gram"`
			LocationID   int       `json:"location_id"`
			Lokasi       string    `json:"lokasi"`
			OriginID     int       `json:"origin_location_id"`
			Asal         string    `json:"asal"`
			Status       string    `json:"status"`
			EntryDate    time.Time `json:"entry_date"`
			Note         string    `json:"note"`
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

type itemRow struct {
	ID       int
	Name     string
	Karat    string
	Weight   float64
	Location int
	Status   string
}

// ambil dan kunci barang yang dipilih; semua id harus ada
func ambilBarang(tx *sql.Tx, ids []int) (map[int]itemRow, error) {
	uniq := map[int]bool{}
	args := []interface{}{}
	for _, id := range ids {
		if !uniq[id] {
			uniq[id] = true
			args = append(args, id)
		}
	}
	rows, err := tx.Query(
		`SELECT id, item_name, IFNULL(karat,''), weight_gram, location_id, status
		 FROM items WHERE is_void = FALSE AND id IN (`+placeholders(len(args))+`) FOR UPDATE`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]itemRow{}
	for rows.Next() {
		var r itemRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Karat, &r.Weight, &r.Location, &r.Status); err != nil {
			return nil, err
		}
		out[r.ID] = r
	}
	if len(out) != len(args) {
		return nil, fmt.Errorf("ada barang yang tidak ditemukan")
	}
	return out, nil
}

func barangPindah(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
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
		tgl, ok := parseTanggal(c, req.Date)
		if !ok {
			return
		}
		boID, err := idBackOffice(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
			return
		}
		if req.Action == "kirim_toko" {
			if _, ok := cekCabang(db, req.TargetLocationID); !ok {
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

		items, err := ambilBarang(tx, req.ItemIDs)
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

			r1, err := tx.Exec(sqlStokItem, src, "out", outCat, it.Name, it.Weight, it.Karat, outNote, tgl, cl.UserID, it.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			r2, err := tx.Exec(sqlStokItem, dst, "in", inCat, it.Name, it.Weight, it.Karat, inNote, tgl, cl.UserID, it.ID)
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
		catatAudit(tx, cl.UserID, req.Action, "items", 0, req)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang dipindahkan", len(items))})
	}
}