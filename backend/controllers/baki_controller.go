package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// ---------- DAFTAR BAKI ----------

type bakiRow struct {
	ID         int    `json:"id"`
	No         int    `json:"no"`
	LocationID int    `json:"location_id"`
	Lokasi     string `json:"lokasi"`
	Isi        int    `json:"isi"`
	Kapasitas  int    `json:"kapasitas"`
}

func BakiList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		q := `SELECT t.id, t.no, t.location_id, l.name,
		             (SELECT COUNT(*) FROM items i WHERE i.tray_id = t.id AND i.location_id = t.location_id
		              AND i.is_void = FALSE AND i.status NOT IN ('terjual','lebur','hasil'))
		      FROM trays t JOIN locations l ON l.id = t.location_id`
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " WHERE t.location_id = ?"
			args = append(args, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" && loc != "0" {
			q += " WHERE t.location_id = ?"
			args = append(args, loc)
		}
		q += " ORDER BY t.location_id, t.no"

		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		out := []bakiRow{}
		for rows.Next() {
			r := bakiRow{Kapasitas: services.KapasitasBaki}
			if err := rows.Scan(&r.ID, &r.No, &r.LocationID, &r.Lokasi, &r.Isi); err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "kapasitas": services.KapasitasBaki})
	}
}

// ---------- TAMBAH BAKI ----------

func BakiCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req models.BakiBaruReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, _, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		// kunci baris lokasi supaya dua permintaan bersamaan tidak membuat nomor yang sama
		if _, err := tx.Exec("SELECT id FROM locations WHERE id = ? FOR UPDATE", locID); err != nil {
			utils.Gagal(c, err)
			return
		}
		// nomor baki = nomor terkecil yang belum dipakai, jadi nomor baki yang pernah dihapus muncul lagi
		dipakai, err := services.AmbilIDs(tx, "SELECT no FROM trays WHERE location_id = ? ORDER BY no", locID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		no := 1
		for _, n := range dipakai {
			if n != no {
				break
			}
			no++
		}
		res, err := tx.Exec("INSERT INTO trays (location_id, no, created_by) VALUES (?,?,?)", locID, no, cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "create", "trays", id, gin.H{"location_id": locID, "no": no})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("Baki %d ditambahkan", no), "id": id, "no": no})
	}
}

// ---------- HAPUS BAKI (hanya yang kosong) ----------

func BakiHapus(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var loc, no int
		if err := tx.QueryRow("SELECT location_id, no FROM trays WHERE id = ? FOR UPDATE", id).Scan(&loc, &no); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "baki tidak ditemukan"})
			return
		}
		if cl.Role == "cabang" && loc != cl.LocationID {
			c.JSON(http.StatusForbidden, gin.H{"error": "bukan baki di lokasi kamu"})
			return
		}
		sisa, err := services.SisaBaki(tx, id, loc)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		if sisa < services.KapasitasBaki {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Baki %d masih berisi %d barang; pindahkan dulu isinya", no, services.KapasitasBaki-sisa)})
			return
		}
		// barang lama (terjual/dilebur/pindah lokasi) yang masih menunjuk baki ini dilepas
		if _, err := tx.Exec("UPDATE items SET tray_id = NULL WHERE tray_id = ?", id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec("DELETE FROM trays WHERE id = ?", id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "delete", "trays", int64(id), gin.H{"location_id": loc, "no": no}, nil)
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Baki %d dihapus", no)})
	}
}

// ---------- PINDAH BAKI ----------

func BakiPindah(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		var req models.PindahBakiReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 || req.TrayID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih barang dan baki tujuan"})
			return
		}
		if len(req.ItemIDs) > services.KapasitasBaki {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("maksimal %d barang sekali pindah", services.KapasitasBaki)})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var loc, no int
		if err := tx.QueryRow("SELECT location_id, no FROM trays WHERE id = ?", req.TrayID).Scan(&loc, &no); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "baki tujuan tidak ditemukan"})
			return
		}
		if cl.Role == "cabang" && loc != cl.LocationID {
			c.JSON(http.StatusForbidden, gin.H{"error": "bukan baki di lokasi kamu"})
			return
		}
		items, err := services.AmbilBarang(tx, req.ItemIDs)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		args := make([]interface{}, 0, len(items))
		for _, it := range items {
			if it.Status != "tersedia" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sedang tidak tersedia (%s)", it.Name, it.Status)})
				return
			}
			if it.Location != loc {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak berada di lokasi baki tujuan", it.Name)})
				return
			}
			args = append(args, it.ID)
		}
		var bukanPerhiasan, sudahDiSana int
		tx.QueryRow("SELECT COUNT(*) FROM items WHERE jenis <> 'perhiasan' AND id IN ("+utils.Placeholders(len(args))+")", args...).Scan(&bukanPerhiasan)
		if bukanPerhiasan > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hanya perhiasan yang bisa dimasukkan ke baki"})
			return
		}
		tx.QueryRow("SELECT COUNT(*) FROM items WHERE tray_id = ? AND id IN ("+utils.Placeholders(len(args))+")",
			append([]interface{}{req.TrayID}, args...)...).Scan(&sudahDiSana)

		sisa, err := services.SisaBaki(tx, req.TrayID, loc)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		masuk := len(items) - sudahDiSana
		if masuk > sisa {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Baki %d hanya punya sisa tempat %d, barang yang dipindah %d", no, sisa, masuk)})
			return
		}

		lama := gin.H{}
		rows, err := tx.Query("SELECT id, IFNULL(tray_id,0) FROM items WHERE id IN ("+utils.Placeholders(len(args))+")", args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		for rows.Next() {
			var id, tray int
			rows.Scan(&id, &tray)
			lama[fmt.Sprint(id)] = tray
		}
		rows.Close()

		if _, err := tx.Exec("UPDATE items SET tray_id = ? WHERE id IN ("+utils.Placeholders(len(args))+")",
			append([]interface{}{req.TrayID}, args...)...); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "pindah_baki", "items", 0, lama, gin.H{"tray_id": req.TrayID, "item_ids": req.ItemIDs})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang dipindah ke Baki %d", len(items), no)})
	}
}

// ---------- EDIT BARANG ----------

func BarangEdit(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.BarangEditReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		req.ItemName = strings.TrimSpace(req.ItemName)
		req.Brand = strings.TrimSpace(req.Brand)
		req.Note = strings.TrimSpace(req.Note)
		if req.WeightGram <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "berat wajib diisi"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var lama struct {
			Name, Karat, Brand, Note, Jenis, Status string
			Weight, Harga                           float64
			Loc, Tray                               int
			Cash                                    sql.NullInt64
		}
		err = tx.QueryRow(
			`SELECT item_name, IFNULL(karat,''), IFNULL(brand,''), IFNULL(note,''), jenis, status,
			        weight_gram, harga_beli, location_id, IFNULL(tray_id,0), purchase_cash_id
			 FROM items WHERE id = ? AND is_void = FALSE FOR UPDATE`, id).
			Scan(&lama.Name, &lama.Karat, &lama.Brand, &lama.Note, &lama.Jenis, &lama.Status,
				&lama.Weight, &lama.Harga, &lama.Loc, &lama.Tray, &lama.Cash)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "barang tidak ditemukan"})
			return
		}
		if cl.Role == "cabang" && lama.Loc != cl.LocationID {
			c.JSON(http.StatusForbidden, gin.H{"error": "bukan barang di lokasi kamu"})
			return
		}
		if lama.Status != "tersedia" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hanya barang yang tersedia di stok yang bisa diedit"})
			return
		}

		nama, karat, brand := req.ItemName, "", ""
		switch lama.Jenis {
		case "logam_mulia":
			if req.Brand == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "merek logam mulia wajib diisi"})
				return
			}
			brand, karat, nama = req.Brand, "24K", "Logam mulia "+req.Brand
			req.TrayID = 0
		case "perhiasan":
			if nama == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "nama perhiasan wajib diisi"})
				return
			}
			k, ok := utils.NormalKarat(req.Karat)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "kadar harus 1K sampai 24K"})
				return
			}
			karat = k
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang hasil lebur dikelola di menu Lebur"})
			return
		}

		// harga beli dari customer sudah tercatat di kas; ubah lewat batalkan lalu input ulang
		if lama.Cash.Valid && req.HargaBeli != lama.Harga {
			c.JSON(http.StatusBadRequest, gin.H{"error": "harga beli barang dari customer sudah masuk kas dan tidak bisa diubah di sini"})
			return
		}

		if req.TrayID > 0 && req.TrayID != lama.Tray {
			sisa, err := services.SisaBaki(tx, req.TrayID, lama.Loc)
			if err != nil {
				balasKesalahan(c, err)
				return
			}
			if sisa < 1 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("baki sudah penuh (maksimal %d barang)", services.KapasitasBaki)})
				return
			}
		}

		if _, err := tx.Exec(
			`UPDATE items SET item_name = ?, karat = ?, brand = ?, weight_gram = ?, harga_beli = ?, note = ?, tray_id = ? WHERE id = ?`,
			nama, karat, utils.StrNull(brand), req.WeightGram, req.HargaBeli, utils.StrNull(req.Note),
			sql.NullInt64{Int64: int64(req.TrayID), Valid: req.TrayID > 0}, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		// catatan stok ikut diperbarui agar nama, kadar, dan berat tetap sama dengan barangnya
		if _, err := tx.Exec(
			"UPDATE stock_movements SET item_name = ?, karat = ?, weight_gram = ? WHERE item_id = ? AND is_void = FALSE",
			nama, karat, req.WeightGram, id); err != nil {
			utils.Gagal(c, err)
			return
		}

		services.CatatPerubahan(tx, middleware.Meta(c), "edit", "items", int64(id),
			gin.H{"item_name": lama.Name, "karat": lama.Karat, "brand": lama.Brand, "weight_gram": lama.Weight,
				"harga_beli": lama.Harga, "note": lama.Note, "tray_id": lama.Tray},
			gin.H{"item_name": nama, "karat": karat, "brand": brand, "weight_gram": req.WeightGram,
				"harga_beli": req.HargaBeli, "note": req.Note, "tray_id": req.TrayID})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang diperbarui"})
	}
}
