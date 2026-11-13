package services

import (
	"database/sql"
	"fmt"
)

// SiapkanSkemaKirim menambah items.diserahkan (sudah diserahkan ke kurir) bila belum ada. Barang lama dianggap sudah diserahkan.
// Isinya sama dengan db/migrations/031_kirim_kurir.sql (bisa dijalankan manual oleh admin database).
func SiapkanSkemaKirim(db *sql.DB) error {
	if kolomAda(db, "items", "diserahkan") {
		return nil
	}
	if _, err := db.Exec("ALTER TABLE items ADD COLUMN diserahkan BOOLEAN NOT NULL DEFAULT TRUE"); err != nil {
		return fmt.Errorf("kolom items.diserahkan belum ada dan gagal dibuat (%v); jalankan db/migrations/031_kirim_kurir.sql dengan user database yang punya hak ALTER", err)
	}
	return nil
}
