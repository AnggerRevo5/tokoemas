package services

import (
	"database/sql"
	"os"
	"testing"
)

// Laporan talangan lama harus tetap jalan lewat tabel arsip (talangan_lama, tukar_bp_lama).
// Dijalankan hanya bila TEST_DB_DSN menunjuk ke database uji.
func TestLaporanTalanganBPMemakaiTabelArsip(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN tidak diisi")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := SiapkanSkemaTalanganBO(db); err != nil {
		t.Fatal(err)
	}
	tbl, err := TabelTalanganBP(db, "2000-01-01", "2100-01-01", "")
	if err != nil {
		t.Fatalf("laporan talangan_bp gagal: %v", err)
	}
	if tbl.Kode != "talangan_bp" {
		t.Errorf("kode laporan = %q", tbl.Kode)
	}
}
