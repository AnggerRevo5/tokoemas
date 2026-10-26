// File ini berisi kode lama yang sudah tidak terdaftar di routes.
// Dibiarkan dulu sampai ada konfirmasi boleh dihapus.

package controllers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

type stokReq struct {
	LocationID int     `json:"location_id"`
	Direction  string  `json:"direction"`
	Category   string  `json:"category"`
	ItemName   string  `json:"item_name"`
	WeightGram float64 `json:"weight_gram"`
	Karat      string  `json:"karat"`
	Qty        int     `json:"qty"`
	Note       string  `json:"note"`
	Date       string  `json:"transaction_date"`
}

func LegacyStokCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req stokReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if !kategoriValid[tipe+"|stok|"+req.Direction][req.Category] {
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
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(
			`INSERT INTO stock_movements (location_id, direction, category, item_name, weight_gram, karat, qty, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?,?)`,
			locID, req.Direction, req.Category, req.ItemName, req.WeightGram, req.Karat, req.Qty, req.Note, tgl, cl.UserID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "create", "stock_movements", id, req)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "stok tersimpan"})
	}
}
