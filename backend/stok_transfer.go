package main

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

var stokKategori = map[string]map[string]bool{
	"cabang|in":       {"barang_masuk": true},
	"cabang|out":      {"laku": true, "pegadaian": true, "cuci": true, "lebur": true, "jual": true, "kembali_bo": true},
	"back_office|in":  {"pabrik": true, "konsinyasi": true, "cucian": true, "barang_toko": true},
	"back_office|out": {"ke_toko": true, "lebur": true, "jual": true, "cuci": true},
}

type stokReq2 struct {
	LocationID       int     `json:"location_id"`
	TargetLocationID int     `json:"target_location_id"`
	Direction        string  `json:"direction"`
	Category         string  `json:"category"`
	ItemName         string  `json:"item_name"`
	WeightGram       float64 `json:"weight_gram"`
	Karat            string  `json:"karat"`
	Qty              int     `json:"qty"`
	Note             string  `json:"note"`
	Date             string  `json:"transaction_date"`
}

func stokCreateV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		var req stokReq2
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if !stokKategori[tipe+"|"+req.Direction][req.Category] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "arah atau kategori tidak valid untuk lokasi ini"})
			return
		}
		if req.ItemName == "" || req.WeightGram <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama barang dan berat (gram) wajib diisi"})
			return
		}
		if req.Qty < 1 {
			req.Qty = 1
		}
		tgl, ok := parseTanggal(c, req.Date)
		if !ok {
			return
		}

		// Tentukan pasangan otomatis (kalau ada)
		targetID := 0
		targetKat := ""
		targetNote := ""
		switch req.Category {
		case "ke_toko":
			var tt string
			err := db.QueryRow("SELECT type FROM locations WHERE id = ?", req.TargetLocationID).Scan(&tt)
			if err != nil || tt != "cabang" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "pilih toko tujuan (Kepanjen atau Mall Dinoyo)"})
				return
			}
			targetID = req.TargetLocationID
			targetKat = "barang_masuk"
			targetNote = "Kiriman dari Back Office"
		case "kembali_bo":
			if err := db.QueryRow("SELECT id FROM locations WHERE type='back_office' LIMIT 1").Scan(&targetID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
				return
			}
			targetKat = "kembali_cabang"
			targetNote = "Dikembalikan dari cabang"
		}
		if targetID != 0 && req.Note != "" {
			targetNote = targetNote + " - " + req.Note
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		const q = `INSERT INTO stock_movements
			(location_id, direction, category, item_name, weight_gram, karat, qty, note, transaction_date, created_by)
			VALUES (?,?,?,?,?,?,?,?,?,?)`

		res, err := tx.Exec(q, locID, req.Direction, req.Category, req.ItemName, req.WeightGram, req.Karat, req.Qty, req.Note, tgl, cl.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		catatAudit(tx, cl.UserID, "create", "stock_movements", id, req)

		msg := "stok tersimpan"
		if targetID != 0 {
			res2, err := tx.Exec(q, targetID, "in", targetKat, req.ItemName, req.WeightGram, req.Karat, req.Qty, targetNote, tgl, cl.UserID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			id2, _ := res2.LastInsertId()
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", id2, id)
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", id, id2)
			catatAudit(tx, cl.UserID, "create_auto", "stock_movements", id2, gin.H{"dari_stok_id": id})
			msg = "stok tersimpan dan otomatis tercatat di tujuan"
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": msg})
	}
}