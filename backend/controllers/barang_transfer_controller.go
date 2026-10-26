package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// toko mengantar barang ke back office (status dalam perjalanan)
func BarangKirimBO(db *sql.DB) gin.HandlerFunc {
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
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "kirim_bo", "items", 0, keadaanBarang(items),
			gin.H{"status": "dikirim", "request": req})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang diantar ke back office, menunggu konfirmasi", len(items))})
	}
}

// back office mengonfirmasi: sudah sampai atau belum sampai
func BarangTerimaBO(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req models.TerimaReq
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
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		boID, err := services.IDBackOffice(db)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		note := strings.TrimSpace(req.Note)

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
			if it.Status != "dikirim" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak sedang dalam perjalanan", it.Name)})
				return
			}
		}
		{
			idsArah := services.Kunci(items)
			argsArah := make([]interface{}, 0, len(idsArah))
			for _, id := range idsArah {
				argsArah = append(argsArah, id)
			}
			var arahLain int
			tx.QueryRow(
				"SELECT COUNT(*) FROM items WHERE id IN ("+utils.Placeholders(len(argsArah))+") AND kirim_ke_location_id IS NOT NULL",
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
					utils.Gagal(c, err)
					return
				}
			}
			services.CatatPerubahan(tx, middleware.Meta(c), "belum_sampai", "items", 0, keadaanBarang(items),
				gin.H{"status": "dikirim", "belum_sampai": true, "request": req})
			if err := tx.Commit(); err != nil {
				utils.Gagal(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang ditandai belum sampai", len(items))})
			return
		}

		for _, it := range items {
			r1, err := tx.Exec(services.SQLStokItem, it.Location, "out", "kembali_bo", it.Name, it.Weight, it.Karat,
				"Diantar ke Back Office", tgl, cl.UserID, it.ID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			r2, err := tx.Exec(services.SQLStokItem, boID, "in", "kembali_cabang", it.Name, it.Weight, it.Karat,
				"Diterima di Back Office", tgl, cl.UserID, it.ID)
			if err != nil {
				utils.Gagal(c, err)
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
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "terima_bo", "items", 0, keadaanBarang(items),
			gin.H{"status": "tersedia", "location_id": boID, "request": req})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang diterima di back office", len(items))})
	}
}

func BarangOper(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req models.BarangOperRequest
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
		namaToko, ok := services.CekCabang(db, req.TargetLocationID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih toko tujuan"})
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
			if it.Status != "tersedia" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak tersedia (%s)", it.Name, it.Status)})
				return
			}
			if it.Location != boID {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak berada di back office", it.Name)})
				return
			}
		}
		for _, it := range items {
			if _, err := tx.Exec(
				`UPDATE items SET status = 'dikirim', kirim_ke_location_id = ?, kirim_tgl = ?,
				        belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?`,
				req.TargetLocationID, tgl, it.ID,
			); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "oper_toko", "items", 0, keadaanBarang(items),
			gin.H{"status": "dikirim", "tujuan": req.TargetLocationID, "request": req})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("%d barang dioper ke %s, menunggu konfirmasi toko", len(items), namaToko),
		})
	}
}

// ---------- TOKO: KONFIRMASI BARANG DARI BACK OFFICE ----------

func BarangTerimaToko(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		// konfirmasi hanya oleh akun cabang tujuan; owner/admin hanya memantau progres
		if cl.Role != "cabang" {
			c.JSON(http.StatusForbidden, gin.H{"error": "konfirmasi barang kiriman hanya oleh akun toko tujuan"})
			return
		}
		var req models.BarangTerimaTokoRequest
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if req.Result != "sampai" && req.Result != "belum" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih sudah diterima atau belum diterima"})
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
		note := strings.TrimSpace(req.Note)

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
			`SELECT id, item_name, IFNULL(karat,''), weight_gram, location_id, status, IFNULL(kirim_ke_location_id,0)
			 FROM items WHERE is_void = FALSE AND id IN (`+utils.Placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		type brg struct {
			ID     int
			Name   string
			Karat  string
			Weight float64
			Loc    int
			Status string
			Tujuan int
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight, &b.Loc, &b.Status, &b.Tujuan); err != nil {
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
		for _, b := range list {
			if b.Status != "dikirim" || b.Tujuan == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak sedang dikirim ke toko", b.Name)})
				return
			}
			if cl.Role == "cabang" && b.Tujuan != cl.LocationID {
				c.JSON(http.StatusForbidden, gin.H{"error": "bukan barang kiriman untuk toko kamu"})
				return
			}
		}
		lama := gin.H{}
		for _, b := range list {
			lama[strconv.Itoa(b.ID)] = gin.H{"status": b.Status, "location_id": b.Loc, "tujuan": b.Tujuan}
		}

		if req.Result == "belum" {
			for _, b := range list {
				if _, err := tx.Exec("UPDATE items SET belum_sampai = TRUE, belum_sampai_note = ? WHERE id = ?",
					utils.StrNull(note), b.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			services.CatatPerubahan(tx, middleware.Meta(c), "belum_diterima_toko", "items", 0, lama,
				gin.H{"status": "dikirim", "belum_sampai": true, "request": req})
			if err := tx.Commit(); err != nil {
				utils.Gagal(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang ditandai belum diterima", len(list))})
			return
		}

		for _, b := range list {
			r1, err := tx.Exec(services.SQLStokItem, b.Loc, "out", "ke_toko", b.Name, b.Weight, b.Karat,
				"Oper ke toko", tgl, cl.UserID, b.ID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			r2, err := tx.Exec(services.SQLStokItem, b.Tujuan, "in", "barang_masuk", b.Name, b.Weight, b.Karat,
				"Kiriman dari Back Office", tgl, cl.UserID, b.ID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			outID, _ := r1.LastInsertId()
			inID, _ := r2.LastInsertId()
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", inID, outID)
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", outID, inID)
			if _, err := tx.Exec(
				`UPDATE items SET location_id = ?, status = 'tersedia', kirim_ke_location_id = NULL,
				        belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?`,
				b.Tujuan, b.ID,
			); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "terima_toko", "items", 0, lama,
			gin.H{"status": "tersedia", "location_id": cl.LocationID, "request": req})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang diterima di toko", len(list))})
	}
}

// ---------- ANGKA MERAH (dua arah) ----------

func BarangNotif(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var menunggu, belum, keluar int
		if cl.Role == "cabang" {
			// kiriman dari back office yang harus dikonfirmasi toko ini
			db.QueryRow(
				`SELECT COUNT(*), IFNULL(SUM(belum_sampai),0) FROM items
				 WHERE is_void = FALSE AND status = 'dikirim' AND kirim_ke_location_id = ?`, cl.LocationID,
			).Scan(&menunggu, &belum)
			// barang toko ini yang sedang diantar ke back office
			db.QueryRow(
				`SELECT COUNT(*) FROM items
				 WHERE is_void = FALSE AND status = 'dikirim' AND kirim_ke_location_id IS NULL AND location_id = ?`, cl.LocationID,
			).Scan(&keluar)
		} else {
			// kiriman dari toko yang harus dikonfirmasi back office
			db.QueryRow(
				`SELECT COUNT(*), IFNULL(SUM(belum_sampai),0) FROM items
				 WHERE is_void = FALSE AND status IN ('dikirim','kembali') AND kirim_ke_location_id IS NULL`,
			).Scan(&menunggu, &belum)
			// barang yang sedang dioper ke toko
			db.QueryRow(
				`SELECT COUNT(*) FROM items
				 WHERE is_void = FALSE AND status = 'dikirim' AND kirim_ke_location_id IS NOT NULL`,
			).Scan(&keluar)
		}
		c.JSON(http.StatusOK, gin.H{"menunggu": menunggu, "belum_sampai": belum, "keluar": keluar})
	}
}

func BarangTerimaKembali(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req models.BarangTerimaKembaliRequest
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if req.Result != "sampai" && req.Result != "belum" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih sudah diterima atau belum diterima"})
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
		note := strings.TrimSpace(req.Note)
		boID, err := services.IDBackOffice(db)
		if err != nil {
			utils.Gagal(c, err)
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
			`SELECT id, item_name, IFNULL(karat,''), weight_gram, status, IFNULL(kembali_jenis,'')
			 FROM items WHERE is_void = FALSE AND id IN (`+utils.Placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		type brg struct {
			ID     int
			Name   string
			Karat  string
			Weight float64
			Status string
			Jenis  string
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight, &b.Status, &b.Jenis); err != nil {
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
		for _, b := range list {
			if b.Status != "kembali" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak sedang dikembalikan ke back office", b.Name)})
				return
			}
		}
		lama := gin.H{}
		for _, b := range list {
			lama[strconv.Itoa(b.ID)] = gin.H{"status": b.Status, "kembali_jenis": b.Jenis}
		}

		if req.Result == "belum" {
			for _, b := range list {
				if _, err := tx.Exec("UPDATE items SET belum_sampai = TRUE, belum_sampai_note = ? WHERE id = ?", utils.StrNull(note), b.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			services.CatatPerubahan(tx, middleware.Meta(c), "belum_diterima_kembali", "items", 0, lama,
				gin.H{"status": "kembali", "belum_sampai": true, "request": req})
			if err := tx.Commit(); err != nil {
				utils.Gagal(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang ditandai belum diterima", len(list))})
			return
		}

		for _, b := range list {
			kategori, catatan := "batal_cuci", "Cuci dibatalkan, diterima back office"
			switch b.Jenis {
			case "cuci_selesai":
				kategori, catatan = "cucian", "Selesai cuci, diterima back office"
			case "lebur_batal":
				kategori, catatan = "batal_lebur", "Lebur dibatalkan, diterima back office"
			}
			if _, err := tx.Exec(services.SQLStokItem, boID, "in", kategori, b.Name, b.Weight, b.Karat, catatan, tgl, cl.UserID, b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec(
				`UPDATE items SET status = 'tersedia', kondisi = IF(? = 'cuci_selesai', 'bersih', kondisi),
				        kembali_jenis = NULL, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?`, b.Jenis, b.ID,
			); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "terima_kembali", "items", 0, lama,
			gin.H{"status": "tersedia", "location_id": boID, "request": req})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang diterima di back office", len(list))})
	}
}

// ---------- DAFTAR BARANG DI PENCUCI (dengan berat awal) ----------
