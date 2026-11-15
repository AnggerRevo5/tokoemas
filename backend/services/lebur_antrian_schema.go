package services

import (
	"database/sql"
	"fmt"
)

const sqlTabelAntrianLebur = `CREATE TABLE IF NOT EXISTS melt_queue_items (
	item_id INT PRIMARY KEY,
	queued_date DATE NOT NULL,
	note VARCHAR(200) NOT NULL DEFAULT '',
	queued_by INT NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	INDEX idx_melt_queue_date (queued_date),
	FOREIGN KEY (item_id) REFERENCES items(id),
	FOREIGN KEY (queued_by) REFERENCES users(id)
)`

func SiapkanSkemaAntrianLebur(db *sql.DB) error {
	if _, err := db.Exec(sqlTabelAntrianLebur); err != nil && !tabelAda(db, "melt_queue_items") {
		return fmt.Errorf("tabel melt_queue_items belum ada dan gagal dibuat (%v); jalankan backend/migrations/021_antrian_lebur.sql dengan user database yang punya hak CREATE", err)
	}
	return nil
}
