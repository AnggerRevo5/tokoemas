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

func LeburKembali(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.LeburKembaliRequest
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan wajib diisi"})
			return
		}
		alasan := strings.TrimSpace(req.Reason)
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var status string
		var isVoid bool
		err = tx.QueryRow("SELECT status, is_void FROM melt_batches WHERE id = ? FOR UPDATE", id).Scan(&status, &isVoid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "lebur tidak ditemukan"})
			return
		}
		if isVoid || status == "batal" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lebur sudah dibatalkan"})
			return
		}
		if status != "proses" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang sudah dilebur, tidak bisa dikembalikan. Kalau salah input, minta Super Admin membatalkan langkahnya"})
			return
		}

		ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE melt_id = ? AND jenis <> 'hasil_lebur' AND status = 'lebur' AND is_void = FALSE FOR UPDATE", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if len(ids) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak ada barang dalam proses lebur"})
			return
		}
		for _, itemID := range ids {
			if _, err := tx.Exec(
				`UPDATE items SET status = 'kembali', kembali_jenis = 'lebur_batal', belum_sampai = FALSE, belum_sampai_note = NULL
				 WHERE id = ?`, itemID,
			); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		if _, err := tx.Exec(
			"UPDATE melt_batches SET status = 'batal', cancel_reason = ?, cancelled_by = ?, cancelled_date = ? WHERE id = ?",
			utils.BatasTeks(alasan, 250), cl.UserID, tgl, id,
		); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "lebur_batal_kembali", "melt_batches", int64(id), gin.H{"alasan": alasan, "dikembalikan": len(ids)})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("lebur #%d dibatalkan, %d barang dikembalikan ke back office (menunggu konfirmasi owner)", id, len(ids)),
		})
	}
}

// ---------- OWNER: KONFIRMASI BARANG KEMBALI DARI PJ ----------

func LeburMulai(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req models.LeburMulaiRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.StartDate)
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

		items, ok := barangDiBO(c, tx, req.ItemIDs, boID)
		if !ok {
			return
		}
		ids := services.Kunci(items)
		args := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		var modal, berat float64
		if err := tx.QueryRow(
			"SELECT IFNULL(SUM(harga_beli),0), IFNULL(SUM(weight_gram),0) FROM items WHERE id IN ("+utils.Placeholders(len(args))+")",
			args...,
		).Scan(&modal, &berat); err != nil {
			utils.Gagal(c, err)
			return
		}

		note := strings.TrimSpace(req.Note)
		res, err := tx.Exec(
			`INSERT INTO melt_batches (start_date, status, item_count, source_weight, modal, note, created_by)
			 VALUES (?, 'proses', ?, ?, ?, ?, ?)`,
			tgl, len(ids), berat, modal, sql.NullString{String: note, Valid: note != ""}, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		batchID, _ := res.LastInsertId()

		for _, it := range items {
			if _, err := tx.Exec(services.SQLStokItem, boID, "out", "lebur", it.Name, it.Weight, it.Karat,
				fmt.Sprintf("Masuk proses lebur #%d", batchID), tgl, cl.UserID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'lebur', melt_id = ? WHERE id = ?", batchID, it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatAudit(tx, cl.UserID, "lebur_mulai", "melt_batches", batchID, gin.H{"jumlah": len(ids), "berat": berat, "modal": modal})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":      batchID,
			"message": fmt.Sprintf("%d barang masuk proses lebur (lebur #%d). PJ mencatat hasilnya di menu Lebur", len(items), batchID),
		})
	}
}

// ---------- 2. PJ: BERAT SETELAH POTONG DAN LEBURAN KOTOR ----------

func LeburCatatLeburan(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.LeburCatatLeburanRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.CutWeight <= 0 || req.LeburanWeight <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi berat setelah dipotong dan berat leburan kotor"})
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

		var status string
		var isVoid bool
		var source, modal float64
		err = tx.QueryRow("SELECT status, is_void, source_weight, modal FROM melt_batches WHERE id = ? FOR UPDATE", id).
			Scan(&status, &isVoid, &source, &modal)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "lebur tidak ditemukan"})
			return
		}
		if isVoid || status != "proses" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "langkah ini sudah dicatat atau lebur dibatalkan"})
			return
		}
		if req.CutWeight > source+0.0005 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("berat setelah dipotong tidak boleh lebih dari berat awal (%.3f g)", source)})
			return
		}
		if req.LeburanWeight > req.CutWeight+0.0005 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "berat leburan kotor tidak boleh lebih dari berat setelah dipotong"})
			return
		}

		note := strings.TrimSpace(req.Note)
		itemID, err := services.BuatItemHasil(tx, "Leburan kotor", sql.NullString{}, req.LeburanWeight, boID, tgl, note,
			cl.UserID, "kotor", modal, int64(id), "leburan_kotor")
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec(
			`UPDATE melt_batches SET status = 'leburan_kotor', cut_weight = ?, cut_note = ?, leburan_weight = ?,
			        leburan_date = ?, leburan_by = ?, leburan_item_id = ? WHERE id = ?`,
			req.CutWeight, sql.NullString{String: note, Valid: note != ""}, req.LeburanWeight, tgl, cl.UserID, itemID, id,
		); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "leburan_kotor", "melt_batches", int64(id), req)

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "leburan kotor dicatat, masuk stok back office"})
	}
}

// ---------- 3. PJ: CUKIM (HASIL PEMURNIAN) ----------

func LeburCatatCukim(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPJ(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.LeburCatatCukimRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.CukimWeight <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi berat cukim"})
			return
		}
		karat := sql.NullString{}
		if strings.TrimSpace(req.Karat) != "" {
			k, ok := utils.NormalKarat(req.Karat)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "kadar harus 1K sampai 24K"})
				return
			}
			karat = sql.NullString{String: k, Valid: true}
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

		var status string
		var isVoid bool
		var leburan, modal float64
		var leburanItem sql.NullInt64
		err = tx.QueryRow("SELECT status, is_void, leburan_weight, modal, leburan_item_id FROM melt_batches WHERE id = ? FOR UPDATE", id).
			Scan(&status, &isVoid, &leburan, &modal, &leburanItem)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "lebur tidak ditemukan"})
			return
		}
		if isVoid || status != "leburan_kotor" || !leburanItem.Valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "catat leburan kotor dulu, atau langkah ini sudah dicatat"})
			return
		}
		if req.CukimWeight > leburan+0.0005 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("berat cukim tidak boleh lebih dari leburan kotor (%.3f g)", leburan)})
			return
		}

		note := strings.TrimSpace(req.Note)
		if _, err := tx.Exec(services.SQLStokItem, boID, "out", "murnikan", "Leburan kotor", leburan, "",
			"Dimurnikan jadi cukim", tgl, cl.UserID, leburanItem.Int64); err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec("UPDATE items SET status = 'lebur' WHERE id = ?", leburanItem.Int64); err != nil {
			utils.Gagal(c, err)
			return
		}
		cukimID, err := services.BuatItemHasil(tx, "Cukim", karat, req.CukimWeight, boID, tgl, note,
			cl.UserID, "bersih", modal, int64(id), "cukim")
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec(
			`UPDATE melt_batches SET status = 'cukim', cukim_weight = ?, cukim_karat = ?, cukim_date = ?,
			        cukim_by = ?, cukim_item_id = ? WHERE id = ?`,
			req.CukimWeight, karat, tgl, cl.UserID, cukimID, id,
		); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, cl.UserID, "cukim", "melt_batches", int64(id), req)

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cukim dicatat, siap dijual ke penadah"})
	}
}

// ---------- BATALKAN LANGKAH TERAKHIR (Super Admin) ----------

func LeburUndo(db *sql.DB) gin.HandlerFunc {
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

		var status string
		var isVoid bool
		var leburanItem, cukimItem sql.NullInt64
		err = tx.QueryRow("SELECT status, is_void, leburan_item_id, cukim_item_id FROM melt_batches WHERE id = ? FOR UPDATE", id).
			Scan(&status, &isVoid, &leburanItem, &cukimItem)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "lebur tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "lebur sudah dibatalkan"})
			return
		}

		msg := ""
		switch status {
		case "terjual":
			c.JSON(http.StatusBadRequest, gin.H{"error": "batalkan penjualan ke penadah dulu"})
			return

		case "cukim":
			if !cukimItem.Valid || !leburanItem.Valid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "data cukim tidak lengkap"})
				return
			}
			tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE item_id = ? AND is_void = FALSE", reason, cukimItem.Int64)
			tx.Exec("UPDATE items SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, cukimItem.Int64)
			tx.Exec(`UPDATE stock_movements SET is_void = TRUE, void_reason = ?
			         WHERE item_id = ? AND category = 'murnikan' AND direction = 'out' AND is_void = FALSE`, reason, leburanItem.Int64)
			tx.Exec("UPDATE items SET status = 'hasil' WHERE id = ?", leburanItem.Int64)
			tx.Exec(`UPDATE melt_batches SET status = 'leburan_kotor', cukim_weight = NULL, cukim_karat = NULL,
			         cukim_date = NULL, cukim_by = NULL, cukim_item_id = NULL WHERE id = ?`, id)
			msg = "langkah cukim dibatalkan, kembali ke tahap leburan kotor"

		case "leburan_kotor":
			if !leburanItem.Valid {
				c.JSON(http.StatusBadRequest, gin.H{"error": "data leburan tidak lengkap"})
				return
			}
			tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE item_id = ? AND is_void = FALSE", reason, leburanItem.Int64)
			tx.Exec("UPDATE items SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, leburanItem.Int64)
			tx.Exec(`UPDATE melt_batches SET status = 'proses', cut_weight = NULL, cut_note = NULL, leburan_weight = NULL,
			         leburan_date = NULL, leburan_by = NULL, leburan_item_id = NULL WHERE id = ?`, id)
			msg = "langkah leburan kotor dibatalkan, kembali ke tahap proses lebur"

		case "proses":
			ids, err := services.AmbilIDs(tx, "SELECT id FROM items WHERE melt_id = ? AND jenis <> 'hasil_lebur' AND status = 'lebur' FOR UPDATE", id)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			for _, itemID := range ids {
				tx.Exec(`UPDATE stock_movements SET is_void = TRUE, void_reason = ?
				         WHERE item_id = ? AND category = 'lebur' AND direction = 'out' AND is_void = FALSE
				         ORDER BY id DESC LIMIT 1`, reason, itemID)
				tx.Exec("UPDATE items SET status = 'tersedia', melt_id = NULL WHERE id = ?", itemID)
			}
			tx.Exec("UPDATE melt_batches SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
			msg = fmt.Sprintf("lebur dibatalkan, %d barang kembali tersedia di back office", len(ids))
		}

		services.CatatAudit(tx, cl.UserID, "lebur_undo", "melt_batches", int64(id), gin.H{"dari_tahap": status, "alasan": reason})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": msg})
	}
}

// ---------- DAFTAR LEBUR ----------

func LeburList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.BolehLihatLebur(c); !ok {
			return
		}
		q := `SELECT b.id, b.start_date, b.status, b.item_count, b.source_weight, b.modal,
		             b.cut_weight, b.leburan_weight, b.cukim_weight, IFNULL(b.cukim_karat,''),
		             IFNULL(b.note,''), IFNULL(b.cut_note,''), b.leburan_date, b.cukim_date,
		             IFNULL(u1.username,''), IFNULL(u2.username,''),
		             IFNULL((SELECT GROUP_CONCAT(CONCAT(i.item_name,' ',IFNULL(i.karat,''),' ',i.weight_gram,'g') SEPARATOR '; ')
		                     FROM items i WHERE i.melt_id = b.id AND i.jenis <> 'hasil_lebur'),''),
		             ps.sale_total, IFNULL(ps.penadah_name,''), ps.sale_date
		      FROM melt_batches b
		      LEFT JOIN users u1 ON u1.id = b.leburan_by
		      LEFT JOIN users u2 ON u2.id = b.cukim_by
		      LEFT JOIN items ci ON ci.id = b.cukim_item_id
		      LEFT JOIN penadah_sales ps ON ps.id = ci.sale_id AND ps.is_void = FALSE
		      WHERE b.is_void = FALSE`
		switch c.Query("status") {
		case "terjual":
			q += " AND b.status = 'terjual'"
		case "batal":
			q += " AND b.status = 'batal'"
		case "semua":
		default:
			q += " AND b.status NOT IN ('terjual','batal')"
		}
		q += " ORDER BY b.id DESC LIMIT 200"

		rows, err := db.Query(q)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.LeburListRow{}
		for rows.Next() {
			var r models.LeburListRow
			var cut, leb, cuk, sale sql.NullFloat64
			var lebDate, cukDate, saleDate sql.NullTime
			if err := rows.Scan(&r.ID, &r.StartDate, &r.Status, &r.ItemCount, &r.SourceWeight, &r.Modal,
				&cut, &leb, &cuk, &r.CukimKarat, &r.Note, &r.CutNote, &lebDate, &cukDate,
				&r.LeburanBy, &r.CukimBy, &r.Barang, &sale, &r.Penadah, &saleDate); err != nil {
				utils.Gagal(c, err)
				return
			}
			r.CutWeight, r.LeburanWeight, r.CukimWeight = utils.PtrFloat(cut), utils.PtrFloat(leb), utils.PtrFloat(cuk)
			r.LeburanDate, r.CukimDate, r.SaleDate = utils.PtrTime(lebDate), utils.PtrTime(cukDate), utils.PtrTime(saleDate)
			r.SaleTotal = utils.PtrFloat(sale)
			if sale.Valid {
				u := sale.Float64 - r.Modal
				r.Untung = &u
			}
			if cuk.Valid {
				s := r.SourceWeight - cuk.Float64
				r.Susut = &s
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- BARANG SIAP DIJUAL KE PENADAH ----------
