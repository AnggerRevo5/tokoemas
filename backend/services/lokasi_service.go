package services

import (
	"database/sql"
)

func CekCabang(db *sql.DB, id int) (string, bool) {
	var nama, tipe string
	if err := db.QueryRow("SELECT name, type FROM locations WHERE id = ?", id).Scan(&nama, &tipe); err != nil || tipe != "cabang" {
		return "", false
	}
	return nama, true
}

func IDBackOffice(db *sql.DB) (int, error) {
	var id int
	err := db.QueryRow("SELECT id FROM locations WHERE type = 'back_office' LIMIT 1").Scan(&id)
	return id, err
}
