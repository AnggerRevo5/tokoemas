package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type kirimBOReq struct {
	ItemIDs []int  `json:"item_ids"`
	Date    string `json:"date"`
}

// toko mengantar barang ke back office (status dalam perjalanan)
func barangKirimBO(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		var req kirimBOReq
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
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak bisa diantar (%s)", it.Name, it.Status)})
				return
			}
			if it.Location == boID {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sudah di back office", it.Name)})
				return
			}
			if cl.Role == "cabang" && it.Location != cl.LocationID {
				c.JSON(http.StatusForbidden, gin.H{"error": "bukan barang di lokasi kamu"})
				return
			}
		}
		for _, it := range items {
			if _, err := tx.Exec(
				"UPDATE items SET status = 'dikirim', kirim_tgl = ?, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?",
				tgl, it.ID,
			); err != nil {
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "kirim_bo", "items", 0, req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang diantar ke back office, menunggu konfirmasi", len(items))})
	}
}

type terimaReq struct {
	ItemIDs []int  `json:"item_ids"`
	Result  string `json:"result"` // sampai atau belum
	Date    string `json:"date"`
	Note    string `json:"note"`
}

// back office mengonfirmasi: sudah sampai atau belum sampai
func barangTerima(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req terimaReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if req.Result != "sampai" && req.Result != "belum" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih sudah sampai atau belum sampai"})
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
		note := strings.TrimSpace(req.Note)

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
			if it.Status != "dikirim" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak sedang dalam perjalanan", it.Name)})
				return
			}
		}
		{
			idsArah := kunci(items)
			argsArah := make([]interface{}, 0, len(idsArah))
			for _, id := range idsArah {
				argsArah = append(argsArah, id)
			}
			var arahLain int
			tx.QueryRow(
				"SELECT COUNT(*) FROM items WHERE id IN ("+placeholders(len(argsArah))+") AND kirim_ke_location_id IS NOT NULL",
				argsArah...,
			).Scan(&arahLain)
			if arahLain > 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "barang ini sedang menuju toko, konfirmasinya oleh akun toko tujuan"})
				return
			}
		}

		if req.Result == "belum" {
			for _, it := range items {
				if _, err := tx.Exec(
					"UPDATE items SET belum_sampai = TRUE, belum_sampai_note = ? WHERE id = ?",
					sql.NullString{String: note, Valid: note != ""}, it.ID,
				); err != nil {
					gagal(c, err)
					return
				}
			}
			catatAudit(tx, cl.UserID, "belum_sampai", "items", 0, req)
			if err := tx.Commit(); err != nil {
				gagal(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang ditandai belum sampai", len(items))})
			return
		}

		for _, it := range items {
			r1, err := tx.Exec(sqlStokItem, it.Location, "out", "kembali_bo", it.Name, it.Weight, it.Karat,
				"Diantar ke Back Office", tgl, cl.UserID, it.ID)
			if err != nil {
				gagal(c, err)
				return
			}
			r2, err := tx.Exec(sqlStokItem, boID, "in", "kembali_cabang", it.Name, it.Weight, it.Karat,
				"Diterima di Back Office", tgl, cl.UserID, it.ID)
			if err != nil {
				gagal(c, err)
				return
			}
			outID, _ := r1.LastInsertId()
			inID, _ := r2.LastInsertId()
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", inID, outID)
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", outID, inID)
			if _, err := tx.Exec(
				"UPDATE items SET location_id = ?, status = 'tersedia', belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?",
				boID, it.ID,
			); err != nil {
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "terima_bo", "items", 0, req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang diterima di back office", len(items))})
	}
}

// jumlah barang yang menunggu konfirmasi (untuk notifikasi)
func barangNotif(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		q := "SELECT COUNT(*), IFNULL(SUM(belum_sampai), 0) FROM items WHERE is_void = FALSE AND status = 'dikirim'"
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND location_id = ?"
			args = append(args, cl.LocationID)
		}
		var menunggu, belum int
		if err := db.QueryRow(q, args...).Scan(&menunggu, &belum); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"menunggu": menunggu, "belum_sampai": belum})
	}
}