package services

import (
	"database/sql"
	"fmt"
)

// Isinya sama dengan db/migrations/021_investor.sql (bisa dijalankan manual oleh admin database).
var tabelInvestor = []struct{ nama, sql string }{
	{"investors", `CREATE TABLE IF NOT EXISTS investors (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		note VARCHAR(255) NULL,
		start_date DATE NOT NULL,
		rate_bp INT NOT NULL DEFAULT 100,
		created_by INT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (created_by) REFERENCES users(id)
	)`},
	{"investor_movements", `CREATE TABLE IF NOT EXISTS investor_movements (
		id INT AUTO_INCREMENT PRIMARY KEY,
		investor_id INT NOT NULL,
		jenis ENUM('setor','tarik_pokok','bayar_bunga') NOT NULL,
		amount BIGINT NOT NULL,
		tx_date DATE NOT NULL,
		note VARCHAR(255) NULL,
		is_void BOOLEAN NOT NULL DEFAULT FALSE,
		void_reason VARCHAR(255) NULL,
		created_by INT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_investor_gerak (investor_id, tx_date),
		FOREIGN KEY (investor_id) REFERENCES investors(id),
		FOREIGN KEY (created_by) REFERENCES users(id)
	)`},
}

// SiapkanSkemaInvestor membuat tabel investor bila belum ada.
func SiapkanSkemaInvestor(db *sql.DB) error {
	for _, t := range tabelInvestor {
		if _, err := db.Exec(t.sql); err != nil && !tabelAda(db, t.nama) {
			return fmt.Errorf("tabel %s belum ada dan gagal dibuat (%v); jalankan db/migrations/021_investor.sql dengan user database yang punya hak CREATE", t.nama, err)
		}
	}
	return nil
}

// penanya dipenuhi oleh *sql.DB maupun *sql.Tx.
type penanya interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// GerakInvestor memuat catatan investor yang belum dibatalkan.
func GerakInvestor(q penanya, investorID int) ([]Gerak, error) {
	rows, err := q.Query(
		"SELECT id, jenis, amount, tx_date FROM investor_movements WHERE investor_id = ? AND is_void = FALSE ORDER BY tx_date, id",
		investorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Gerak{}
	for rows.Next() {
		var g Gerak
		if err := rows.Scan(&g.ID, &g.Jenis, &g.Jumlah, &g.Tanggal); err != nil {
			return nil, err
		}
		g.Tanggal = TanggalSaja(g.Tanggal)
		out = append(out, g)
	}
	return out, rows.Err()
}
