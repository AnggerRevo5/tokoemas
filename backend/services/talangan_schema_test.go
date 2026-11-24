package services

import (
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestDDLTalanganBOLengkap(t *testing.T) {
	ada := map[string]string{}
	for _, tb := range tabelTalanganBO {
		ada[tb.nama] = tb.sql
	}
	for _, nama := range []string{"bp_cabang", "talangan_barang", "talangan_status_log", "talangan_koreksi_harga", "talangan_tukar", "talangan_tukar_item"} {
		if _, ok := ada[nama]; !ok {
			t.Errorf("tabel %s belum didefinisikan", nama)
		}
	}
	// index di cabang_id, status, arah (spec)
	barang := ada["talangan_barang"]
	for _, idx := range []string{"idx_tb_cabang", "idx_tb_status", "idx_tb_arah"} {
		if !strings.Contains(barang, idx) {
			t.Errorf("talangan_barang tanpa index %s", idx)
		}
	}
	// satu barang hanya boleh masuk satu penukaran
	if !strings.Contains(ada["talangan_tukar_item"], "UNIQUE KEY uq_tukar_barang (barang_id)") {
		t.Error("talangan_tukar_item harus UNIQUE pada barang_id")
	}
}

// Dijalankan hanya bila TEST_DB_DSN menunjuk ke database kosong/uji (bukan database asli).
func TestSkemaTalanganBOIdempotenDanRename(t *testing.T) {
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN tidak diisi")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 2; i++ {
		if err := SiapkanSkemaTalanganBO(db); err != nil {
			t.Fatalf("panggilan %d: %v", i+1, err)
		}
	}
	for _, nama := range []string{"talangan_lama", "tukar_bp_lama", "bp_cabang", "talangan_barang", "talangan_tukar"} {
		if !tabelAda(db, nama) {
			t.Errorf("tabel %s tidak ada", nama)
		}
	}
	if tabelAda(db, "talangan") || tabelAda(db, "tukar_bp") {
		t.Error("tabel lama talangan/tukar_bp masih ada (harus sudah di-rename)")
	}
	if !kolomAda(db, "items", "talangan_id") {
		t.Error("items.talangan_id tidak ada")
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM bp_cabang WHERE aktif = TRUE").Scan(&n)
	if n < 4 {
		t.Errorf("seed BP kurang: %d", n)
	}
	bp, err := DaftarBPAktif(db)
	if err != nil || len(bp) != n {
		t.Errorf("DaftarBPAktif = %d (err %v), seharusnya %d", len(bp), err, n)
	}
	if !BPValid(db, "BP Iss") || BPValid(db, "BP Fiktif") {
		t.Error("BPValid salah")
	}
}
