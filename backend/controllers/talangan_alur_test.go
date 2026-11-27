package controllers

import (
	"fmt"
	"net/http"
	"testing"
)

// masukUji mencatat satu talangan MASUK lewat API dan mengembalikan id-nya.
func masukUji(t *testing.T, a akunUji, harga int64) int {
	t.Helper()
	db := dbUji(t)
	r := mesinTalangan(db, "cabang", a.User, a.Toko)
	body := fmt.Sprintf(`{"bp_id":%d,"nama_barang":"Gelang uji","jenis":"perhiasan","karat":"18K","berat":4.5,"harga_beli":%d}`, a.BPID, harga)
	w := panggil(r, "POST", "/talangan", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("buat masuk: %d %s", w.Code, w.Body.String())
	}
	var res struct{ ID int }
	jsonKe(t, w, &res)
	return res.ID
}

func statusTalangan(t *testing.T, id int) string {
	t.Helper()
	db := dbUji(t)
	var s string
	db.QueryRow("SELECT status FROM talangan_barang WHERE id = ?", id).Scan(&s)
	return s
}

func TestTalanganAlurHanyaRoleYangBerhak(t *testing.T) {
	// db nil: penolakan role harus terjadi sebelum menyentuh database
	cek := func(role, path string, harapan int) {
		r := mesinTalangan(nil, role, 1, 1)
		if w := panggil(r, "POST", path, `{}`); w.Code != harapan {
			t.Errorf("%s %s: kode %d, seharusnya %d", role, path, w.Code, harapan)
		}
	}
	cek("cabang", "/talangan/1/kirim-bo", http.StatusForbidden)
	cek("owner", "/talangan/1/kirim-bo", http.StatusForbidden)
	cek("pj", "/talangan/1/terima-bo", http.StatusForbidden)
	cek("cabang", "/talangan/1/terima-bo", http.StatusForbidden)
	cek("cabang", "/talangan/keluar", http.StatusForbidden)
	cek("owner", "/talangan/keluar", http.StatusForbidden)
}

func TestTalanganAlurLengkapMasukSampaiDiBO(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	id := masukUji(t, a, 2000000)
	pj := mesinTalangan(db, "pj", a.User, 0)
	owner := mesinTalangan(db, "owner", a.User, 0)
	path := func(aksi string) string { return fmt.Sprintf("/talangan/%d/%s", id, aksi) }

	// tidak boleh loncat: TALANGAN langsung ke terima-bo
	if w := panggil(owner, "POST", path("terima-bo"), `{}`); w.Code != http.StatusConflict {
		t.Errorf("loncat status: kode %d, seharusnya 409 (%s)", w.Code, w.Body.String())
	}
	if w := panggil(pj, "POST", path("kirim-bo"), `{}`); w.Code != http.StatusOK {
		t.Fatalf("kirim-bo: %d %s", w.Code, w.Body.String())
	}
	if s := statusTalangan(t, id); s != "KIRIM_BO" {
		t.Fatalf("status = %s", s)
	}
	// kirim dua kali ditolak
	if w := panggil(pj, "POST", path("kirim-bo"), `{}`); w.Code != http.StatusConflict {
		t.Errorf("kirim-bo dua kali: kode %d, seharusnya 409", w.Code)
	}
	var pjOleh int
	db.QueryRow("SELECT IFNULL(dikonfirmasi_pj_oleh,0) FROM talangan_barang WHERE id = ?", id).Scan(&pjOleh)
	if pjOleh != a.User {
		t.Errorf("dikonfirmasi_pj_oleh = %d", pjOleh)
	}

	if w := panggil(owner, "POST", path("terima-bo"), `{}`); w.Code != http.StatusOK {
		t.Fatalf("terima-bo: %d %s", w.Code, w.Body.String())
	}
	if s := statusTalangan(t, id); s != "DI_BO" {
		t.Fatalf("status = %s", s)
	}
	var stokID, boItem, tgn int
	var statusItem, kondisi string
	var harga int64
	db.QueryRow("SELECT IFNULL(stok_id,0) FROM talangan_barang WHERE id = ?", id).Scan(&stokID)
	if err := db.QueryRow("SELECT location_id, status, kondisi, CAST(harga_beli AS SIGNED), IFNULL(talangan_id,0) FROM items WHERE id = ?", stokID).
		Scan(&boItem, &statusItem, &kondisi, &harga, &tgn); err != nil {
		t.Fatalf("barang BO tidak dibuat: %v", err)
	}
	if boItem != a.BO || statusItem != "tersedia" || tgn != id || harga != 2000000 {
		t.Errorf("item BO salah: lokasi %d status %s talangan_id %d harga %d", boItem, statusItem, tgn, harga)
	}
	var mv int
	db.QueryRow("SELECT COUNT(*) FROM stock_movements WHERE item_id = ? AND direction='in' AND location_id = ?", stokID, a.BO).Scan(&mv)
	if mv != 1 {
		t.Errorf("gerak stok masuk BO = %d", mv)
	}
	var log int
	db.QueryRow("SELECT COUNT(*) FROM talangan_status_log WHERE talangan_id = ?", id).Scan(&log)
	if log != 3 { // TALANGAN, KIRIM_BO, DI_BO
		t.Errorf("baris log = %d, seharusnya 3", log)
	}
}

func TestTalanganKeluarCreate(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	pj := mesinTalangan(db, "pj", a.User, 0)
	owner := mesinTalangan(db, "owner", a.User, 0)
	id := keluarTercatatUji(t, a, 3000000)
	var arah, status string
	var stokID int
	db.QueryRow("SELECT arah, status, IFNULL(stok_id,0) FROM talangan_barang WHERE id = ?", id).Scan(&arah, &status, &stokID)
	if arah != "KELUAR" || status != "TERCATAT" || stokID != 0 {
		t.Fatalf("barang kembali belum boleh masuk stok sebelum owner konfirmasi: %s %s stok %d", arah, status, stokID)
	}
	// KELUAR tidak lewat jalur kirim-bo, dan belum bisa ditukar sebelum dikonfirmasi owner
	if w := panggil(pj, "POST", fmt.Sprintf("/talangan/%d/kirim-bo", id), `{}`); w.Code != http.StatusConflict {
		t.Errorf("kirim-bo pada KELUAR: kode %d, seharusnya 409", w.Code)
	}
	tukar := fmt.Sprintf(`{"bp_id":%d,"masuk_ids":[],"keluar_ids":[%d]}`, a.BPID, id)
	if w := panggil(pj, "POST", "/talangan/tukar", tukar); w.Code != http.StatusConflict {
		t.Errorf("tukar sebelum konfirmasi owner: kode %d, seharusnya 409 (%s)", w.Code, w.Body.String())
	}

	if w := panggil(owner, "POST", fmt.Sprintf("/talangan/%d/terima-bo", id), `{}`); w.Code != http.StatusOK {
		t.Fatalf("terima-bo keluar: %d %s", w.Code, w.Body.String())
	}
	db.QueryRow("SELECT status, IFNULL(stok_id,0) FROM talangan_barang WHERE id = ?", id).Scan(&status, &stokID)
	if status != "DI_BO" || stokID == 0 {
		t.Fatalf("setelah konfirmasi: %s stok %d", status, stokID)
	}
	var loc, tgn int
	var st string
	db.QueryRow("SELECT location_id, status, IFNULL(talangan_id,0) FROM items WHERE id = ?", stokID).Scan(&loc, &st, &tgn)
	if loc != a.BO || st != "tersedia" || tgn != 0 {
		t.Errorf("barang kita di BO harus stok biasa: lokasi %d status %s talangan_id %d", loc, st, tgn)
	}
}

func TestTalanganTerimaBOBelumSampai(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	id := masukUji(t, a, 1000000)
	pj := mesinTalangan(db, "pj", a.User, 0)
	owner := mesinTalangan(db, "owner", a.User, 0)
	path := fmt.Sprintf("/talangan/%d/terima-bo", id)
	panggil(pj, "POST", fmt.Sprintf("/talangan/%d/kirim-bo", id), `{}`)

	if w := panggil(owner, "POST", path, `{"result":"belum","catatan":"kurir belum datang"}`); w.Code != http.StatusOK {
		t.Fatalf("belum sampai: %d %s", w.Code, w.Body.String())
	}
	var belum bool
	var note string
	var stokID int
	db.QueryRow("SELECT belum_sampai, IFNULL(belum_sampai_note,''), IFNULL(stok_id,0) FROM talangan_barang WHERE id = ?", id).Scan(&belum, &note, &stokID)
	if !belum || note != "kurir belum datang" || stokID != 0 || statusTalangan(t, id) != "KIRIM_BO" {
		t.Errorf("belum sampai: tanda %v catatan %q stok %d status %s", belum, note, stokID, statusTalangan(t, id))
	}
	if w := panggil(owner, "POST", path, `{}`); w.Code != http.StatusOK {
		t.Fatalf("sampai setelah belum: %d %s", w.Code, w.Body.String())
	}
	db.QueryRow("SELECT belum_sampai FROM talangan_barang WHERE id = ?", id).Scan(&belum)
	if belum || statusTalangan(t, id) != "DI_BO" {
		t.Errorf("tanda belum sampai harus hilang setelah konfirmasi sampai (tanda %v, status %s)", belum, statusTalangan(t, id))
	}
}

func TestTalanganTerimaBOKoreksiBeratKadar(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	id := masukUji(t, a, 1000000) // berat 4.5, 18K
	panggil(mesinTalangan(db, "pj", a.User, 0), "POST", fmt.Sprintf("/talangan/%d/kirim-bo", id), `{}`)
	owner := mesinTalangan(db, "owner", a.User, 0)
	path := fmt.Sprintf("/talangan/%d/terima-bo", id)

	if w := panggil(owner, "POST", path, `{"berat":4.2}`); w.Code != http.StatusBadRequest {
		t.Errorf("koreksi tanpa alasan: kode %d, seharusnya 400", w.Code)
	}
	if s := statusTalangan(t, id); s != "KIRIM_BO" {
		t.Fatalf("koreksi yang ditolak tidak boleh mengubah status: %s", s)
	}
	if w := panggil(owner, "POST", path, `{"berat":4.2,"karat":"17K","alasan":"timbang ulang"}`); w.Code != http.StatusOK {
		t.Fatalf("koreksi: %d %s", w.Code, w.Body.String())
	}
	var berat float64
	var kadar string
	var stokID int
	db.QueryRow("SELECT berat, kadar, stok_id FROM talangan_barang WHERE id = ?", id).Scan(&berat, &kadar, &stokID)
	var beratItem float64
	var karatItem string
	db.QueryRow("SELECT weight_gram, karat FROM items WHERE id = ?", stokID).Scan(&beratItem, &karatItem)
	if berat != 4.2 || kadar != "17K" || beratItem != 4.2 || karatItem != "17K" {
		t.Errorf("hasil koreksi: talangan %v/%s, stok %v/%s", berat, kadar, beratItem, karatItem)
	}
}

// "sambung ke cuci" dihapus: alur cuci dari BO sudah ada, tidak perlu tombol terpisah di Talangan
func TestEndpointProsesTalanganSudahDihapus(t *testing.T) {
	r := mesinTalangan(nil, "owner", 1, 0)
	if w := panggil(r, "POST", "/talangan/1/proses", `{"jenis":"cuci"}`); w.Code != http.StatusNotFound {
		t.Errorf("kode %d, seharusnya 404", w.Code)
	}
}
