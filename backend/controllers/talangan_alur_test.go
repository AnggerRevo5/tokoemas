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
	cek("cabang", "/talangan/1/proses", http.StatusForbidden)
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

	// proses lama: belum diproses -> 409; setelah barang masuk cuci -> tersambung
	if w := panggil(pj, "POST", path("proses"), `{"jenis":"cuci"}`); w.Code != http.StatusConflict {
		t.Errorf("proses sebelum cuci: kode %d, seharusnya 409", w.Code)
	}
	var job int
	db.QueryRow("SELECT id FROM washing_jobs ORDER BY id LIMIT 1").Scan(&job)
	if job == 0 {
		t.Skip("database uji tanpa washing_jobs; bagian proses dilewati")
	}
	db.Exec("UPDATE items SET washing_job_id = ?, status = 'cuci' WHERE id = ?", job, stokID)
	if w := panggil(pj, "POST", path("proses"), `{"jenis":"cuci"}`); w.Code != http.StatusOK {
		t.Fatalf("proses: %d %s", w.Code, w.Body.String())
	}
	var pid int
	var pj2, st string
	db.QueryRow("SELECT IFNULL(proses_id,0), IFNULL(proses_jenis,''), status FROM talangan_barang WHERE id = ?", id).Scan(&pid, &pj2, &st)
	if pid != job || pj2 != "cuci" || st != "DI_BO" {
		t.Errorf("proses_id %d jenis %s status %s (status harus tetap DI_BO)", pid, pj2, st)
	}
}

func TestTalanganKeluarCreate(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	pj := mesinTalangan(db, "pj", a.User, 0)
	body := fmt.Sprintf(`{"bp_id":%d,"nama_barang":"Kalung kita","jenis":"perhiasan","karat":"17K","berat":6,"harga_beli":3000000,"no_referensi":"N-9"}`, a.BPID)
	w := panggil(pj, "POST", "/talangan/keluar", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("keluar: %d %s", w.Code, w.Body.String())
	}
	var res struct{ ID int }
	jsonKe(t, w, &res)
	var arah, status string
	var stokID int
	db.QueryRow("SELECT arah, status, IFNULL(stok_id,0) FROM talangan_barang WHERE id = ?", res.ID).Scan(&arah, &status, &stokID)
	if arah != "KELUAR" || status != "TERCATAT" || stokID == 0 {
		t.Fatalf("baris salah: %s %s stok %d", arah, status, stokID)
	}
	var loc, tgn int
	var st string
	db.QueryRow("SELECT location_id, status, IFNULL(talangan_id,0) FROM items WHERE id = ?", stokID).Scan(&loc, &st, &tgn)
	if loc != a.BO || st != "tersedia" || tgn != 0 {
		t.Errorf("barang kita di BO harus stok biasa: lokasi %d status %s talangan_id %d", loc, st, tgn)
	}
	// KELUAR tidak lewat jalur kirim-bo/terima-bo
	if w := panggil(pj, "POST", fmt.Sprintf("/talangan/%d/kirim-bo", res.ID), `{}`); w.Code != http.StatusConflict {
		t.Errorf("kirim-bo pada KELUAR: kode %d, seharusnya 409", w.Code)
	}
}
