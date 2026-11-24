package services

import (
	"database/sql"
	"fmt"
)

// SiapkanSkemaTukarTambah membuat tabel tukar_tambah dan kolom items.tukar_tambah_id bila belum ada.
// Aman dijalankan berulang. Bila user database tidak punya hak CREATE/ALTER, backend berhenti dengan pesan jelas.
func SiapkanSkemaTukarTambah(db *sql.DB) error {
	gagal := func(apa string, err error) error {
		return fmt.Errorf("%s belum ada dan gagal dibuat (%v); jalankan dengan user database yang punya hak CREATE dan ALTER", apa, err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS tukar_tambah (
		id INT AUTO_INCREMENT PRIMARY KEY,
		location_id INT NOT NULL,
		total_lama DECIMAL(15,2) NOT NULL,
		total_baru DECIMAL(15,2) NOT NULL,
		selisih DECIMAL(15,2) NOT NULL,
		metode VARCHAR(10) NOT NULL DEFAULT 'tunai',
		biaya DECIMAL(15,2) NOT NULL DEFAULT 0,
		kas_id INT NULL,
		biaya_kas_id INT NULL,
		note VARCHAR(255) NULL,
		transaction_date DATE NOT NULL,
		created_by INT NOT NULL,
		created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
		is_void TINYINT(1) NOT NULL DEFAULT 0,
		void_reason VARCHAR(255) NULL,
		INDEX idx_tukar_tambah_lokasi (location_id, transaction_date),
		FOREIGN KEY (location_id) REFERENCES locations(id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`); err != nil && !tabelAda(db, "tukar_tambah") {
		return gagal("tabel tukar_tambah", err)
	}
	if !kolomAda(db, "tukar_tambah", "kas_id") {
		if _, err := db.Exec("ALTER TABLE tukar_tambah ADD COLUMN kas_id INT NULL"); err != nil {
			return gagal("kolom tukar_tambah.kas_id", err)
		}
	}
	if !kolomAda(db, "tukar_tambah", "biaya_kas_id") {
		if _, err := db.Exec("ALTER TABLE tukar_tambah ADD COLUMN biaya_kas_id INT NULL"); err != nil {
			return gagal("kolom tukar_tambah.biaya_kas_id", err)
		}
	}
	if !kolomAda(db, "items", "tukar_tambah_id") {
		if _, err := db.Exec("ALTER TABLE items ADD COLUMN tukar_tambah_id INT NULL, ADD INDEX idx_items_tukar_tambah (tukar_tambah_id)"); err != nil {
			return gagal("kolom items.tukar_tambah_id", err)
		}
	}
	return nil
}
