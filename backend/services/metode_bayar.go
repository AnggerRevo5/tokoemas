package services

import (
	"database/sql"
	"fmt"
	"strings"
)

// Metode pembayaran penjualan. Tunai dan transfer sama-sama tetap di kas lokasi yang menjual;
// hanya QRIS/Debit yang masuk ke kas back office.
const (
	MetodeTunai    = "tunai"
	MetodeTransfer = "transfer"
	MetodeQRIS     = "qris"
)

// KategoriJualQRIS adalah kategori kas back office untuk penjualan QRIS/Debit dari toko cabang.
// KategoriBiayaQRIS adalah potongan (biaya admin) yang dipotong dari uang QRIS/Debit.
const (
	KategoriJualQRIS  = "jual_qris"
	KategoriBiayaQRIS = "biaya_qris"
)

// NormalMetode mengembalikan metode yang valid. Kosong dianggap tunai.
func NormalMetode(m string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "", MetodeTunai:
		return MetodeTunai, true
	case MetodeTransfer:
		return MetodeTransfer, true
	case MetodeQRIS:
		return MetodeQRIS, true
	}
	return "", false
}

// LabelMetode adalah nama metode untuk catatan kas dan pesan.
func LabelMetode(m string) string {
	switch m {
	case MetodeQRIS:
		return "QRIS/Debit"
	case MetodeTransfer:
		return "Transfer"
	}
	return "Tunai"
}

// ValidasiBiaya memastikan potongan hanya dipakai untuk QRIS/Debit dan tidak melebihi nilai penjualan.
func ValidasiBiaya(metode string, biaya, total float64) error {
	if biaya < 0 {
		return salah("potongan tidak boleh negatif")
	}
	if biaya == 0 {
		return nil
	}
	if metode != MetodeQRIS {
		return salah("potongan hanya untuk pembayaran QRIS/Debit")
	}
	if biaya > total {
		return salah(fmt.Sprintf("potongan (%s) melebihi total penjualan (%s)", FmtRp(biaya), FmtRp(total)))
	}
	return nil
}

// Isinya sama dengan db/migrations/022_metode_bayar.sql (bisa dijalankan manual oleh admin database).
var kolomMetode = []struct{ tabel, kolom, ddl string }{
	{"cash_transactions", "metode", "metode VARCHAR(10) NOT NULL DEFAULT 'tunai'"},
	{"cash_transactions", "asal_location_id", "asal_location_id INT NULL"},
	{"penadah_sales", "metode", "metode VARCHAR(10) NOT NULL DEFAULT 'tunai'"},
	{"penadah_sales", "biaya_cash_id", "biaya_cash_id INT NULL"},
	// potongan QRIS/Debit penjualan dari Stok, supaya ikut dikoreksi saat penjualan dibatalkan
	// (isinya sama dengan db/migrations/025_jual_biaya.sql)
	{"items", "jual_biaya_cash_id", "jual_biaya_cash_id INT NULL"},
}

// SiapkanSkemaMetode menambah kolom metode pembayaran bila belum ada. Data lama dianggap tunai.
func SiapkanSkemaMetode(db *sql.DB) error {
	for _, k := range kolomMetode {
		if kolomAda(db, k.tabel, k.kolom) {
			continue
		}
		if _, err := db.Exec("ALTER TABLE " + k.tabel + " ADD COLUMN " + k.ddl); err != nil {
			return fmt.Errorf("kolom %s.%s belum ada dan gagal dibuat (%v); jalankan db/migrations/022_metode_bayar.sql dengan user database yang punya hak ALTER", k.tabel, k.kolom, err)
		}
	}
	return nil
}
