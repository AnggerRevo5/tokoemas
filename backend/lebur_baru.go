package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ---------- AKSES ----------

func hanyaPJ(c *gin.Context) (*Claims, bool) {
	cl := c.MustGet("claims").(*Claims)
	if cl.Role != "pj" && cl.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "hanya PJ yang bisa mencatat langkah ini"})
		return nil, false
	}
	return cl, true
}

func bolehLihatLebur(c *gin.Context) (*Claims, bool) {
	cl := c.MustGet("claims").(*Claims)
	if cl.Role != "pj" && cl.Role != "owner" && cl.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
		return nil, false
	}
	return cl, true
}

func batasiPJ() gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		if cl.Role == "pj" {
			p := c.FullPath()
			boleh := p == "/me" || p == "/barang/notif" ||
				strings.HasPrefix(p, "/lebur") ||
				strings.HasPrefix(p, "/penjualan") ||
				strings.HasPrefix(p, "/service")
			if !boleh {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
				return
			}
		}
		c.Next()
	}
}

// ---------- BANTUAN ----------

const sqlItemHasil = `INSERT INTO items
	(item_name, karat, weight_gram, origin_location_id, location_id, source_category, entry_date, note, created_by,
	 jenis, kondisi, harga_beli, status, melt_id)
	VALUES (?,?,?,?,?,'lebur',?,?,?,'hasil_lebur',?,?,'hasil',?)`

func buatItemHasil(tx *sql.Tx, nama string, karat sql.NullString, berat float64, boID int, tgl, note string,
	userID int, kondisi string, modal float64, batchID int64, kategori string) (int64, error) {
	res, err := tx.Exec(sqlItemHasil, nama, karat, berat, boID, boID, tgl, note, userID, kondisi, modal, batchID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	tx.Exec("UPDATE items SET item_no = ? WHERE id = ?", fmt.Sprintf("BRG-%05d", id), id)
	karatStr := ""
	if karat.Valid {
		karatStr = karat.String
	}
	if _, err := tx.Exec(sqlStokItem, boID, "in", kategori, nama, berat, karatStr, note, tgl, userID, id); err != nil {
		return 0, err
	}
	return id, nil
}

func ptrFloat(n sql.NullFloat64) *float64 {
	if n.Valid {
		v := n.Float64
		return &v
	}
	return nil
}

func ptrTime(n sql.NullTime) *time.Time {
	if n.Valid {
		v := n.Time
		return &v
	}
	return nil
}

// ---------- 1. OWNER MEMULAI LEBUR ----------

func leburMulai(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaAdminOwner(c)
		if !ok {
			return
		}
		var req struct {
			ItemIDs   []int  `json:"item_ids"`
			StartDate string `json:"start_date"`
			Note      string `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		tgl, ok := parseTanggal(c, req.StartDate)
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

		items, ok := barangDiBO(c, tx, req.ItemIDs, boID)
		if !ok {
			return
		}
		ids := kunci(items)
		args := make([]interface{}, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		var modal, berat float64
		if err := tx.QueryRow(
			"SELECT IFNULL(SUM(harga_beli),0), IFNULL(SUM(weight_gram),0) FROM items WHERE id IN ("+placeholders(len(args))+")",
			args...,
		).Scan(&modal, &berat); err != nil {
			gagal(c, err)
			return
		}

		note := strings.TrimSpace(req.Note)
		res, err := tx.Exec(
			`INSERT INTO melt_batches (start_date, status, item_count, source_weight, modal, note, created_by)
			 VALUES (?, 'proses', ?, ?, ?, ?, ?)`,
			tgl, len(ids), berat, modal, sql.NullString{String: note, Valid: note != ""}, cl.UserID,
		)
		if err != nil {
			gagal(c, err)
			return
		}
		batchID, _ := res.LastInsertId()

		for _, it := range items {
			if _, err := tx.Exec(sqlStokItem, boID, "out", "lebur", it.Name, it.Weight, it.Karat,
				fmt.Sprintf("Masuk proses lebur #%d", batchID), tgl, cl.UserID, it.ID); err != nil {
				gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'lebur', melt_id = ? WHERE id = ?", batchID, it.ID); err != nil {
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "lebur_mulai", "melt_batches", batchID, gin.H{"jumlah": len(ids), "berat": berat, "modal": modal})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":      batchID,
			"message": fmt.Sprintf("%d barang masuk proses lebur (lebur #%d). PJ mencatat hasilnya di menu Lebur", len(items), batchID),
		})
	}
}

// ---------- 2. PJ: BERAT SETELAH POTONG DAN LEBURAN KOTOR ----------

func leburCatatLeburan(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaPJ(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		var req struct {
			CutWeight     float64 `json:"cut_weight"`
			LeburanWeight float64 `json:"leburan_weight"`
			Date          string  `json:"date"`
			Note          string  `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.CutWeight <= 0 || req.LeburanWeight <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi berat setelah dipotong dan berat leburan kotor"})
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
		itemID, err := buatItemHasil(tx, "Leburan kotor", sql.NullString{}, req.LeburanWeight, boID, tgl, note,
			cl.UserID, "kotor", modal, int64(id), "leburan_kotor")
		if err != nil {
			gagal(c, err)
			return
		}
		if _, err := tx.Exec(
			`UPDATE melt_batches SET status = 'leburan_kotor', cut_weight = ?, cut_note = ?, leburan_weight = ?,
			        leburan_date = ?, leburan_by = ?, leburan_item_id = ? WHERE id = ?`,
			req.CutWeight, sql.NullString{String: note, Valid: note != ""}, req.LeburanWeight, tgl, cl.UserID, itemID, id,
		); err != nil {
			gagal(c, err)
			return
		}
		catatAudit(tx, cl.UserID, "leburan_kotor", "melt_batches", int64(id), req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "leburan kotor dicatat, masuk stok back office"})
	}
}

// ---------- 3. PJ: CUKIM (HASIL PEMURNIAN) ----------

func leburCatatCukim(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaPJ(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		var req struct {
			CukimWeight float64 `json:"cukim_weight"`
			Karat       string  `json:"karat"`
			Date        string  `json:"date"`
			Note        string  `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.CukimWeight <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi berat cukim"})
			return
		}
		karat := sql.NullString{}
		if strings.TrimSpace(req.Karat) != "" {
			k, ok := normalKarat(req.Karat)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "kadar harus 1K sampai 24K"})
				return
			}
			karat = sql.NullString{String: k, Valid: true}
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
		if _, err := tx.Exec(sqlStokItem, boID, "out", "murnikan", "Leburan kotor", leburan, "",
			"Dimurnikan jadi cukim", tgl, cl.UserID, leburanItem.Int64); err != nil {
			gagal(c, err)
			return
		}
		if _, err := tx.Exec("UPDATE items SET status = 'lebur' WHERE id = ?", leburanItem.Int64); err != nil {
			gagal(c, err)
			return
		}
		cukimID, err := buatItemHasil(tx, "Cukim", karat, req.CukimWeight, boID, tgl, note,
			cl.UserID, "bersih", modal, int64(id), "cukim")
		if err != nil {
			gagal(c, err)
			return
		}
		if _, err := tx.Exec(
			`UPDATE melt_batches SET status = 'cukim', cukim_weight = ?, cukim_karat = ?, cukim_date = ?,
			        cukim_by = ?, cukim_item_id = ? WHERE id = ?`,
			req.CukimWeight, karat, tgl, cl.UserID, cukimID, id,
		); err != nil {
			gagal(c, err)
			return
		}
		catatAudit(tx, cl.UserID, "cukim", "melt_batches", int64(id), req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cukim dicatat, siap dijual ke penadah"})
	}
}

// ---------- BATALKAN LANGKAH TERAKHIR (Super Admin) ----------

func leburUndo(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		reason, ok := bacaAlasan(c)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			gagal(c, err)
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
			ids, err := ambilIDs(tx, "SELECT id FROM items WHERE melt_id = ? AND jenis <> 'hasil_lebur' AND status = 'lebur' FOR UPDATE", id)
			if err != nil {
				gagal(c, err)
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

		catatAudit(tx, cl.UserID, "lebur_undo", "melt_batches", int64(id), gin.H{"dari_tahap": status, "alasan": reason})
		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": msg})
	}
}

// ---------- DAFTAR LEBUR ----------

func leburListV2(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := bolehLihatLebur(c); !ok {
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
		case "semua":
		default:
			q += " AND b.status <> 'terjual'"
		}
		q += " ORDER BY b.id DESC LIMIT 200"

		rows, err := db.Query(q)
		if err != nil {
			gagal(c, err)
			return
		}
		defer rows.Close()

		type row struct {
			ID            int        `json:"id"`
			StartDate     time.Time  `json:"start_date"`
			Status        string     `json:"status"`
			ItemCount     int        `json:"item_count"`
			SourceWeight  float64    `json:"source_weight"`
			Modal         float64    `json:"modal"`
			CutWeight     *float64   `json:"cut_weight"`
			LeburanWeight *float64   `json:"leburan_weight"`
			CukimWeight   *float64   `json:"cukim_weight"`
			CukimKarat    string     `json:"cukim_karat"`
			Note          string     `json:"note"`
			CutNote       string     `json:"cut_note"`
			LeburanDate   *time.Time `json:"leburan_date"`
			CukimDate     *time.Time `json:"cukim_date"`
			LeburanBy     string     `json:"leburan_by"`
			CukimBy       string     `json:"cukim_by"`
			Barang        string     `json:"barang"`
			SaleTotal     *float64   `json:"sale_total"`
			Penadah       string     `json:"penadah"`
			SaleDate      *time.Time `json:"sale_date"`
			Untung        *float64   `json:"untung"`
			Susut         *float64   `json:"susut"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var cut, leb, cuk, sale sql.NullFloat64
			var lebDate, cukDate, saleDate sql.NullTime
			if err := rows.Scan(&r.ID, &r.StartDate, &r.Status, &r.ItemCount, &r.SourceWeight, &r.Modal,
				&cut, &leb, &cuk, &r.CukimKarat, &r.Note, &r.CutNote, &lebDate, &cukDate,
				&r.LeburanBy, &r.CukimBy, &r.Barang, &sale, &r.Penadah, &saleDate); err != nil {
				gagal(c, err)
				return
			}
			r.CutWeight, r.LeburanWeight, r.CukimWeight = ptrFloat(cut), ptrFloat(leb), ptrFloat(cuk)
			r.LeburanDate, r.CukimDate, r.SaleDate = ptrTime(lebDate), ptrTime(cukDate), ptrTime(saleDate)
			r.SaleTotal = ptrFloat(sale)
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

func penjualanSiap(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := bolehLihatLebur(c); !ok {
			return
		}
		boID, err := idBackOffice(db)
		if err != nil {
			gagal(c, err)
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
			gagal(c, err)
			return
		}
		defer rows.Close()

		type row struct {
			ID         int     `json:"id"`
			ItemNo     string  `json:"item_no"`
			ItemName   string  `json:"item_name"`
			Karat      string  `json:"karat"`
			WeightGram float64 `json:"weight_gram"`
			Jenis      string  `json:"jenis"`
			Brand      string  `json:"brand"`
			HargaBeli  float64 `json:"harga_beli"`
			BatchID    *int64  `json:"batch_id"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var batch sql.NullInt64
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram, &r.Jenis, &r.Brand, &r.HargaBeli, &batch); err != nil {
				gagal(c, err)
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

func penjualanPenadah(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaPJ(c)
		if !ok {
			return
		}
		var req struct {
			ItemIDs     []int   `json:"item_ids"`
			PenadahName string  `json:"penadah_name"`
			SaleTotal   float64 `json:"sale_total"`
			SaleDate    string  `json:"sale_date"`
			Note        string  `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || len(req.ItemIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		penadah := strings.TrimSpace(req.PenadahName)
		if penadah == "" || req.SaleTotal <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama penadah dan harga jual wajib diisi"})
			return
		}
		tgl, ok := parseTanggal(c, req.SaleDate)
		if !ok {
			return
		}
		boID, err := idBackOffice(db)
		if err != nil {
			gagal(c, err)
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
			gagal(c, err)
			return
		}
		defer tx.Rollback()

		rows, err := tx.Query(
			`SELECT id, item_name, jenis, status, location_id, harga_beli, weight_gram, IFNULL(karat,'')
			 FROM items WHERE is_void = FALSE AND id IN (`+placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			gagal(c, err)
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
			gagal(c, err)
			return
		}
		cashID, _ := res.LastInsertId()

		res2, err := tx.Exec(
			`INSERT INTO penadah_sales (sale_date, penadah_name, sale_total, modal, weight_gram, item_count, note, cash_id, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			tgl, penadah, req.SaleTotal, modal, berat, len(list), sql.NullString{String: note, Valid: note != ""}, cashID, cl.UserID,
		)
		if err != nil {
			gagal(c, err)
			return
		}
		saleID, _ := res2.LastInsertId()

		for _, b := range list {
			if _, err := tx.Exec(sqlStokItem, boID, "out", "jual_penadah", b.Name, b.Berat, b.Karat,
				"Jual ke penadah "+penadah, tgl, cl.UserID, b.ID); err != nil {
				gagal(c, err)
				return
			}
			if _, err := tx.Exec("UPDATE items SET status = 'terjual', sale_id = ? WHERE id = ?", saleID, b.ID); err != nil {
				gagal(c, err)
				return
			}
			if b.Jenis == "hasil_lebur" {
				tx.Exec("UPDATE melt_batches SET status = 'terjual' WHERE cukim_item_id = ? AND is_void = FALSE", b.ID)
			}
		}
		catatAudit(tx, cl.UserID, "jual_penadah", "penadah_sales", saleID, gin.H{"penadah": penadah, "harga_jual": req.SaleTotal, "modal": modal})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
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

func penjualanList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := bolehLihatLebur(c); !ok {
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
			gagal(c, err)
			return
		}
		defer rows.Close()

		type row struct {
			ID          int       `json:"id"`
			SaleDate    time.Time `json:"sale_date"`
			PenadahName string    `json:"penadah_name"`
			ItemCount   int       `json:"item_count"`
			WeightGram  float64   `json:"weight_gram"`
			Modal       float64   `json:"modal"`
			SaleTotal   float64   `json:"sale_total"`
			Untung      float64   `json:"untung"`
			Note        string    `json:"note"`
			PJ          string    `json:"pj"`
			Barang      string    `json:"barang"`
		}
		out := []row{}
		var totalModal, totalJual float64
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.ID, &r.SaleDate, &r.PenadahName, &r.ItemCount, &r.WeightGram, &r.Modal, &r.SaleTotal,
				&r.Note, &r.PJ, &r.Barang); err != nil {
				gagal(c, err)
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

func penjualanVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		reason, ok := bacaAlasan(c)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			gagal(c, err)
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

		ids, err := ambilIDs(tx, "SELECT id FROM items WHERE sale_id = ? FOR UPDATE", id)
		if err != nil {
			gagal(c, err)
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
			catatAudit(tx, cl.UserID, "void", "cash_transactions", cashID.Int64, gin.H{"reason": reason})
		}
		tx.Exec("UPDATE penadah_sales SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		catatAudit(tx, cl.UserID, "void", "penadah_sales", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "penjualan dibatalkan, barang dan kas dikembalikan"})
	}
}