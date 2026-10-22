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

func PenjualanBatal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if cl.Role != "pj" && cl.Role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
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

		var isVoid, recent bool
		var cashID sql.NullInt64
		err = tx.QueryRow(
			"SELECT is_void, cash_id, created_at >= NOW() - INTERVAL 1 DAY FROM penadah_sales WHERE id = ? FOR UPDATE", id,
		).Scan(&isVoid, &cashID, &recent)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "penjualan tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}
		if cl.Role == "pj" && !recent {
			c.JSON(http.StatusForbidden, gin.H{"error": "lebih dari 24 jam, minta Super Admin yang membatalkan"})
			return
		}

		ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE sale_id = ? FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		for _, itemID := range ids {
			var jenis string
			tx.QueryRow("SELECT jenis FROM items WHERE id = ?", itemID).Scan(&jenis)
			st := "tersedia"
			if jenis == "hasil_lebur" {
				st = "hasil"
			}
			tx.Exec("UPDATE items SET status = ?, sale_id = NULL WHERE id = ?", st, itemID)
			tx.Exec(`UPDATE stock_movements SET is_void = TRUE, void_reason = ?
			         WHERE item_id = ? AND category = 'jual_penadah' AND direction = 'out' AND is_void = FALSE
			         ORDER BY id DESC LIMIT 1`, reason, itemID)
			if jenis == "hasil_lebur" {
				tx.Exec("UPDATE melt_batches SET status = 'cukim' WHERE cukim_item_id = ? AND is_void = FALSE", itemID)
			}
		}
		if cashID.Valid {
			tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, cashID.Int64)
			services.CatatAudit(tx, cl.UserID, "void", "cash_transactions", cashID.Int64, gin.H{"reason": reason})
		}
		tx.Exec("UPDATE penadah_sales SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		services.CatatAudit(tx, cl.UserID, "penjualan_batal", "penadah_sales", int64(id), gin.H{"reason": reason, "oleh": cl.Role})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "penjualan dibatalkan, barang kembali siap dijual dan kas dikembalikan"})
	}
}

func PenjualanSiap(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.BolehLihatLebur(c); !ok {
			return
		}
		boID, err := services.IDBackOffice(db)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		rows, err := db.Query(
			`SELECT i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), i.weight_gram, i.jenis,
			        IFNULL(i.brand,''), i.harga_beli, i.melt_id
			 FROM items i
			 WHERE i.is_void = FALSE AND (
			   (i.jenis = 'logam_mulia' AND i.status = 'tersedia' AND i.location_id = ?)
			   OR (i.jenis = 'hasil_lebur' AND i.item_name = 'Cukim' AND i.status = 'hasil'))
			 ORDER BY i.id DESC`, boID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.PenjualanSiapRow{}
		for rows.Next() {
			var r models.PenjualanSiapRow
			var batch sql.NullInt64
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram, &r.Jenis, &r.Brand, &r.HargaBeli, &batch); err != nil {
				utils.Gagal(c, err)
				return
			}
			if batch.Valid {
				v := batch.Int64
				r.BatchID = &v
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- 4. PJ: JUAL KE PENADAH ----------

func PenjualanPenadah(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		var req models.PenjualanPenadahRequest
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		penadah := strings.TrimSpace(req.PenadahName)
		if penadah == "" || req.SaleTotal <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama penadah dan harga jual wajib diisi"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.SaleDate)
		if !ok {
			return
		}
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
			`SELECT id, item_name, jenis, status, location_id, harga_beli, weight_gram, IFNULL(karat,'')
			 FROM items WHERE is_void = FALSE AND id IN (`+utils.Placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		type brg struct {
			ID     int
			Name   string
			Jenis  string
			Status string
			Loc    int
			Harga  float64
			Berat  float64
			Karat  string
		}
		list := []brg{}
		for rows.Next() {
			var b brg
			if err := rows.Scan(&b.ID, &b.Name, &b.Jenis, &b.Status, &b.Loc, &b.Harga, &b.Berat, &b.Karat); err != nil {
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

		nCukim, nLM := 0, 0
		var modal, berat float64
		for _, b := range list {
			switch {
			case b.Jenis == "logam_mulia" && b.Status == "tersedia" && b.Loc == boID:
				nLM++
			case b.Jenis == "hasil_lebur" && b.Name == "Cukim" && b.Status == "hasil":
				nCukim++
			default:
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s tidak bisa dijual ke penadah", b.Name)})
				return
			}
			modal += b.Harga
			berat += b.Berat
		}
		if nCukim > 0 && nLM > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cukim dan logam mulia dijual terpisah"})
			return
		}
		if nCukim > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cukim dijual satu per satu"})
			return
		}

		note := strings.TrimSpace(req.Note)
		res, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			boID, "in", "jual_barang", req.SaleTotal,
			fmt.Sprintf("Jual ke penadah %s: %d barang", penadah, len(list)), tgl, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		cashID, _ := res.LastInsertId()

		res2, err := tx.Exec(
			`INSERT INTO penadah_sales (sale_date, penadah_name, sale_total, modal, weight_gram, item_count, note, cash_id, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			tgl, penadah, req.SaleTotal, modal, berat, len(list), sql.NullString{String: note, Valid: note != ""}, cashID, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		saleID, _ := res2.LastInsertId()

		for _, b := range list {
			if _, err := tx.Exec(services.SQLStokItem, boID, "out", "jual_penadah", b.Name, b.Berat, b.Karat,
				"Jual ke penadah "+penadah, tgl, cl.UserID, b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'terjual', sale_id = ? WHERE id = ?", saleID, b.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if b.Jenis == "hasil_lebur" {
				tx.Exec("UPDATE melt_batches SET status = 'terjual' WHERE cukim_item_id = ? AND is_void = FALSE", b.ID)
			}
		}
		services.CatatAudit(tx, cl.UserID, "jual_penadah", "penadah_sales", saleID, gin.H{"penadah": penadah, "harga_jual": req.SaleTotal, "modal": modal})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message": fmt.Sprintf("terjual ke %s, kas back office bertambah", penadah),
			"modal":   modal,
			"untung":  req.SaleTotal - modal,
		})
	}
}

// ---------- RIWAYAT PENJUALAN ----------

func PenjualanList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.BolehLihatLebur(c); !ok {
			return
		}
		q := `SELECT s.id, s.sale_date, s.penadah_name, s.item_count, s.weight_gram, s.modal, s.sale_total,
		             IFNULL(s.note,''), IFNULL(u.username,''),
		             IFNULL((SELECT GROUP_CONCAT(CONCAT(i.item_name,' ',IFNULL(i.karat,''),' ',i.weight_gram,'g') SEPARATOR '; ')
		                     FROM items i WHERE i.sale_id = s.id),'')
		      FROM penadah_sales s LEFT JOIN users u ON u.id = s.created_by
		      WHERE s.is_void = FALSE`
		args := []interface{}{}
		if v := c.Query("from"); v != "" {
			q += " AND s.sale_date >= ?"
			args = append(args, v)
		}
		if v := c.Query("to"); v != "" {
			q += " AND s.sale_date <= ?"
			args = append(args, v)
		}
		q += " ORDER BY s.sale_date DESC, s.id DESC LIMIT 200"

		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.PenjualanListRow{}
		var totalModal, totalJual float64
		for rows.Next() {
			var r models.PenjualanListRow
			if err := rows.Scan(&r.ID, &r.SaleDate, &r.PenadahName, &r.ItemCount, &r.WeightGram, &r.Modal, &r.SaleTotal,
				&r.Note, &r.PJ, &r.Barang); err != nil {
				utils.Gagal(c, err)
				return
			}
			r.Untung = r.SaleTotal - r.Modal
			totalModal += r.Modal
			totalJual += r.SaleTotal
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{
			"rows":         out,
			"total_modal":  totalModal,
			"total_jual":   totalJual,
			"total_untung": totalJual - totalModal,
		})
	}
}

// ---------- BATALKAN PENJUALAN (Super Admin) ----------

func PenjualanVoid(db *sql.DB) gin.HandlerFunc {
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
		err = tx.QueryRow("SELECT is_void, cash_id FROM penadah_sales WHERE id = ? FOR UPDATE", id).Scan(&isVoid, &cashID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "penjualan tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}

		ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE sale_id = ? FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		for _, itemID := range ids {
			var jenis string
			tx.QueryRow("SELECT jenis FROM items WHERE id = ?", itemID).Scan(&jenis)
			st := "tersedia"
			if jenis == "hasil_lebur" {
				st = "hasil"
			}
			tx.Exec("UPDATE items SET status = ?, sale_id = NULL WHERE id = ?", st, itemID)
			tx.Exec(`UPDATE stock_movements SET is_void = TRUE, void_reason = ?
			         WHERE item_id = ? AND category = 'jual_penadah' AND direction = 'out' AND is_void = FALSE
			         ORDER BY id DESC LIMIT 1`, reason, itemID)
			if jenis == "hasil_lebur" {
				tx.Exec("UPDATE melt_batches SET status = 'cukim' WHERE cukim_item_id = ? AND is_void = FALSE", itemID)
			}
		}
		if cashID.Valid {
			tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, cashID.Int64)
			services.CatatAudit(tx, cl.UserID, "void", "cash_transactions", cashID.Int64, gin.H{"reason": reason})
		}
		tx.Exec("UPDATE penadah_sales SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		services.CatatAudit(tx, cl.UserID, "void", "penadah_sales", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "penjualan dibatalkan, barang dan kas dikembalikan"})
	}
}
