package services

import (
	"database/sql"
	"fmt"
)

// BP: cabang BP (pihak luar tanpa akun). Daftar dikelola super admin lewat tabel bp_cabang.
type BP struct {
	ID    int    `json:"id"`
	Nama  string `json:"nama"`
	Aktif bool   `json:"aktif"`
}

var bpAwal = []string{"BP Induk", "BP Pecinan", "BP Iss", "BP Lawang"}

// tabelTalanganBO: urutan penting (tabel yang dirujuk dibuat lebih dulu). Sama dengan db/migrations/033_talangan_bo.sql.
var tabelTalanganBO = []struct{ nama, sql string }{
	{"bp_cabang", `CREATE TABLE IF NOT EXISTS bp_cabang (
		id INT AUTO_INCREMENT PRIMARY KEY,
		nama VARCHAR(30) NOT NULL,
		aktif BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY uq_bp_nama (nama)
	)`},
	{"talangan_tukar", `CREATE TABLE IF NOT EXISTS talangan_tukar (
		id INT AUTO_INCREMENT PRIMARY KEY,
		cabang_id INT NOT NULL,
		total_masuk DECIMAL(15,2) NOT NULL DEFAULT 0,
		total_keluar DECIMAL(15,2) NOT NULL DEFAULT 0,
		saldo DECIMAL(15,2) NOT NULL DEFAULT 0,
		arah_uang VARCHAR(10) NOT NULL,
		metode_bayar VARCHAR(8) NULL,
		nominal_bayar DECIMAL(15,2) NULL,
		no_referensi VARCHAR(60) NOT NULL DEFAULT '',
		status_uang VARCHAR(14) NOT NULL DEFAULT 'BELUM_BAYAR',
		kas_id INT NULL,
		pj_id INT NOT NULL,
		tanggal DATE NOT NULL,
		dikonfirmasi_oleh INT NULL,
		koreksi_alasan VARCHAR(255) NULL,
		catatan VARCHAR(255) NULL,
		created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_tt_cabang (cabang_id),
		INDEX idx_tt_status (status_uang)
	)`},
	{"talangan_barang", `CREATE TABLE IF NOT EXISTS talangan_barang (
		id INT AUTO_INCREMENT PRIMARY KEY,
		arah VARCHAR(8) NOT NULL,
		cabang_pemilik_id INT NOT NULL,
		toko_input_id INT NULL,
		stok_id INT NULL,
		proses_id INT NULL,
		proses_jenis VARCHAR(10) NULL,
		nama_barang VARCHAR(100) NOT NULL,
		kategori VARCHAR(30) NOT NULL DEFAULT 'barang_cabang',
		jenis VARCHAR(20) NOT NULL DEFAULT 'perhiasan',
		bahan VARCHAR(5) NOT NULL DEFAULT 'emas',
		berat DECIMAL(10,3) NOT NULL DEFAULT 0,
		kadar VARCHAR(10) NULL,
		harga_beli DECIMAL(15,2) NOT NULL,
		ubah_harga BOOLEAN NOT NULL DEFAULT FALSE,
		tanggal_beli DATE NOT NULL,
		no_surat VARCHAR(60) NOT NULL DEFAULT '',
		no_referensi VARCHAR(60) NOT NULL DEFAULT '',
		status VARCHAR(16) NOT NULL,
		tukar_id INT NULL,
		kas_id INT NULL,
		diinput_oleh INT NOT NULL,
		dikonfirmasi_pj_oleh INT NULL,
		dikonfirmasi_owner_oleh INT NULL,
		belum_sampai BOOLEAN NOT NULL DEFAULT FALSE,
		belum_sampai_note VARCHAR(255) NULL,
		catatan VARCHAR(255) NULL,
		created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_tb_cabang (cabang_pemilik_id),
		INDEX idx_tb_status (status),
		INDEX idx_tb_arah (arah),
		INDEX idx_tb_tukar (tukar_id),
		INDEX idx_tb_stok (stok_id)
	)`},
	// log ini hanya di-INSERT; tidak ada kode yang meng-UPDATE atau DELETE
	{"talangan_status_log", `CREATE TABLE IF NOT EXISTS talangan_status_log (
		id INT AUTO_INCREMENT PRIMARY KEY,
		talangan_id INT NOT NULL,
		dari VARCHAR(16) NOT NULL DEFAULT '',
		ke VARCHAR(16) NOT NULL,
		oleh INT NOT NULL,
		waktu TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
		catatan VARCHAR(255) NULL,
		INDEX idx_tsl_barang (talangan_id)
	)`},
	{"talangan_koreksi_harga", `CREATE TABLE IF NOT EXISTS talangan_koreksi_harga (
		id INT AUTO_INCREMENT PRIMARY KEY,
		talangan_id INT NOT NULL,
		harga_lama DECIMAL(15,2) NOT NULL,
		harga_baru DECIMAL(15,2) NOT NULL,
		alasan VARCHAR(255) NOT NULL,
		oleh INT NOT NULL,
		waktu TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_tkh_barang (talangan_id)
	)`},
	{"talangan_tukar_item", `CREATE TABLE IF NOT EXISTS talangan_tukar_item (
		id INT AUTO_INCREMENT PRIMARY KEY,
		tukar_id INT NOT NULL,
		barang_id INT NOT NULL,
		arah VARCHAR(8) NOT NULL,
		nilai DECIMAL(15,2) NOT NULL,
		UNIQUE KEY uq_tukar_barang (barang_id),
		INDEX idx_tti_tukar (tukar_id)
	)`},
}

// Tabel fitur talangan lama (catatan manual per BP) diganti, tapi datanya tetap disimpan sebagai arsip.
var tabelArsipTalangan = []struct{ lama, arsip, ddl string }{
	{"talangan", "talangan_lama", `CREATE TABLE IF NOT EXISTS talangan_lama (
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
	{"tukar_bp", "tukar_bp_lama", `CREATE TABLE IF NOT EXISTS tukar_bp_lama (
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
}

// SiapkanSkemaTalanganBO mengarsipkan tabel talangan lama (RENAME, tanpa menghapus data), membuat tabel baru,
// mengisi daftar BP awal, dan menambah items.talangan_id. Aman dijalankan berulang.
// Isinya sama dengan db/migrations/033_talangan_bo.sql. BACKUP DATABASE sebelum pertama kali dijalankan.
func SiapkanSkemaTalanganBO(db *sql.DB) error {
	gagal := func(apa string, err error) error {
		return fmt.Errorf("%s belum ada dan gagal dibuat (%v); jalankan db/migrations/033_talangan_bo.sql dengan user database yang punya hak ALTER, CREATE, dan DROP", apa, err)
	}
	for _, a := range tabelArsipTalangan {
		switch {
		case tabelAda(db, a.arsip):
			// sudah diarsipkan; tabel lama yang muncul lagi (mis. migrasi 032 dijalankan ulang) dibiarkan
		case tabelAda(db, a.lama):
			if _, err := db.Exec("RENAME TABLE " + a.lama + " TO " + a.arsip); err != nil {
				return gagal("arsip tabel "+a.arsip, err)
			}
		default:
			if _, err := db.Exec(a.ddl); err != nil && !tabelAda(db, a.arsip) {
				return gagal("tabel "+a.arsip, err)
			}
		}
	}
	for _, t := range tabelTalanganBO {
		if _, err := db.Exec(t.sql); err != nil && !tabelAda(db, t.nama) {
			return gagal("tabel "+t.nama, err)
		}
	}
	for _, n := range bpAwal {
		if _, err := db.Exec("INSERT IGNORE INTO bp_cabang (nama) VALUES (?)", n); err != nil {
			return gagal("isi bp_cabang", err)
		}
	}
	if !kolomAda(db, "talangan_barang", "belum_sampai") {
		if _, err := db.Exec("ALTER TABLE talangan_barang ADD COLUMN belum_sampai BOOLEAN NOT NULL DEFAULT FALSE, ADD COLUMN belum_sampai_note VARCHAR(255) NULL"); err != nil {
			return gagal("kolom talangan_barang.belum_sampai", err)
		}
	}
	if !kolomAda(db, "items", "talangan_id") {
		if _, err := db.Exec("ALTER TABLE items ADD COLUMN talangan_id INT NULL, ADD INDEX idx_items_talangan (talangan_id)"); err != nil {
			return gagal("kolom items.talangan_id", err)
		}
	}
	return nil
}

// DaftarBPAktif: BP yang boleh dipilih (nonaktif tidak ditawarkan, tapi datanya tetap terbaca).
func DaftarBPAktif(db *sql.DB) ([]BP, error) {
	rows, err := db.Query("SELECT id, nama, aktif FROM bp_cabang WHERE aktif = TRUE ORDER BY nama")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hasil []BP
	for rows.Next() {
		var b BP
		if err := rows.Scan(&b.ID, &b.Nama, &b.Aktif); err != nil {
			return nil, err
		}
		hasil = append(hasil, b)
	}
	return hasil, rows.Err()
}

// BPValid: nama BP ada di bp_cabang dan masih aktif (dipakai oper barang ke BP).
func BPValid(db *sql.DB, nama string) bool {
	var n int
	db.QueryRow("SELECT COUNT(*) FROM bp_cabang WHERE nama = ? AND aktif = TRUE", nama).Scan(&n)
	return n > 0
}
