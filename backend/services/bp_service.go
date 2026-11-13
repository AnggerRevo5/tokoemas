package services

import (
	"database/sql"
	"fmt"
	"strings"
)

// DaftarBP: pihak luar (tanpa akun, tanpa stok di aplikasi) tujuan oper barang dan mitra tukar.
// shortcut: daftar tetap di kode, ubah jadi tabel bila nama BP perlu diatur dari aplikasi.
var DaftarBP = []string{"BP Induk", "BP Pecinan", "BP Iss", "BP Lawang"}

func BPValid(nama string) bool {
	for _, b := range DaftarBP {
		if b == nama {
			return true
		}
	}
	return false
}

var tabelBP = []struct{ nama, sql string }{
	{"tukar_bp", `CREATE TABLE IF NOT EXISTS tukar_bp (
		id INT AUTO_INCREMENT PRIMARY KEY,
		bp_nama VARCHAR(30) NOT NULL,
		location_id INT NOT NULL,
		tukar_date DATE NOT NULL,
		total_kita DECIMAL(15,2) NOT NULL DEFAULT 0,
		total_bp DECIMAL(15,2) NOT NULL DEFAULT 0,
		selisih DECIMAL(15,2) NOT NULL DEFAULT 0,
		status VARCHAR(12) NOT NULL DEFAULT 'diajukan',
		cash_id INT NULL,
		note VARCHAR(255) NULL,
		batal_alasan VARCHAR(255) NULL,
		created_by INT NOT NULL,
		approved_by INT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_tukar_bp (bp_nama, status)
	)`},
	{"talangan", `CREATE TABLE IF NOT EXISTS talangan (
		id INT AUTO_INCREMENT PRIMARY KEY,
		location_id INT NOT NULL,
		bp_nama VARCHAR(30) NOT NULL,
		arah VARCHAR(12) NOT NULL,
		item_name VARCHAR(100) NOT NULL,
		weight_gram DECIMAL(10,3) NOT NULL DEFAULT 0,
		karat VARCHAR(10) NULL,
		nilai DECIMAL(15,2) NOT NULL,
		entry_date DATE NOT NULL,
		note VARCHAR(255) NULL,
		konfirmasi BOOLEAN NOT NULL DEFAULT FALSE,
		konfirmasi_note VARCHAR(255) NULL,
		konfirmasi_tgl DATE NULL,
		status VARCHAR(12) NOT NULL DEFAULT 'aktif',
		tukar_id INT NULL,
		batal_alasan VARCHAR(255) NULL,
		created_by INT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_talangan (location_id, bp_nama, status)
	)`},
}

var kolomBP = []struct{ tabel, kolom, ddl string }{
	{"items", "bp_nama", "bp_nama VARCHAR(30) NULL"},
	// asal dan tanggal terima kiriman terakhir (label "baru datang" dan filter asal di daftar stok)
	{"items", "diterima_dari_id", "diterima_dari_id INT NULL"},
	{"items", "diterima_tgl", "diterima_tgl DATE NULL"},
	// penjualan penadah dicatat PJ dulu (konfirmasi = FALSE, belum masuk kas); data lama dianggap sudah dikonfirmasi
	{"penadah_sales", "konfirmasi", "konfirmasi BOOLEAN NOT NULL DEFAULT TRUE"},
	{"penadah_sales", "biaya", "biaya DECIMAL(15,2) NOT NULL DEFAULT 0"},
}

// SiapkanSkemaBP menyiapkan kolom dan tabel untuk oper ke BP, talangan, tukar BP, dan penjualan penadah berkonfirmasi.
// Isinya sama dengan db/migrations/032_bp_talangan.sql (bisa dijalankan manual oleh admin database).
func SiapkanSkemaBP(db *sql.DB) error {
	gagal := func(apa string, err error) error {
		return fmt.Errorf("%s belum ada dan gagal dibuat (%v); jalankan db/migrations/032_bp_talangan.sql dengan user database yang punya hak ALTER dan CREATE", apa, err)
	}
	for _, t := range tabelBP {
		if _, err := db.Exec(t.sql); err != nil && !tabelAda(db, t.nama) {
			return gagal("tabel "+t.nama, err)
		}
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
