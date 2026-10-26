package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// semua barang harus tersedia dan berada di back office
func barangDiBO(c *gin.Context, tx *sql.Tx, ids []int, boID int) (map[int]models.ItemRow, bool) {
	if len(ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
		return nil, false
	}
	if len(ids) > 300 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
		return nil, false
	}
	items, err := services.AmbilBarang(tx, ids)
	if err != nil {
		balasKesalahan(c, err)
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
	tx.QueryRow("SELECT COUNT(*) FROM items WHERE jenis = 'logam_mulia' AND id IN ("+utils.Placeholders(len(cekArgs))+")", cekArgs...).Scan(&lm)
	if lm > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "logam mulia tidak bisa dicuci atau dilebur, langsung dijual saja"})
		return nil, false
	}
	return items, true
}

// keadaanBarang meringkas status dan lokasi barang sebelum diubah (nilai "sebelum" di audit).
func keadaanBarang(items map[int]models.ItemRow) gin.H {
	out := gin.H{}
	for id, it := range items {
		out[strconv.Itoa(id)] = gin.H{"status": it.Status, "location_id": it.Location}
	}
	return out
}

// ---------- CUCI ----------

func resolveLocation(c *gin.Context, db *sql.DB, requested int) (int, string, bool) {
	cl := c.MustGet("claims").(*models.Claims)
	id := requested
	if cl.Role == "cabang" {
		id = cl.LocationID
	} else if cl.Role != "super_admin" && cl.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
		return 0, "", false
	}
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "location_id wajib diisi"})
		return 0, "", false
	}
	var tipe string
	if err := db.QueryRow("SELECT type FROM locations WHERE id = ?", id).Scan(&tipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lokasi tidak ditemukan"})
		return 0, "", false
	}
	return id, tipe, true
}
