package services

import (
	"database/sql"
	"fmt"
)

// SiapkanSkemaHutang menghubungkan setiap pembayaran hutang dengan satu transaksi kas.
func SiapkanSkemaHutang(db *sql.DB) error {
	if kolomAda(db, "debt_payments", "cash_id") {
		return nil
	}
	if _, err := db.Exec("ALTER TABLE debt_payments ADD COLUMN cash_id INT NULL"); err != nil {
		return fmt.Errorf("kolom debt_payments.cash_id belum ada dan gagal dibuat (%v); jalankan migrations/023_hutang_pembayaran_cash.sql dengan user database yang punya hak ALTER", err)
	}
	return nil
}
