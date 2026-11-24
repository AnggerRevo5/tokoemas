package controllers

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

type baris struct {
	BPID       int    `json:"bp_id"`
	BP         string `json:"bp"`
	Aktif      bool   `json:"aktif"`
	DiBOJumlah int    `json:"di_bo_jumlah"`
	DiBONilai  int64  `json:"di_bo_nilai"`
	TukarBelum int    `json:"tukar_belum_lunas"`
	Piutang    int64  `json:"piutang"`
	Hutang     int64  `json:"hutang"`
}

func ambilPiutang(t *testing.T, a akunUji, bp int) baris {
	t.Helper()
	db := dbUji(t)
	w := panggil(mesinTalangan(db, "owner", a.User, 0), "GET", "/talangan/piutang", "")
	if w.Code != http.StatusOK {
		t.Fatalf("piutang: %d %s", w.Code, w.Body.String())
	}
	var rows []baris
	jsonKe(t, w, &rows)
	for _, r := range rows {
		if r.BPID == bp {
			return r
		}
	}
	t.Fatalf("BP %d tidak ada di piutang", bp)
	return baris{}
}

func TestPiutangPerCabangBP(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	// BP khusus uji supaya angkanya bersih
	sa := mesinTalangan(db, "super_admin", a.User, 0)
	nama := fmt.Sprintf("BP Piutang %d", time.Now().UnixNano()%1000000)
	w := panggil(sa, "POST", "/talangan/bp", fmt.Sprintf(`{"nama":%q}`, nama))
	if w.Code != http.StatusCreated {
		t.Fatalf("tambah BP: %d %s", w.Code, w.Body.String())
	}
	var bp struct{ ID int }
	jsonKe(t, w, &bp)
	a.BPID = bp.ID

	// cabang BP baru langsung bisa dipakai tanpa ubah kode
	m1, _ := masukDiBO(t, a, 1600000)
	m2, _ := masukDiBO(t, a, 400000)
	r := ambilPiutang(t, a, bp.ID)
	if r.DiBOJumlah != 2 || r.DiBONilai != 2000000 || r.Piutang != 0 {
		t.Errorf("sebelum penukaran: %+v", r)
	}
	// tukar m1 dengan barang keluar 1.300.000: BP masih berutang 300.000
	k := keluarUji(t, a, 1300000)
	h, _ := buatTukar(t, a, []int{m1}, []int{k})
	r = ambilPiutang(t, a, bp.ID)
	if r.DiBOJumlah != 1 || r.DiBONilai != 400000 || r.Piutang != 300000 || r.TukarBelum != 1 {
		t.Errorf("sesudah penukaran: %+v", r)
	}
	// lunas: piutang hilang
	panggil(mesinTalangan(db, "pj", a.User, 0), "POST", pathTukar(h.ID, "transfer"), `{"nominal":300000,"no_referensi":"P1"}`)
	r = ambilPiutang(t, a, bp.ID)
	if r.Piutang != 0 || r.TukarBelum != 0 {
		t.Errorf("sesudah lunas: %+v", r)
	}
	// kita yang berutang
	k2 := keluarUji(t, a, 900000)
	h2, _ := buatTukar(t, a, []int{m2}, []int{k2})
	if h2 == nil {
		t.Fatal("tukar kedua gagal")
	}
	r = ambilPiutang(t, a, bp.ID)
	if r.Hutang != 500000 || r.Piutang != 0 {
		t.Errorf("kita bayar: %+v", r)
	}
	// BP dinonaktifkan: hilang dari pilihan, tapi saldo tetap terbaca
	if w := panggil(sa, "PUT", fmt.Sprintf("/talangan/bp/%d", bp.ID), `{"aktif":false}`); w.Code != http.StatusOK {
		t.Fatalf("nonaktifkan: %d %s", w.Code, w.Body.String())
	}
	r = ambilPiutang(t, a, bp.ID)
	if r.Aktif || r.Hutang != 500000 {
		t.Errorf("BP nonaktif harus tetap terbaca: %+v", r)
	}
	var pilihan []struct {
		ID int `json:"id"`
	}
	jsonKe(t, panggil(sa, "GET", "/talangan/bp", ""), &pilihan)
	for _, p := range pilihan {
		if p.ID == bp.ID {
			t.Error("BP nonaktif masih muncul di pilihan")
		}
	}
	if w := panggil(mesinTalangan(db, "cabang", a.User, a.Toko), "POST", "/talangan",
		fmt.Sprintf(`{"bp_id":%d,"nama_barang":"X","jenis":"perhiasan","karat":"18K","berat":1,"harga_beli":1000}`, bp.ID)); w.Code != http.StatusBadRequest {
		t.Errorf("input ke BP nonaktif: kode %d, seharusnya 400", w.Code)
	}
}

func TestKelolaBPHanyaSuperAdmin(t *testing.T) {
	for _, role := range []string{"owner", "pj", "cabang"} {
		r := mesinTalangan(nil, role, 1, 1)
		if w := panggil(r, "POST", "/talangan/bp", `{"nama":"BP Baru"}`); w.Code != http.StatusForbidden {
			t.Errorf("%s tambah BP: kode %d, seharusnya 403", role, w.Code)
		}
		if w := panggil(r, "PUT", "/talangan/bp/1", `{"aktif":false}`); w.Code != http.StatusForbidden {
			t.Errorf("%s ubah BP: kode %d, seharusnya 403", role, w.Code)
		}
	}
	sa := mesinTalangan(nil, "super_admin", 1, 0)
	for nama, b := range map[string]string{"nama kosong": `{"nama":"  "}`, "nama panjang": `{"nama":"` + string(make([]byte, 0)) + `xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`} {
		if w := panggil(sa, "POST", "/talangan/bp", b); w.Code != http.StatusBadRequest {
			t.Errorf("%s: kode %d, seharusnya 400", nama, w.Code)
		}
	}
}

func TestPiutangHanyaRoleBerhak(t *testing.T) {
	r := mesinTalangan(nil, "cabang", 1, 1)
	if w := panggil(r, "GET", "/talangan/piutang", ""); w.Code != http.StatusForbidden {
		t.Errorf("cabang: kode %d, seharusnya 403", w.Code)
	}
}
