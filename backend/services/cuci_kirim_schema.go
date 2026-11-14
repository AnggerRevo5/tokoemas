package services

import (
	"database/sql"
	"fmt"
)

// SiapkanSkemaCuciKirim menambah metadata pengiriman per barang untuk mendukung pengiriman bertahap.
func SiapkanSkemaCuciKirim(db *sql.DB) error {
	kolom := []struct {
		nama, definisi string
	}{
		{"washer_name", "VARCHAR(100) NOT NULL DEFAULT ''"},
		{"sent_date", "DATE NULL"},
	}
	for _, k := range kolom {
		if kolomAda(db, "washing_job_items", k.nama) {
			continue
		}
		if _, err := db.Exec("ALTER TABLE washing_job_items ADD COLUMN " + k.nama + " " + k.definisi); err != nil {
			return fmt.Errorf("kolom washing_job_items.%s belum ada dan gagal dibuat (%v); jalankan backend/migrations/020_cuci_item_pengiriman.sql dengan user database yang punya hak ALTER", k.nama, err)
		}
	}
	return nil
}
