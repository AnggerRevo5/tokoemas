package services

import (
	"database/sql"
	"fmt"
)

const sqlTabelAntrianCuci = `CREATE TABLE IF NOT EXISTS washing_queue_items (
	item_id INT PRIMARY KEY,
	queued_date DATE NOT NULL,
	note VARCHAR(200) NOT NULL DEFAULT '',
	queued_by INT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_washing_queue_date (queued_date),
	FOREIGN KEY (item_id) REFERENCES items(id),
	FOREIGN KEY (queued_by) REFERENCES users(id)
)`

func SiapkanSkemaAntrianCuci(db *sql.DB) error {
	if _, err := db.Exec(sqlTabelAntrianCuci); err != nil && !tabelAda(db, "washing_queue_items") {
		return fmt.Errorf("tabel washing_queue_items belum ada dan gagal dibuat (%v); jalankan backend/migrations/022_antrian_cuci.sql dengan user database yang punya hak CREATE", err)
	}
	return nil
}
