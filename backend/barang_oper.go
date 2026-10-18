package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ---------- OWNER: OPER KE TOKO (menunggu konfirmasi toko) ----------

func barangOper(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req struct {
			ItemIDs          []int  `json:"item_ids"`
			TargetLocationID int    `json:"target_location_id"`
			Date             string `json:"date"`
		}
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
		namaToko, ok := cekCabang(db, req.TargetLocationID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih toko tujuan (Kepanjen atau Mall Dinoyo)"})
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
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "oper_toko", "items", 0, req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("%d barang dioper ke %s, menunggu konfirmasi toko", len(items), namaToko),
		})
	}
}

// ---------- TOKO: KONFIRMASI BARANG DARI BACK OFFICE ----------

func barangTerimaToko(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		if cl.Role == "pj" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
			return
		}
		var req struct {
			ItemIDs []int  `json:"item_ids"`
			Result  string `json:"result"`
			Date    string `json:"date"`
			Note    string `json:"note"`
		}
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
		tgl, ok := parseTanggal(c, req.Date)
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
			gagal(c, err)
			return
		}
		defer tx.Rollback()

		rows, err := tx.Query(
			`SELECT id, item_name, IFNULL(karat,''), weight_gram, location_id, status, IFNULL(kirim_ke_location_id,0)
			 FROM items WHERE is_void = FALSE AND id IN (`+placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			gagal(c, err)
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
				gagal(c, err)
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

		if req.Result == "belum" {
			for _, b := range list {
				if _, err := tx.Exec("UPDATE items SET belum_sampai = TRUE, belum_sampai_note = ? WHERE id = ?",
					strNull(note), b.ID); err != nil {
					gagal(c, err)
					return
				}
			}
			catatAudit(tx, cl.UserID, "belum_diterima_toko", "items", 0, req)
			if err := tx.Commit(); err != nil {
				gagal(c, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang ditandai belum diterima", len(list))})
			return
		}

		for _, b := range list {
			r1, err := tx.Exec(sqlStokItem, b.Loc, "out", "ke_toko", b.Name, b.Weight, b.Karat,
				"Oper ke toko", tgl, cl.UserID, b.ID)
			if err != nil {
				gagal(c, err)
				return
			}
			r2, err := tx.Exec(sqlStokItem, b.Tujuan, "in", "barang_masuk", b.Name, b.Weight, b.Karat,
				"Kiriman dari Back Office", tgl, cl.UserID, b.ID)
			if err != nil {
				gagal(c, err)
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
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "terima_toko", "items", 0, req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang diterima di toko", len(list))})
	}
}

// ---------- ANGKA MERAH (dua arah) ----------

func barangNotifV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
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