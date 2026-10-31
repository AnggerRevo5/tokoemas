package services

import (
	"database/sql"
	"fmt"
)

// SiapkanSkemaBahan menambah items.bahan (emas atau perak) bila belum ada. Barang lama otomatis berbahan emas.
// Isinya sama dengan db/migrations/028_bahan.sql (bisa dijalankan manual oleh admin database).
func SiapkanSkemaBahan(db *sql.DB) error {
	if kolomAda(db, "items", "bahan") {
		return nil
	}
	if _, err := db.Exec("ALTER TABLE items ADD COLUMN bahan ENUM('emas','perak') NOT NULL DEFAULT 'emas'"); err != nil {
		return fmt.Errorf("kolom items.bahan belum ada dan gagal dibuat (%v); jalankan db/migrations/028_bahan.sql dengan user database yang punya hak ALTER", err)
	}
	return nil
}
