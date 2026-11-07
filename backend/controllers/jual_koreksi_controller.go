package controllers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// Pembatalan dan koreksi penjualan dari menu Stok (bukan penjualan ke penadah).
// Akun cabang hanya untuk barang di lokasinya sendiri dan penjualan yang tanggalnya paling lama 7 hari lalu;
// Super Admin dan Owner tanpa batas. Koreksi = batalkan penjualan lama lalu catat penjualan baru dalam satu transaksi.

type jualLama struct {
	Lokasi         int
	Nama, Karat    string
	Berat          float64
	Status, Jenis  string
	Harga          float64
	SaleID         sql.NullInt64
	KasID, BiayaID sql.NullInt64
	Baru           bool
}

// kunciJual mengunci satu barang terjual beserta catatan kas penjualannya.
func kunciJual(tx *sql.Tx, id int) (jualLama, error) {
	var j jualLama
	err := tx.QueryRow(
		`SELECT location_id, item_name, IFNULL(karat,''), weight_gram, status, jenis, IFNULL(harga_jual,0), sale_id,
		        jual_cash_id, jual_biaya_cash_id, IFNULL(tanggal_jual >= DATE_SUB(CURDATE(), INTERVAL 7 DAY), 0)
		 FROM items WHERE id = ? AND is_void = FALSE FOR UPDATE`, id,
	).Scan(&j.Lokasi, &j.Nama, &j.Karat, &j.Berat, &j.Status, &j.Jenis, &j.Harga, &j.SaleID, &j.KasID, &j.BiayaID, &j.Baru)
	return j, err
}

// periksaJual memastikan penjualan ini boleh dibatalkan atau dikoreksi oleh akun ini; bila tidak, membalas dan false.
func periksaJual(c *gin.Context, cl *models.Claims, j jualLama) bool {
	switch {
	case j.Status != "terjual":
		c.JSON(http.StatusBadRequest, gin.H{"error": "barang ini tidak berstatus terjual"})
	case j.SaleID.Valid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "penjualan ke penadah dibatalkan dari menu Lebur"})
	case !j.KasID.Valid:
		c.JSON(http.StatusBadRequest, gin.H{"error": "penjualan ini dicatat tanpa catatan kas, jadi tidak bisa dibatalkan otomatis"})
	case !bolehUbahTransaksi(cl, j.Lokasi, j.Baru):
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("bukan barang lokasi kamu atau penjualan lebih dari %d hari lalu, minta Super Admin atau Owner yang mengubahnya", services.BatasHariUbahGadai)})
	default:
		return true
	}
	return false
}

// kurangiKas mengurangi catatan kas penjualan sebesar harga barang yang dibatalkan; habis berarti dibatalkan penuh.
// Potongan QRIS/Debit (bila ada) dikurangi sebanding dengan sisa penjualan. Mengembalikan sisa kas penjualan.
func kurangiKas(tx *sql.Tx, meta services.MetaAudit, alasan string, j jualLama) error {
	var amt float64
	if err := tx.QueryRow("SELECT amount FROM cash_transactions WHERE id = ? FOR UPDATE", j.KasID.Int64).Scan(&amt); err != nil {
		return err
	}
	sisa := amt - j.Harga
	habis := sisa < 0.005
	if habis {
		if _, err := tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", alasan, j.KasID.Int64); err != nil {
			return err
		}
	} else if _, err := tx.Exec("UPDATE cash_transactions SET amount = ? WHERE id = ?", sisa, j.KasID.Int64); err != nil {
		return err
	}
	services.CatatAudit(tx, meta, "koreksi_kas", "cash_transactions", j.KasID.Int64, gin.H{"dari": amt, "menjadi": math.Max(sisa, 0), "alasan": alasan})

	if !j.BiayaID.Valid {
		return nil
	}
	var biaya float64
	if err := tx.QueryRow("SELECT amount FROM cash_transactions WHERE id = ? AND is_void = FALSE FOR UPDATE", j.BiayaID.Int64).Scan(&biaya); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	if habis || amt <= 0 {
		if _, err := tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", alasan, j.BiayaID.Int64); err != nil {
			return err
		}
	} else {
		baru := math.Round(biaya * sisa / amt)
		if _, err := tx.Exec("UPDATE cash_transactions SET amount = ? WHERE id = ?", baru, j.BiayaID.Int64); err != nil {
			return err
		}
	}
	services.CatatAudit(tx, meta, "koreksi_kas", "cash_transactions", j.BiayaID.Int64, gin.H{"potongan_dari": biaya, "alasan": alasan})
	return nil
}

// batalkanJual mengembalikan barang menjadi tersedia dan membereskan stok serta kas penjualannya.
func batalkanJual(tx *sql.Tx, meta services.MetaAudit, id int, j jualLama, alasan string) error {
	var stokID sql.NullInt64
	err := tx.QueryRow(
		`SELECT id FROM stock_movements WHERE item_id = ? AND direction = 'out' AND category IN ('laku','jual') AND is_void = FALSE
		 ORDER BY id DESC LIMIT 1 FOR UPDATE`, id).Scan(&stokID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if stokID.Valid {
		if _, err := tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE id = ?", alasan, stokID.Int64); err != nil {
			return err
		}
	}
	if err := kurangiKas(tx, meta, alasan, j); err != nil {
		return err
	}
	_, err = tx.Exec(
		`UPDATE items SET status = 'tersedia', harga_jual = NULL, tanggal_jual = NULL, jual_cash_id = NULL, jual_biaya_cash_id = NULL WHERE id = ?`, id)
	return err
}

// ---------- BATALKAN PENJUALAN ----------

func JualBatal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
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

		j, err := kunciJual(tx, id)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "barang tidak ditemukan"})
			return
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if !periksaJual(c, cl, j) {
			return
		}
		if err := batalkanJual(tx, middleware.Meta(c), id, j, reason); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "batal_jual", "items", int64(id),
			gin.H{"status": "terjual", "harga_jual": j.Harga}, gin.H{"status": "tersedia", "alasan": reason})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Penjualan %s dibatalkan, barang kembali tersedia dan kas dikoreksi", j.Nama)})
	}
}

// ---------- KOREKSI PENJUALAN (harga, tanggal, metode, potongan) ----------

func JualKoreksi(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req struct {
			Harga  float64 `json:"harga" binding:"gte=0,lte=100000000000"`
			Date   string  `json:"date" binding:"max=10"`
			Metode string  `json:"metode" binding:"max=10"`
			Biaya  float64 `json:"biaya" binding:"gte=0,lte=100000000000"`
			Reason string  `json:"reason" binding:"max=250"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data koreksi tidak valid"})
			return
		}
		req.Reason = strings.TrimSpace(req.Reason)
		if req.Reason == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan koreksi wajib diisi"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		metode, okMetode := services.NormalMetode(req.Metode)
		if !okMetode {
			c.JSON(http.StatusBadRequest, gin.H{"error": "metode pembayaran tidak dikenal"})
			return
		}
		if err := services.ValidasiBiaya(metode, req.Biaya, req.Harga); err != nil {
			balasKesalahan(c, err)
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

		j, err := kunciJual(tx, id)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "barang tidak ditemukan"})
			return
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if !periksaJual(c, cl, j) {
			return
		}
		// harga 0 (gratis) hanya untuk tempat dan kotak perhiasan
		if req.Harga == 0 && !services.BolehGratis(j.Jenis) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "harga jual wajib diisi (lebih dari 0)"})
			return
		}
		meta := middleware.Meta(c)
		if err := batalkanJual(tx, meta, id, j, "Koreksi penjualan: "+req.Reason); err != nil {
			utils.Gagal(c, err)
			return
		}

		// catat penjualan baru untuk barang ini (aturan kas sama dengan BarangJual)
		kat := "laku"
		if j.Lokasi == boID {
			kat = "jual"
		}
		if _, err := tx.Exec(services.SQLStokItem, j.Lokasi, "out", kat, j.Nama, j.Berat, j.Karat,
			"Terjual (koreksi)", tgl, cl.UserID, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec("UPDATE items SET status = 'terjual', harga_jual = ?, tanggal_jual = ? WHERE id = ?", req.Harga, tgl, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		kasLokasi, kategoriKas := j.Lokasi, "omzet"
		if j.Lokasi == boID {
			kategoriKas = "jual_barang"
		}
		catatan := "Penjualan 1 barang (koreksi)"
		var asal sql.NullInt64
		if metode == services.MetodeQRIS {
			catatan = "Penjualan 1 barang (" + services.LabelMetode(metode) + ", koreksi)"
			if j.Lokasi != boID {
				var namaAsal string
				tx.QueryRow("SELECT name FROM locations WHERE id = ?", j.Lokasi).Scan(&namaAsal)
				kasLokasi, kategoriKas = boID, services.KategoriJualQRIS
				asal = sql.NullInt64{Int64: int64(j.Lokasi), Valid: true}
				catatan += " dari " + namaAsal
			}
		} else if metode == services.MetodeTransfer {
			catatan = "Penjualan 1 barang (" + services.LabelMetode(metode) + ", koreksi)"
		}
		rk, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by, metode, asal_location_id)
			 VALUES (?, 'in', ?, ?, ?, ?, ?, ?, ?)`,
			kasLokasi, kategoriKas, req.Harga, catatan, tgl, cl.UserID, metode, asal)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		kasID, _ := rk.LastInsertId()
		if _, err := tx.Exec("UPDATE items SET jual_cash_id = ? WHERE id = ?", kasID, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, meta, "create_auto", "cash_transactions", kasID, gin.H{"koreksi_penjualan": id, "total": req.Harga, "metode": metode})
		if req.Biaya > 0 {
			rb, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by, metode)
				 VALUES (?, 'out', ?, ?, ?, ?, ?, ?)`,
				boID, services.KategoriBiayaQRIS, req.Biaya, "Potongan "+services.LabelMetode(metode)+" penjualan 1 barang (koreksi)", tgl, cl.UserID, metode)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			biayaID, _ := rb.LastInsertId()
			if _, err := tx.Exec("UPDATE items SET jual_biaya_cash_id = ? WHERE id = ?", biayaID, id); err != nil {
				utils.Gagal(c, err)
				return
			}
			services.CatatAudit(tx, meta, "create_auto", "cash_transactions", biayaID, gin.H{"biaya": req.Biaya, "metode": metode})
		}
		services.CatatPerubahan(tx, meta, "koreksi_jual", "items", int64(id),
			gin.H{"harga_jual": j.Harga}, gin.H{"harga_jual": req.Harga, "tanggal": tgl, "metode": metode, "biaya": req.Biaya, "alasan": req.Reason})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Penjualan %s dikoreksi menjadi %s", j.Nama, services.FmtRp(req.Harga))})
	}
}
