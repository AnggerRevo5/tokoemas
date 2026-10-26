package services

import (
	"database/sql"
	"encoding/json"
	"log"

	"toko-emas/utils"
)

// MetaAudit adalah identitas pelaku sebuah aksi: diambil dari sesi di server, bukan dari isian pengguna.
type MetaAudit struct {
	UserID     int
	Username   string
	LocationID int
	IP         string
}

// CatatAudit menyimpan jejak aksi (nilai sesudah) ke audit_logs.
func CatatAudit(tx *sql.Tx, meta MetaAudit, action, tabel string, recordID int64, data interface{}) error {
	return CatatPerubahan(tx, meta, action, tabel, recordID, nil, data)
}

// CatatPerubahan menyimpan jejak aksi beserta nilai sebelum (lama) dan sesudah (baru).
// Jangan pernah mengirim password atau token ke fungsi ini.
func CatatPerubahan(tx *sql.Tx, meta MetaAudit, action, tabel string, recordID int64, lama, baru interface{}) error {
	var lamaJSON interface{}
	if lama != nil {
		j, _ := json.Marshal(lama)
		lamaJSON = string(j)
	}
	baruJSON, _ := json.Marshal(baru)
	action = utils.BatasTeks(action, 30)

	var err error
	if auditLengkap {
		_, err = tx.Exec(
			`INSERT INTO audit_logs (user_id, username, location_id, ip, action, table_name, record_id, old_value, new_value)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			meta.UserID, utils.StrNull(meta.Username), nolJadiNull(meta.LocationID), utils.StrNull(meta.IP),
			action, tabel, recordID, lamaJSON, string(baruJSON),
		)
	} else {
		_, err = tx.Exec(
			"INSERT INTO audit_logs (user_id, action, table_name, record_id, old_value, new_value) VALUES (?,?,?,?,?,?)",
			meta.UserID, action, tabel, recordID, lamaJSON, string(baruJSON),
		)
	}
	if err != nil {
		log.Printf("audit gagal dicatat (%s %s #%d): %v", action, tabel, recordID, err)
	}
	return err
}

// EventKeamanan adalah satu kejadian keamanan (login berhasil/gagal, logout, sesi dicabut).
type EventKeamanan struct {
	Event      string
	Username   string
	UserID     int
	LocationID int
	IP         string
	UserAgent  string
	Detail     string
}

// CatatKeamanan menyimpan kejadian keamanan ke security_events. Kegagalan hanya dicatat di log server.
func CatatKeamanan(db *sql.DB, e EventKeamanan) {
	_, err := db.Exec(
		`INSERT INTO security_events (event, username, user_id, location_id, ip, user_agent, detail)
		 VALUES (?,?,?,?,?,?,?)`,
		utils.BatasTeks(e.Event, 40), utils.StrNull(utils.BatasTeks(e.Username, 50)), nolJadiNull(e.UserID),
		nolJadiNull(e.LocationID), utils.StrNull(utils.BatasTeks(e.IP, 45)),
		utils.StrNull(utils.BatasTeks(e.UserAgent, 255)), utils.StrNull(utils.BatasTeks(e.Detail, 255)),
	)
	if err != nil {
		log.Printf("event keamanan gagal dicatat (%s): %v", e.Event, err)
	}
}

func nolJadiNull(n int) interface{} {
	if n == 0 {
		return nil
	}
	return n
}
