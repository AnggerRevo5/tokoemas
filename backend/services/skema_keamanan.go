package services

import (
	"database/sql"
	"fmt"
	"log"
)

// Tabel dan kolom tambahan untuk keamanan. Semuanya aditif: tidak ada data lama yang diubah atau dihapus.
// Isinya sama dengan backend/migrations/019_keamanan.sql (bisa dijalankan manual oleh admin database).
var tabelKeamanan = []struct{ nama, sql string }{
	{"user_sessions", `CREATE TABLE IF NOT EXISTS user_sessions (
		id CHAR(32) NOT NULL PRIMARY KEY,
		user_id INT NOT NULL,
		created_at DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		expires_at DATETIME NOT NULL,
		revoked_at DATETIME NULL,
		ip VARCHAR(45) NULL,
		user_agent VARCHAR(255) NULL,
		INDEX idx_sesi_user (user_id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	)`},
	{"security_events", `CREATE TABLE IF NOT EXISTS security_events (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		event VARCHAR(40) NOT NULL,
		username VARCHAR(50) NULL,
		user_id INT NULL,
		location_id INT NULL,
		ip VARCHAR(45) NULL,
		user_agent VARCHAR(255) NULL,
		detail VARCHAR(255) NULL,
		INDEX idx_event_waktu (created_at),
		INDEX idx_event_username (username)
	)`},
}

var kolomAudit = []struct{ nama, definisi string }{
	{"username", "VARCHAR(50) NULL"},
	{"location_id", "INT NULL"},
	{"ip", "VARCHAR(45) NULL"},
}

// auditLengkap bernilai true bila audit_logs sudah punya kolom username, location_id, dan ip.
var auditLengkap bool

// SiapkanSkemaKeamanan membuat tabel sesi dan log keamanan bila belum ada, dan menambah kolom audit.
// Tabel sesi wajib ada (tanpanya tidak ada yang bisa login); kolom audit tambahan bersifat opsional.
func SiapkanSkemaKeamanan(db *sql.DB) error {
	for _, t := range tabelKeamanan {
		if _, err := db.Exec(t.sql); err != nil && !tabelAda(db, t.nama) {
			return fmt.Errorf("tabel %s belum ada dan gagal dibuat (%v); jalankan backend/migrations/019_keamanan.sql dengan user database yang punya hak CREATE", t.nama, err)
		}
	}

	auditLengkap = true
	for _, k := range kolomAudit {
		if kolomAda(db, "audit_logs", k.nama) {
			continue
		}
		// nama kolom dan definisi berasal dari daftar tetap di atas, bukan dari input pengguna
		if _, err := db.Exec("ALTER TABLE audit_logs ADD COLUMN " + k.nama + " " + k.definisi); err != nil {
			log.Printf("peringatan: kolom audit_logs.%s belum bisa ditambahkan (%v); jalankan 019_keamanan.sql", k.nama, err)
			auditLengkap = false
		}
	}
	return nil
}

func tabelAda(db *sql.DB, nama string) bool {
	var n int
	db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", nama).Scan(&n)
	return n > 0
}

func kolomAda(db *sql.DB, tabel, kolom string) bool {
	var n int
	db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?",
		tabel, kolom,
	).Scan(&n)
	return n > 0
}
