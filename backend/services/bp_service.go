package services

import (
	"database/sql"
	"fmt"
	"strings"
)

var kolomBP = []struct{ tabel, kolom, ddl string }{
	{"items", "bp_nama", "bp_nama VARCHAR(30) NULL"},
	// asal dan tanggal terima kiriman terakhir (label "baru datang" dan filter asal di daftar stok)
	{"items", "diterima_dari_id", "diterima_dari_id INT NULL"},
	{"items", "diterima_tgl", "diterima_tgl DATE NULL"},
	// penjualan penadah dicatat PJ dulu (konfirmasi = FALSE, belum masuk kas); data lama dianggap sudah dikonfirmasi
	{"penadah_sales", "konfirmasi", "konfirmasi BOOLEAN NOT NULL DEFAULT TRUE"},
	{"penadah_sales", "biaya", "biaya DECIMAL(15,2) NOT NULL DEFAULT 0"},
}

// SiapkanSkemaBP menyiapkan kolom untuk oper ke BP dan penjualan penadah berkonfirmasi (tabel talangan: lihat talangan_schema.go).
// Isinya sama dengan db/migrations/032_bp_talangan.sql (bisa dijalankan manual oleh admin database).
func SiapkanSkemaBP(db *sql.DB) error {
	gagal := func(apa string, err error) error {
		return fmt.Errorf("%s belum ada dan gagal dibuat (%v); jalankan db/migrations/032_bp_talangan.sql dengan user database yang punya hak ALTER dan CREATE", apa, err)
	}
	for _, k := range kolomBP {
		if kolomAda(db, k.tabel, k.kolom) {
			continue
		}
		if _, err := db.Exec("ALTER TABLE " + k.tabel + " ADD COLUMN " + k.ddl); err != nil {
			return gagal("kolom "+k.tabel+"."+k.kolom, err)
		}
	}
	var tipe string
	db.QueryRow(
		"SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'status'",
	).Scan(&tipe)
	if !strings.Contains(tipe, "di_bp") {
		if _, err := db.Exec("ALTER TABLE items MODIFY COLUMN status " + enumStatusBarang); err != nil {
			return gagal("status barang di_bp", err)
		}
	}
	return nil
}
