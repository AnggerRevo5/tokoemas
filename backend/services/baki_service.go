package services

import (
	"database/sql"
	"fmt"
)

// KapasitasBaki adalah jumlah maksimal barang dalam satu baki.
const KapasitasBaki = 100

// Isinya sama dengan db/migrations/020_baki.sql (bisa dijalankan manual oleh admin database).
const sqlTabelBaki = `CREATE TABLE IF NOT EXISTS trays (
	id INT AUTO_INCREMENT PRIMARY KEY,
	location_id INT NOT NULL,
	no INT NOT NULL,
	created_by INT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	UNIQUE KEY uq_baki_lokasi (location_id, no),
	FOREIGN KEY (location_id) REFERENCES locations(id)
)`

// SiapkanSkemaBaki membuat tabel baki dan kolom items.tray_id bila belum ada.
func SiapkanSkemaBaki(db *sql.DB) error {
	if _, err := db.Exec(sqlTabelBaki); err != nil && !tabelAda(db, "trays") {
		return fmt.Errorf("tabel trays belum ada dan gagal dibuat (%v); jalankan db/migrations/020_baki.sql dengan user database yang punya hak CREATE", err)
	}
	if !kolomAda(db, "items", "tray_id") {
		if _, err := db.Exec("ALTER TABLE items ADD COLUMN tray_id INT NULL"); err != nil {
			return fmt.Errorf("kolom items.tray_id belum ada dan gagal dibuat (%v); jalankan db/migrations/020_baki.sql dengan user database yang punya hak ALTER", err)
		}
	}
	return nil
}

// sqlIsiBaki menghitung barang yang masih memenuhi tempat di baki: masih milik lokasi baki dan belum keluar (terjual/lebur/hasil).
const sqlIsiBaki = `SELECT COUNT(*) FROM items
	WHERE tray_id = ? AND location_id = ? AND is_void = FALSE AND status NOT IN ('terjual','lebur','hasil')`

// SisaBaki mengunci baki dan mengembalikan lokasi serta sisa tempatnya. Baki harus milik lokasi yang diminta.
func SisaBaki(tx *sql.Tx, trayID, lokasi int) (int, error) {
	var locBaki int
	if err := tx.QueryRow("SELECT location_id FROM trays WHERE id = ? FOR UPDATE", trayID).Scan(&locBaki); err != nil {
		return 0, salah("baki tidak ditemukan")
	}
	if locBaki != lokasi {
		return 0, salah("baki bukan milik lokasi barang ini")
	}
	var isi int
	if err := tx.QueryRow(sqlIsiBaki, trayID, lokasi).Scan(&isi); err != nil {
		return 0, err
	}
	return KapasitasBaki - isi, nil
}
