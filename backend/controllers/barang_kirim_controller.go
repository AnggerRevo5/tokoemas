package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// Kiriman baru berstatus "disiapkan" (diserahkan = FALSE): pengirim masih bisa membatalkan, penerima belum bisa konfirmasi.
// Setelah diserahkan ke kurir, barang dalam perjalanan dan hanya penerima yang mengonfirmasi.
func BarangSerahKurir(db *sql.DB) gin.HandlerFunc { return ubahTahapKirim(db, false) }

// Batal kirim: barang kembali tersedia di lokasi asal, tanpa menunggu penerima (stok belum pernah berpindah).
func BarangBatalKirim(db *sql.DB) gin.HandlerFunc { return ubahTahapKirim(db, true) }

func ubahTahapKirim(db *sql.DB, batal bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		var req models.KirimBOReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.ItemIDs) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
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
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		rows, err := tx.Query(
			`SELECT id, item_name, location_id, status, diserahkan, IFNULL(kirim_ke_location_id,0), IFNULL(bp_nama,''), weight_gram, IFNULL(karat,'')
			 FROM items WHERE is_void = FALSE AND id IN (`+utils.Placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		type brg struct {
			ID         int
			Name       string
			Loc        int
			Status     string
			Diserahkan bool
			Tujuan     int
		BP         string
		Berat      float64
		Karat      string
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Loc, &b.Status, &b.Diserahkan, &b.Tujuan, &b.BP, &b.Berat, &b.Karat); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			list = append(list, b)
		}
		rows.Close()
		if len(list) != len(args) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang yang tidak ditemukan"})
			return
		}

		lama := gin.H{}
		for _, b := range list {
			if b.Status != "dikirim" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak sedang dikirim", b.Name)})
				return
			}
			if b.Diserahkan {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sudah diserahkan ke kurir", b.Name)})
				return
			}
			// pengirim: toko untuk kiriman ke back office, back office (owner/admin) untuk oper ke toko
			if cl.Role == "cabang" && (b.Tujuan != 0 || b.BP != "" || b.Loc != cl.LocationID) {
				c.JSON(http.StatusForbidden, gin.H{"error": "bukan kiriman dari toko kamu"})
				return
			}
			lama[strconv.Itoa(b.ID)] = gin.H{"status": b.Status, "location_id": b.Loc, "tujuan": b.Tujuan, "diserahkan": b.Diserahkan}
		}

		sqlUbah, aksi, pesan := "UPDATE items SET diserahkan = TRUE WHERE id = ?", "serah_kurir", "%d barang diserahkan ke kurir, menunggu konfirmasi penerima"
		if batal {
			sqlUbah = `UPDATE items SET status = 'tersedia', kirim_ke_location_id = NULL, bp_nama = NULL, kirim_tgl = NULL, diserahkan = TRUE,
			           belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?`
			aksi, pesan = "batal_kirim", "%d barang batal dikirim, kembali ke stok"
		}
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		for _, b := range list {
			if !batal && b.BP != "" {
				// BP tanpa akun: begitu diserahkan ke kurir, stok back office berkurang dan barang tercatat di BP
				if _, err := tx.Exec(services.SQLStokItem, b.Loc, "out", "ke_bp", b.Name, b.Berat, b.Karat,
					"Dikirim ke "+b.BP, tgl, cl.UserID, b.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
				if _, err := tx.Exec("UPDATE items SET status = 'di_bp', diserahkan = TRUE, tray_id = NULL, kirim_tgl = ? WHERE id = ?", tgl, b.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
				continue
			}
			if _, err := tx.Exec(sqlUbah, b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatPerubahan(tx, middleware.Meta(c), aksi, "items", 0, lama, gin.H{"request": req})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf(pesan, len(list))})
	}
}

// BarangTarikBP: barang yang ada di BP ditarik kembali ke stok back office (stok bertambah lagi).
func BarangTarikBP(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req models.KirimBOReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if len(req.ItemIDs) > 300 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 300 barang sekali proses"})
			return
		}
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

		items, err := services.AmbilBarang(tx, req.ItemIDs)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		for _, it := range items {
			if it.Status != "di_bp" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak sedang berada di BP", it.Name)})
				return
			}
		}
		for _, it := range items {
			if _, err := tx.Exec(services.SQLStokItem, boID, "in", "kembali_bp", it.Name, it.Weight, it.Karat,
				"Ditarik kembali dari BP", tgl, cl.UserID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'tersedia', location_id = ?, bp_nama = NULL, kirim_tgl = NULL WHERE id = ?", boID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "tarik_bp", "items", 0, keadaanBarang(items), gin.H{"status": "tersedia", "request": req})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang ditarik kembali dari BP ke stok back office", len(items))})
	}
}
