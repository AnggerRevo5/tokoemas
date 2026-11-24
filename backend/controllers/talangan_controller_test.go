package controllers

import (
	"fmt"
	"net/http"
	"testing"
)

func TestTalanganCreateValidasi(t *testing.T) {
	r := mesinTalangan(nil, "cabang", 1, 1) // db nil: semua kasus harus ditolak sebelum menyentuh database
	kasus := map[string]string{
		"bp kosong":     `{"bp_id":0,"nama_barang":"Cincin","jenis":"perhiasan","karat":"18K","berat":3,"harga_beli":1000000}`,
		"harga nol":     `{"bp_id":1,"nama_barang":"Cincin","jenis":"perhiasan","karat":"18K","berat":3,"harga_beli":0}`,
		"harga negatif": `{"bp_id":1,"nama_barang":"Cincin","jenis":"perhiasan","karat":"18K","berat":3,"harga_beli":-5}`,
		"berat nol":     `{"bp_id":1,"nama_barang":"Cincin","jenis":"perhiasan","karat":"18K","berat":0,"harga_beli":1000000}`,
		"karat salah":   `{"bp_id":1,"nama_barang":"Cincin","jenis":"perhiasan","karat":"99K","berat":3,"harga_beli":1000000}`,
		"jenis salah":   `{"bp_id":1,"nama_barang":"Cincin","jenis":"susuk","karat":"18K","berat":3,"harga_beli":1000000}`,
		"tanggal salah": `{"bp_id":1,"nama_barang":"Cincin","jenis":"perhiasan","karat":"18K","berat":3,"harga_beli":1000000,"tanggal":"kemarin"}`,
		"nama kosong":   `{"bp_id":1,"nama_barang":"","jenis":"perhiasan","karat":"18K","berat":3,"harga_beli":1000000}`,
	}
	for nama, body := range kasus {
		if w := panggil(r, "POST", "/talangan", body); w.Code != http.StatusBadRequest {
			t.Errorf("%s: kode %d, seharusnya 400 (%s)", nama, w.Code, w.Body.String())
		}
	}
}

func TestTalanganCreateHanyaCabang(t *testing.T) {
	body := `{"bp_id":1,"nama_barang":"Cincin","jenis":"perhiasan","karat":"18K","berat":3,"harga_beli":1000000}`
	for _, role := range []string{"pj", "owner"} {
		r := mesinTalangan(nil, role, 1, 0)
		if w := panggil(r, "POST", "/talangan", body); w.Code != http.StatusForbidden {
			t.Errorf("role %s: kode %d, seharusnya 403", role, w.Code)
		}
	}
}

func TestTalanganCreateTersimpan(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	r := mesinTalangan(db, "cabang", a.User, a.Toko)
	body := fmt.Sprintf(`{"bp_id":%d,"nama_barang":"Cincin uji","jenis":"perhiasan","karat":"18K","berat":3.25,"harga_beli":1500000,"no_surat":"S-1"}`, a.BPID)
	w := panggil(r, "POST", "/talangan", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("kode %d: %s", w.Code, w.Body.String())
	}
	var res struct{ ID int }
	jsonKe(t, w, &res)

	var status, arah string
	var harga int64
	var toko int
	db.QueryRow("SELECT status, arah, CAST(harga_beli AS SIGNED), toko_input_id FROM talangan_barang WHERE id = ?", res.ID).Scan(&status, &arah, &harga, &toko)
	if status != "TALANGAN" || arah != "MASUK" || harga != 1500000 || toko != a.Toko {
		t.Errorf("baris salah: %s %s %d toko %d", status, arah, harga, toko)
	}
	var log int
	db.QueryRow("SELECT COUNT(*) FROM talangan_status_log WHERE talangan_id = ? AND ke = 'TALANGAN'", res.ID).Scan(&log)
	if log != 1 {
		t.Errorf("log status = %d", log)
	}
	var kas int64
	var kasToko int
	db.QueryRow("SELECT CAST(amount AS SIGNED), location_id FROM cash_transactions WHERE id = (SELECT kas_id FROM talangan_barang WHERE id = ?) AND direction='out'", res.ID).Scan(&kas, &kasToko)
	if kas != 1500000 || kasToko != a.Toko {
		t.Errorf("kas toko = %d di lokasi %d", kas, kasToko)
	}
	var item int
	db.QueryRow("SELECT COUNT(*) FROM items WHERE talangan_id = ?", res.ID).Scan(&item)
	if item != 0 {
		t.Error("barang MASUK tidak boleh masuk stok toko")
	}
}

func TestTalanganCreateBPNonaktifDitolak(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	db.Exec("INSERT IGNORE INTO bp_cabang (nama, aktif) VALUES ('BP Uji Mati', FALSE)")
	var mati int
	db.QueryRow("SELECT id FROM bp_cabang WHERE nama='BP Uji Mati'").Scan(&mati)
	r := mesinTalangan(db, "cabang", a.User, a.Toko)
	body := fmt.Sprintf(`{"bp_id":%d,"nama_barang":"X","jenis":"perhiasan","karat":"18K","berat":1,"harga_beli":1000}`, mati)
	if w := panggil(r, "POST", "/talangan", body); w.Code != http.StatusBadRequest {
		t.Errorf("kode %d, seharusnya 400", w.Code)
	}
}

func TestTalanganListCabangHanyaLokasiSendiri(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	r := mesinTalangan(db, "cabang", a.User, a.Toko)
	panggil(r, "POST", "/talangan", fmt.Sprintf(`{"bp_id":%d,"nama_barang":"Daftar uji","jenis":"perhiasan","karat":"18K","berat":1,"harga_beli":1000}`, a.BPID))
	w := panggil(r, "GET", "/talangan?arah=MASUK", "")
	if w.Code != http.StatusOK {
		t.Fatalf("kode %d: %s", w.Code, w.Body.String())
	}
	var rows []struct {
		TokoInputID int    `json:"toko_input_id"`
		Arah        string `json:"arah"`
	}
	jsonKe(t, w, &rows)
	if len(rows) == 0 {
		t.Fatal("daftar kosong")
	}
	for _, x := range rows {
		if x.TokoInputID != a.Toko || x.Arah != "MASUK" {
			t.Errorf("baris di luar lokasi/arah: %+v", x)
		}
	}
	// lokasi lain tidak melihat
	r2 := mesinTalangan(db, "cabang", a.User, a.Toko+9999)
	w2 := panggil(r2, "GET", "/talangan", "")
	var rows2 []map[string]interface{}
	jsonKe(t, w2, &rows2)
	if len(rows2) != 0 {
		t.Errorf("cabang lain melihat %d baris", len(rows2))
	}
}
