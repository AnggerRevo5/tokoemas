package services

import (
	"database/sql"
	"log"
	"strings"
)

// Indeks untuk kolom yang sering dicari: kas dan stok per lokasi dan tanggal, barang per lokasi dan status, dan
// penghubung antar tabel (kas penjualan, kas pembelian, baki, penjualan penadah, riwayat stok per barang).
// Isinya sama dengan db/migrations/027_indeks.sql (bisa dijalankan manual oleh admin database).
var daftarIndeks = []struct {
	tabel, nama string
	kolom       []string
}{
	{"cash_transactions", "idx_kas_lokasi_tgl", []string{"location_id", "transaction_date"}},
	{"cash_transactions", "idx_kas_tgl_kat", []string{"transaction_date", "category"}},
	{"items", "idx_items_lokasi_status", []string{"location_id", "status"}},
	{"items", "idx_items_tray", []string{"tray_id"}},
	{"items", "idx_items_jual_kas", []string{"jual_cash_id"}},
	{"items", "idx_items_jual_biaya", []string{"jual_biaya_cash_id"}},
	{"items", "idx_items_beli_kas", []string{"purchase_cash_id"}},
	{"items", "idx_items_tgl_jual", []string{"tanggal_jual"}},
	{"items", "idx_items_sale", []string{"sale_id"}},
	{"stock_movements", "idx_stok_lokasi_tgl", []string{"location_id", "transaction_date"}},
	{"stock_movements", "idx_stok_item", []string{"item_id"}},
	{"audit_logs", "idx_audit_tabel", []string{"table_name", "record_id"}},
	{"audit_logs", "idx_audit_waktu", []string{"created_at"}},
}

func indeksAda(db *sql.DB, tabel, nama string) bool {
	var n int
	db.QueryRow(
		"SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND INDEX_NAME = ?",
		tabel, nama,
	).Scan(&n)
	return n > 0
}

// SiapkanIndeks membuat indeks yang belum ada. Indeks hanya mempercepat query, jadi kegagalan (misalnya user database
// tanpa hak INDEX) cukup dicatat di log dan tidak menghentikan aplikasi.
func SiapkanIndeks(db *sql.DB) {
	for _, d := range daftarIndeks {
		if !tabelAda(db, d.tabel) || indeksAda(db, d.tabel, d.nama) {
			continue
		}
		lengkap := true
		for _, k := range d.kolom {
			if !kolomAda(db, d.tabel, k) {
				lengkap = false
			}
		}
		if !lengkap {
			continue
		}
		// nama tabel, indeks, dan kolom berasal dari daftar tetap di atas, bukan dari input pengguna
		ddl := "CREATE INDEX " + d.nama + " ON " + d.tabel + " (" + strings.Join(d.kolom, ", ") + ")"
		if _, err := db.Exec(ddl); err != nil {
			log.Printf("peringatan: indeks %s belum bisa dibuat (%v); jalankan db/migrations/027_indeks.sql", d.nama, err)
		}
	}
}
