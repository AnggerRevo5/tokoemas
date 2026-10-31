package services

import (
	"database/sql"
	"fmt"
	"strings"
)

// Isinya sama dengan db/migrations/024_gadai.sql (bisa dijalankan manual oleh admin database).
var tabelGadai = []struct{ nama, sql string }{
	{"gadai", `CREATE TABLE IF NOT EXISTS gadai (
		id INT AUTO_INCREMENT PRIMARY KEY,
		item_id INT NOT NULL,
		location_id INT NOT NULL,
		pegadaian VARCHAR(100) NOT NULL,
		dana BIGINT NOT NULL,
		tenor INT NOT NULL,
		cicilan BIGINT NOT NULL,
		total BIGINT NOT NULL,
		start_date DATE NOT NULL,
		first_due DATE NOT NULL,
		status ENUM('aktif','lunas','kembali','tidak_ditebus') NOT NULL DEFAULT 'aktif',
		note VARCHAR(255) NULL,
		cash_id INT NULL,
		out_stock_id INT NULL,
		in_stock_id INT NULL,
		end_date DATE NULL,
		end_note VARCHAR(255) NULL,
		is_void BOOLEAN NOT NULL DEFAULT FALSE,
		void_reason VARCHAR(255) NULL,
		created_by INT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_gadai_item (item_id),
		INDEX idx_gadai_lokasi (location_id, status),
		FOREIGN KEY (item_id) REFERENCES items(id),
		FOREIGN KEY (location_id) REFERENCES locations(id),
		FOREIGN KEY (created_by) REFERENCES users(id)
	)`},
	{"gadai_pembayaran", `CREATE TABLE IF NOT EXISTS gadai_pembayaran (
		id INT AUTO_INCREMENT PRIMARY KEY,
		gadai_id INT NOT NULL,
		jenis ENUM('cicilan','pelunasan') NOT NULL,
		tx_date DATE NOT NULL,
		amount BIGINT NOT NULL,
		denda BIGINT NOT NULL DEFAULT 0,
		diskon BIGINT NOT NULL DEFAULT 0,
		pokok BIGINT NOT NULL DEFAULT 0,
		biaya BIGINT NOT NULL DEFAULT 0,
		cash_pokok_id INT NULL,
		cash_biaya_id INT NULL,
		note VARCHAR(255) NULL,
		is_void BOOLEAN NOT NULL DEFAULT FALSE,
		void_reason VARCHAR(255) NULL,
		created_by INT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_gadai_bayar (gadai_id, tx_date),
		FOREIGN KEY (gadai_id) REFERENCES gadai(id),
		FOREIGN KEY (created_by) REFERENCES users(id)
	)`},
}

const enumStatusBarang = "ENUM('tersedia','dikirim','cuci','lebur','terjual','hasil','kembali','digadai','tidak_ditebus') NOT NULL DEFAULT 'tersedia'"

// SiapkanSkemaGadai membuat tabel gadai dan menambah status digadai dan tidak_ditebus pada barang bila belum ada.
func SiapkanSkemaGadai(db *sql.DB) error {
	for _, t := range tabelGadai {
		if _, err := db.Exec(t.sql); err != nil && !tabelAda(db, t.nama) {
			return fmt.Errorf("tabel %s belum ada dan gagal dibuat (%v); jalankan db/migrations/024_gadai.sql dengan user database yang punya hak CREATE", t.nama, err)
		}
	}
	var tipe string
	db.QueryRow(
		"SELECT COLUMN_TYPE FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'status'",
	).Scan(&tipe)
	if !strings.Contains(tipe, "digadai") {
		if _, err := db.Exec("ALTER TABLE items MODIFY COLUMN status " + enumStatusBarang); err != nil {
			return fmt.Errorf("status barang digadai belum bisa ditambahkan (%v); jalankan db/migrations/024_gadai.sql dengan user database yang punya hak ALTER", err)
		}
	}
	return nil
}
