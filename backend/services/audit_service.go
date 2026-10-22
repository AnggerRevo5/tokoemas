package services

import (
	"database/sql"
	"encoding/json"
)

func CatatAudit(tx *sql.Tx, userID int, action, tabel string, recordID int64, data interface{}) error {
	j, _ := json.Marshal(data)
	_, err := tx.Exec(
		"INSERT INTO audit_logs (user_id, action, table_name, record_id, new_value) VALUES (?,?,?,?,?)",
		userID, action, tabel, recordID, string(j),
	)
	return err
}

// ---------- KAS ----------
