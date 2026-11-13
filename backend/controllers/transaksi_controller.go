package controllers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

var stokKategori = map[string]map[string]bool{
	"cabang|in":       {"barang_masuk": true},
	"cabang|out":      {"laku": true, "pegadaian": true, "cuci": true, "lebur": true, "jual": true, "kembali_bo": true},
	"back_office|in":  {"pabrik": true, "konsinyasi": true, "cucian": true, "barang_toko": true},
	"back_office|out": {"ke_toko": true, "jual": true},
}

func StokCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req models.StokReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if !stokKategori[tipe+"|"+req.Direction][req.Category] {
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

		// Tentukan pasangan otomatis (kalau ada)
		targetID := 0
		targetKat := ""
		targetNote := ""
		switch req.Category {
		case "ke_toko":
			var tt string
			err := db.QueryRow("SELECT type FROM locations WHERE id = ?", req.TargetLocationID).Scan(&tt)
			if err != nil || tt != "cabang" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "pilih toko tujuan"})
				return
			}
			targetID = req.TargetLocationID
			targetKat = "barang_masuk"
			targetNote = "Kiriman dari Back Office"
		case "kembali_bo":
			if err := db.QueryRow("SELECT id FROM locations WHERE type='back_office' LIMIT 1").Scan(&targetID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
				return
			}
			targetKat = "kembali_cabang"
			targetNote = "Dikembalikan dari cabang"
		}
		if targetID != 0 && req.Note != "" {
			targetNote = targetNote + " - " + req.Note
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		const q = `INSERT INTO stock_movements
			(location_id, direction, category, item_name, weight_gram, karat, qty, note, transaction_date, created_by)
			VALUES (?,?,?,?,?,?,?,?,?,?)`

		res, err := tx.Exec(q, locID, req.Direction, req.Category, req.ItemName, req.WeightGram, req.Karat, req.Qty, req.Note, tgl, cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "create", "stock_movements", id, req)

		msg := "stok tersimpan"
		if targetID != 0 {
			res2, err := tx.Exec(q, targetID, "in", targetKat, req.ItemName, req.WeightGram, req.Karat, req.Qty, targetNote, tgl, cl.UserID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			id2, _ := res2.LastInsertId()
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", id2, id)
			tx.Exec("UPDATE stock_movements SET related_id = ? WHERE id = ?", id, id2)
			services.CatatAudit(tx, middleware.Meta(c), "create_auto", "stock_movements", id2, gin.H{"dari_stok_id": id})
			msg = "stok tersimpan dan otomatis tercatat di tujuan"
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": msg})
	}
}

// kunci: tipeLokasi|tabel|arah
var kategoriValid = map[string]map[string]bool{
	"cabang|kas|in":   {"omzet": true, "tambah_kas": true},
	"cabang|kas|out":  {"pengeluaran_toko": true, "pembelian_customer": true, "transfer_bo": true},
	"cabang|stok|in":  {"barang_masuk": true},
	"cabang|stok|out": {"laku": true, "pegadaian": true, "cuci": true, "lebur": true, "jual": true},

	"back_office|kas|in":   {"dari_toko": true, "jual_barang": true, "investor": true, "pinjaman": true, "jual_aset": true, "kembali_cabang": true},
	"back_office|kas|out":  {"kas_toko": true, "bayar_hutang": true, "bulanan_investor": true, "operasional": true, "kembali_cabang": true},
	"back_office|stok|in":  {"pabrik": true, "konsinyasi": true, "cucian": true, "barang_toko": true, "kembali_cabang": true},
	"back_office|stok|out": {"ke_toko": true, "lebur": true, "jual": true, "cuci": true},
}

func KasCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req models.KasReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if !kategoriValid[tipe+"|kas|"+req.Direction][req.Category] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "arah atau kategori tidak valid untuk lokasi ini"})
			return
		}
		if req.Category == "bayar_hutang" || req.Category == "investor" || req.Category == "pinjaman" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dana dari investor atau pinjaman dan pembayaran hutang diinput lewat menu Hutang"})
			return
		}
		if req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nominal harus lebih dari 0"})
			return
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
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			locID, req.Direction, req.Category, req.Amount, req.Note, tgl, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "create", "cash_transactions", id, req)

		// Otomatis: transfer kas cabang -> pemasukan "dari_toko" di back office
		if req.Category == "transfer_bo" {
			var boID int
			if err := tx.QueryRow("SELECT id FROM locations WHERE type='back_office' LIMIT 1").Scan(&boID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
				return
			}
			res2, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, related_id, created_by)
				 VALUES (?,?,?,?,?,?,?,?)`,
				boID, "in", "dari_toko", req.Amount, req.Note, tgl, id, cl.UserID,
			)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			id2, _ := res2.LastInsertId()
			tx.Exec("UPDATE cash_transactions SET related_id = ? WHERE id = ?", id2, id)
			services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", id2, gin.H{"dari_transfer_id": id})
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "kas tersimpan"})
	}
}

// kelompokKas: pengelompokan kategori kas untuk filter "Jenis transaksi" di riwayat kas.
// penadah tidak punya kategori sendiri (memakai jual_barang), jadi dikenali dari catatannya.
var kelompokKas = map[string][]string{
	"penjualan":    {"omzet", "jual_qris", "biaya_qris", "jual_barang"},
	"pembelian":    {"pembelian_customer"},
	"tukar_bp":     {"kembali_cabang"},
	"gadai":        {"gadai_dana", "gadai_pokok", "gadai_biaya", "gadai_kembali"},
	"hutang":       {"bayar_hutang", "pinjaman", "investor", "bulanan_investor", "tarik_investor"},
	"service":      {"biaya_service"},
	"operasional":  {"pengeluaran_toko", "operasional", "jual_aset"},
	"transfer_kas": {"transfer_bo", "dari_toko", "kas_toko", "tambah_kas"},
}

const catatPenadah = "(x.note LIKE 'Jual ke penadah%' OR x.note LIKE 'Potongan % jual ke penadah%')"

// filterKas membangun syarat tambahan riwayat kas dari query: from, to, arah, kelompok, metode, q.
func filterKas(c *gin.Context) (string, []interface{}) {
	q, args := "", []interface{}{}
	if v := c.Query("from"); v != "" {
		q += " AND x.transaction_date >= ?"
		args = append(args, v)
	}
	if v := c.Query("to"); v != "" {
		q += " AND x.transaction_date <= ?"
		args = append(args, v)
	}
	if v := c.Query("arah"); v == "in" || v == "out" {
		q += " AND x.direction = ?"
		args = append(args, v)
	}
	switch k := c.Query("kelompok"); {
	case k == "penadah":
		q += " AND " + catatPenadah
	case kelompokKas[k] != nil:
		q += " AND x.category IN (" + utils.Placeholders(len(kelompokKas[k])) + ")"
		for _, kat := range kelompokKas[k] {
			args = append(args, kat)
		}
		if k == "penjualan" {
			q += " AND NOT " + catatPenadah
		}
	}
	if v := c.Query("metode"); v == "tunai" || v == "transfer" || v == "qris" {
		q += " AND x.metode = ?"
		args = append(args, v)
	}
	if v := strings.TrimSpace(c.Query("q")); v != "" {
		pola := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(utils.BatasTeks(v, 60))
		q += " AND x.note LIKE ?"
		args = append(args, "%"+pola+"%")
	}
	return q, args
}

func KasList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if !utils.TanggalOpsional(c, "from", "to") {
			return
		}
		// lokasi yang dilihat. Riwayat lokasi cabang juga memuat penjualan QRIS/Debit-nya yang uangnya masuk
		// kas back office (beserta potongannya), ditandai KasLain, supaya penjualannya tidak terasa hilang.
		lokasi := c.Query("location_id")
		if cl.Role == "cabang" {
			lokasi = strconv.Itoa(cl.LocationID)
		}
		fil, filArgs := filterKas(c)
		termasukBatal := c.Query("batal") == "1"

		// halaman: 200 baris per muat, ?limit= untuk memuat lebih banyak (maksimal 2000)
		batas := 200
		if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
			batas = v
		}
		if batas > 2000 {
			batas = 2000
		}

		q := `SELECT x.id, x.location_id, x.direction, x.category, x.amount, IFNULL(x.note,''), x.transaction_date, x.created_at,
		             IFNULL(l.name,''),
		             EXISTS (SELECT 1 FROM items j WHERE (j.jual_cash_id = x.id OR j.jual_biaya_cash_id = x.id) AND j.is_void = FALSE),
		             IFNULL((SELECT g.id FROM gadai g WHERE g.cash_id = x.id AND g.is_void = FALSE LIMIT 1),
		                    IFNULL((SELECT p.gadai_id FROM gadai_pembayaran p
		                            WHERE (p.cash_pokok_id = x.id OR p.cash_biaya_id = x.id) AND p.is_void = FALSE LIMIT 1), 0)),
		             x.is_void, IFNULL(x.void_reason,'')`
		args := []interface{}{}
		if lokasi != "" {
			q += ", x.location_id <> ?"
			args = append(args, lokasi)
		} else {
			q += ", FALSE"
		}
		q += " FROM cash_transactions x JOIN locations l ON l.id = x.location_id WHERE 1=1"
		if !termasukBatal {
			q += " AND x.is_void = FALSE"
		}
		if lokasi != "" {
			q += ` AND (x.location_id = ? OR x.asal_location_id = ?
			         OR EXISTS (SELECT 1 FROM items i WHERE i.jual_biaya_cash_id = x.id AND i.location_id = ?))`
			args = append(args, lokasi, lokasi, lokasi)
		}
		q += fil
		args = append(args, filArgs...)
		q += " ORDER BY x.transaction_date DESC, x.id DESC LIMIT ?"
		args = append(args, batas+1)

		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.KasListRow{}
		for rows.Next() {
			var r models.KasListRow
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Direction, &r.Category, &r.Amount, &r.Note, &r.Date, &r.CreatedAt,
				&r.NamaKas, &r.DariPenjualan, &r.GadaiID, &r.IsVoid, &r.VoidReason, &r.KasLain); err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, r)
		}
		adaLagi := len(out) > batas
		if adaLagi {
			out = out[:batas]
		}

		// ringkasan seluruh hasil filter (bukan hanya yang tampil), tanpa transaksi dibatalkan dan tanpa baris kas lain
		tq := "SELECT IFNULL(SUM(IF(x.direction='in', x.amount, 0)),0), IFNULL(SUM(IF(x.direction='out', x.amount, 0)),0) FROM cash_transactions x WHERE x.is_void = FALSE"
		targs := []interface{}{}
		if lokasi != "" {
			tq += " AND x.location_id = ?"
			targs = append(targs, lokasi)
		}
		tq += fil
		targs = append(targs, filArgs...)
		var masuk, keluar float64
		if err := db.QueryRow(tq, targs...).Scan(&masuk, &keluar); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "total_masuk": masuk, "total_keluar": keluar, "ada_lagi": adaLagi, "batas": batas})
	}
}

// ---------- STOK ----------

func StokList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if !utils.TanggalOpsional(c, "from", "to") {
			return
		}
		q := `SELECT id, location_id, direction, category, item_name, weight_gram, IFNULL(karat,''), qty, IFNULL(note,''), transaction_date
		      FROM stock_movements WHERE is_void = FALSE`
		args := []interface{}{}

		if cl.Role == "cabang" {
			q += " AND location_id = ?"
			args = append(args, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" {
			q += " AND location_id = ?"
			args = append(args, loc)
		}
		if v := c.Query("from"); v != "" {
			q += " AND transaction_date >= ?"
			args = append(args, v)
		}
		if v := c.Query("to"); v != "" {
			q += " AND transaction_date <= ?"
			args = append(args, v)
		}
		q += " ORDER BY transaction_date DESC, id DESC LIMIT 200"

		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.StokListRow{}
		for rows.Next() {
			var r models.StokListRow
			if err := rows.Scan(&r.ID, &r.LocationID, &r.Direction, &r.Category, &r.ItemName, &r.WeightGram, &r.Karat, &r.Qty, &r.Note, &r.Date); err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}
