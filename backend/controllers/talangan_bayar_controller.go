package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

const (
	uangBelum    = "BELUM_BAYAR"
	uangPJ       = "DI_TANGAN_PJ"
	uangMasuk    = "MASUK_KAS_BO"
	katKasMasuk  = "talangan_tukar_masuk"
	katKasKeluar = "talangan_tukar_keluar"
)

type tukarKunci struct {
	ID           int
	BP           string
	Saldo        int64
	StatusUang   string
	NominalBayar sql.NullInt64
	Koreksi      string
}

// kunciTukar mengunci baris penukaran (FOR UPDATE) supaya klik ganda atau permintaan serentak
// diperiksa ulang terhadap status uang yang terbaru di dalam transaksi yang sama.
func kunciTukar(c *gin.Context, tx *sql.Tx, id int) (t tukarKunci, ok bool) {
	var koreksi sql.NullString
	err := tx.QueryRow(
		`SELECT x.id, bp.nama, CAST(x.saldo AS SIGNED), x.status_uang, CAST(x.nominal_bayar AS SIGNED), x.koreksi_alasan
		 FROM talangan_tukar x JOIN bp_cabang bp ON bp.id = x.cabang_id WHERE x.id = ? FOR UPDATE`, id).
		Scan(&t.ID, &t.BP, &t.Saldo, &t.StatusUang, &t.NominalBayar, &koreksi)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "penukaran tidak ditemukan"})
		return t, false
	}
	if err != nil {
		utils.Gagal(c, err)
		return t, false
	}
	t.Koreksi = koreksi.String
	return t, true
}

func (t tukarKunci) nominalDiharapkan() int64 {
	if t.Koreksi != "" && t.NominalBayar.Valid {
		return t.NominalBayar.Int64
	}
	if t.Saldo < 0 {
		return -t.Saldo
	}
	return t.Saldo
}

func konflikUang(c *gin.Context, t tukarKunci, perlu string) {
	c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("status uang penukaran %s, bukan %s", t.StatusUang, perlu)})
}

func periksaNominal(c *gin.Context, t tukarKunci, nominal int64) bool {
	if t.Saldo == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "penukaran impas, tidak ada uang yang perlu dibayar"})
		return false
	}
	if nominal != t.nominalDiharapkan() {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("nominal harus %s sesuai saldo penukaran (owner dapat mengoreksi dengan alasan)", utils.RupiahTeks(float64(t.nominalDiharapkan())))})
		return false
	}
	return true
}

// masukKasBO mencatat uang di kas BO (masuk bila BP bayar, keluar bila kita bayar), lalu semua barang penukaran LUNAS.
func masukKasBO(c *gin.Context, tx *sql.Tx, t tukarKunci, nominal int64, metode, ket string, cl *models.Claims) bool {
	boID, ok := idBOTx(c, tx)
	if !ok {
		return false
	}
	arah, kat := "in", katKasMasuk
	if t.Saldo < 0 {
		arah, kat = "out", katKasKeluar
	}
	tgl := time.Now().Format("2006-01-02")
	res, err := tx.Exec(
		`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by, metode)
		 VALUES (?,?,?,?,?,?,?,?)`,
		boID, arah, kat, nominal, fmt.Sprintf("Tukar talangan #%d %s (%s)", t.ID, t.BP, ket), tgl, cl.UserID, metode)
	if err != nil {
		utils.Gagal(c, err)
		return false
	}
	kasID, _ := res.LastInsertId()
	if _, err := tx.Exec("UPDATE talangan_tukar SET status_uang = ?, kas_id = ?, nominal_bayar = ? WHERE id = ?", uangMasuk, kasID, nominal, t.ID); err != nil {
		utils.Gagal(c, err)
		return false
	}
	services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", kasID, gin.H{"talangan_tukar": t.ID, "nominal": nominal, "arah": arah})

	rows, err := tx.Query("SELECT barang_id, arah FROM talangan_tukar_item WHERE tukar_id = ?", t.ID)
	if err != nil {
		utils.Gagal(c, err)
		return false
	}
	type br struct {
		id   int
		arah string
	}
	var daftar []br
	for rows.Next() {
		var b br
		if err := rows.Scan(&b.id, &b.arah); err != nil {
			rows.Close()
			utils.Gagal(c, err)
			return false
		}
		daftar = append(daftar, b)
	}
	rows.Close()
	for _, b := range daftar {
		if err := langkahOtomatis(tx, b.id, b.arah, services.StDihitung, services.StLunas, cl.UserID, "uang penukaran masuk kas BO"); err != nil {
			utils.Gagal(c, err)
			return false
		}
	}
	return true
}

// TukarTerimaPJ: cara 1, PJ menerima/menyerahkan uang (DI_TANGAN_PJ). Kas BO belum berubah.
func TukarTerimaPJ(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "pj", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.TukarBayarReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nominal wajib diisi"})
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		t, ok := kunciTukar(c, tx, id)
		if !ok {
			return
		}
		if t.StatusUang != uangBelum {
			konflikUang(c, t, uangBelum)
			return
		}
		if !periksaNominal(c, t, req.Nominal) {
			return
		}
		if _, err := tx.Exec("UPDATE talangan_tukar SET status_uang = ?, metode_bayar = 'via_pj', nominal_bayar = ? WHERE id = ?", uangPJ, req.Nominal, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "talangan_tukar_terima_pj", "talangan_tukar", int64(id), gin.H{"nominal": req.Nominal, "oleh": cl.UserID})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "uang tercatat di tangan PJ, kas BO berubah setelah disetor"})
	}
}

// TukarSetorBO: cara 1 lanjutan, PJ menyetor ke BO (MASUK_KAS_BO): kas BO berubah dan barang LUNAS.
func TukarSetorBO(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "pj", "super_admin")
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
		t, ok := kunciTukar(c, tx, id)
		if !ok {
			return
		}
		if t.StatusUang != uangPJ || !t.NominalBayar.Valid {
			konflikUang(c, t, uangPJ)
			return
		}
		if !masukKasBO(c, tx, t, t.NominalBayar.Int64, "tunai", "via PJ", cl) {
			return
		}
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "kas BO diperbarui, barang penukaran LUNAS"})
	}
}

// TukarBayarTransfer: cara 2, transfer langsung ke/dari BO. Kas BO langsung berubah dan barang LUNAS.
// no_referensi boleh kosong; penukaran seperti itu ditandai tanpa_bukti agar owner bisa mengecek.
func TukarBayarTransfer(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "pj", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.TukarBayarReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nominal wajib diisi"})
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		t, ok := kunciTukar(c, tx, id)
		if !ok {
			return
		}
		if t.StatusUang != uangBelum {
			konflikUang(c, t, uangBelum)
			return
		}
		if !periksaNominal(c, t, req.Nominal) {
			return
		}
		if _, err := tx.Exec("UPDATE talangan_tukar SET metode_bayar = 'transfer', no_referensi = ? WHERE id = ?", strings.TrimSpace(req.NoReferensi), id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if !masukKasBO(c, tx, t, req.Nominal, "transfer", "transfer", cl) {
			return
		}
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "kas BO diperbarui, barang penukaran LUNAS"})
	}
}

// TukarKoreksiNominal: owner menyetujui nominal yang berbeda dari saldo (alasan wajib), sebelum uang masuk kas BO.
func TukarKoreksiNominal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "owner", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.KoreksiNominalReq
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Alasan) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nominal dan alasan wajib diisi"})
			return
		}
		req.Alasan = strings.TrimSpace(req.Alasan)
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		t, ok := kunciTukar(c, tx, id)
		if !ok {
			return
		}
		if t.StatusUang != uangBelum && t.StatusUang != uangPJ {
			c.JSON(http.StatusConflict, gin.H{"error": "uang penukaran sudah masuk kas BO atau tidak ada yang perlu dibayar; tidak bisa dikoreksi"})
			return
		}
		if _, err := tx.Exec("UPDATE talangan_tukar SET nominal_bayar = ?, koreksi_alasan = ?, dikonfirmasi_oleh = ? WHERE id = ?", req.Nominal, req.Alasan, cl.UserID, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "talangan_tukar_koreksi_nominal", "talangan_tukar", int64(id),
			gin.H{"nominal": t.nominalDiharapkan()}, gin.H{"nominal": req.Nominal, "alasan": req.Alasan})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "nominal dikoreksi"})
	}
}
