package controllers

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

// keluarUji mencatat barang KELUAR (barang kita yang dibeli BP) lewat API; mengembalikan id talangan.
func keluarUji(t *testing.T, a akunUji, harga int64) int {
	t.Helper()
	db := dbUji(t)
	pj := mesinTalangan(db, "pj", a.User, 0)
	body := fmt.Sprintf(`{"bp_id":%d,"nama_barang":"Kalung kita","jenis":"perhiasan","karat":"17K","berat":6,"harga_beli":%d}`, a.BPID, harga)
	w := panggil(pj, "POST", "/talangan/keluar", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("keluar: %d %s", w.Code, w.Body.String())
	}
	var res struct{ ID int }
	jsonKe(t, w, &res)
	return res.ID
}

type hasilTukar struct {
	ID          int    `json:"id"`
	TotalMasuk  int64  `json:"total_masuk"`
	TotalKeluar int64  `json:"total_keluar"`
	Saldo       int64  `json:"saldo"`
	ArahUang    string `json:"arah_uang"`
	StatusUang  string `json:"status_uang"`
}

func buatTukar(t *testing.T, a akunUji, masuk, keluar []int) (*hasilTukar, int) {
	t.Helper()
	db := dbUji(t)
	pj := mesinTalangan(db, "pj", a.User, 0)
	body := fmt.Sprintf(`{"bp_id":%d,"masuk_ids":%s,"keluar_ids":%s}`, a.BPID, daftarJSON(masuk), daftarJSON(keluar))
	w := panggil(pj, "POST", "/talangan/tukar", body)
	if w.Code != http.StatusCreated {
		return nil, w.Code
	}
	var h hasilTukar
	jsonKe(t, w, &h)
	return &h, w.Code
}

func daftarJSON(ids []int) string {
	if len(ids) == 0 {
		return "[]"
	}
	s := "["
	for i, id := range ids {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprint(id)
	}
	return s + "]"
}

func TestTukarCreateHanyaPJ(t *testing.T) {
	for _, role := range []string{"cabang", "owner"} {
		r := mesinTalangan(nil, role, 1, 1)
		if w := panggil(r, "POST", "/talangan/tukar", `{"bp_id":1,"masuk_ids":[1],"keluar_ids":[]}`); w.Code != http.StatusForbidden {
			t.Errorf("%s: kode %d, seharusnya 403", role, w.Code)
		}
	}
	r := mesinTalangan(nil, "pj", 1, 0)
	for nama, b := range map[string]string{
		"keduanya kosong": `{"bp_id":1,"masuk_ids":[],"keluar_ids":[]}`,
		"id ganda":        `{"bp_id":1,"masuk_ids":[3,3],"keluar_ids":[]}`,
		"bp kosong":       `{"bp_id":0,"masuk_ids":[1],"keluar_ids":[]}`,
		"id negatif":      `{"bp_id":1,"masuk_ids":[-1],"keluar_ids":[]}`,
	} {
		if w := panggil(r, "POST", "/talangan/tukar", b); w.Code != http.StatusBadRequest {
			t.Errorf("%s: kode %d, seharusnya 400", nama, w.Code)
		}
	}
}

func TestTukarCreateMenghitungDanMemindahStatus(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	m1, s1 := masukDiBO(t, a, 1000000)
	m2, _ := masukDiBO(t, a, 600000)
	k1 := keluarUji(t, a, 1300000)
	h, kode := buatTukar(t, a, []int{m1, m2}, []int{k1})
	if h == nil {
		t.Fatalf("kode %d", kode)
	}
	if h.TotalMasuk != 1600000 || h.TotalKeluar != 1300000 || h.Saldo != 300000 || h.ArahUang != "BP_BAYAR" || h.StatusUang != "BELUM_BAYAR" {
		t.Errorf("hasil salah: %+v", h)
	}
	for _, id := range []int{m1, m2, k1} {
		if s := statusTalangan(t, id); s != "DIHITUNG" {
			t.Errorf("status %d = %s, seharusnya DIHITUNG", id, s)
		}
	}
	var log int
	db.QueryRow("SELECT COUNT(*) FROM talangan_status_log WHERE talangan_id = ? AND ke IN ('DIKIRIM_KE_BP','DIHITUNG')", m1).Scan(&log)
	if log != 2 {
		t.Errorf("log MASUK harus mencatat DIKIRIM_KE_BP lalu DIHITUNG, dapat %d baris", log)
	}
	var st, bp string
	db.QueryRow("SELECT status, IFNULL(bp_nama,'') FROM items WHERE id = ?", s1).Scan(&st, &bp)
	if st != "di_bp" || bp == "" {
		t.Errorf("barang MASUK harus keluar dari stok BO (di_bp): %s %q", st, bp)
	}
	var out int
	db.QueryRow("SELECT COUNT(*) FROM stock_movements WHERE item_id = ? AND direction='out' AND location_id = ?", s1, a.BO).Scan(&out)
	if out != 1 {
		t.Errorf("gerak stok keluar BO = %d", out)
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM talangan_tukar_item WHERE tukar_id = ?", h.ID).Scan(&n)
	if n != 3 {
		t.Errorf("item penukaran = %d", n)
	}
	// nilai disalin saat penukaran dibuat
	var nilai int64
	db.QueryRow("SELECT CAST(nilai AS SIGNED) FROM talangan_tukar_item WHERE tukar_id = ? AND barang_id = ?", h.ID, m1).Scan(&nilai)
	if nilai != 1000000 {
		t.Errorf("nilai salinan = %d", nilai)
	}
}

func TestTukarCreateBPTanpaBarangKita(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	m1, _ := masukDiBO(t, a, 800000)
	h, kode := buatTukar(t, a, []int{m1}, nil)
	if h == nil {
		t.Fatalf("kode %d", kode)
	}
	if h.TotalKeluar != 0 || h.Saldo != 800000 || h.ArahUang != "BP_BAYAR" {
		t.Errorf("BP bayar penuh: %+v", h)
	}
}

func TestTukarCreateImpasLangsungLunas(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	m1, _ := masukDiBO(t, a, 900000)
	k1 := keluarUji(t, a, 900000)
	h, kode := buatTukar(t, a, []int{m1}, []int{k1})
	if h == nil {
		t.Fatalf("kode %d", kode)
	}
	if h.Saldo != 0 || h.ArahUang != "NOL" || h.StatusUang != "MASUK_KAS_BO" {
		t.Errorf("impas: %+v", h)
	}
	for _, id := range []int{m1, k1} {
		if s := statusTalangan(t, id); s != "LUNAS" {
			t.Errorf("status %d = %s, seharusnya LUNAS", id, s)
		}
	}
	var kas int
	db.QueryRow("SELECT COUNT(*) FROM talangan_tukar WHERE id = ? AND kas_id IS NOT NULL", h.ID).Scan(&kas)
	if kas != 0 {
		t.Error("impas tidak boleh membuat baris kas")
	}
}

func TestTukarCreateAturanBarang(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	// barang masih TALANGAN (belum DI_BO)
	belum := masukUji(t, a, 100000)
	if _, kode := buatTukar(t, a, []int{belum}, nil); kode != http.StatusConflict {
		t.Errorf("barang belum DI_BO: kode %d, seharusnya 409", kode)
	}
	// barang sedang dicuci tidak bisa ditukar
	m, stok := masukDiBO(t, a, 100000)
	db.Exec("UPDATE items SET status = 'cuci' WHERE id = ?", stok)
	if _, kode := buatTukar(t, a, []int{m}, nil); kode != http.StatusConflict {
		t.Errorf("barang sedang diproses: kode %d, seharusnya 409", kode)
	}
	db.Exec("UPDATE items SET status = 'tersedia' WHERE id = ?", stok)
	// cabang BP lain
	var lain int
	db.QueryRow("SELECT id FROM bp_cabang WHERE id <> ? AND aktif = TRUE LIMIT 1", a.BPID).Scan(&lain)
	pj := mesinTalangan(db, "pj", a.User, 0)
	body := fmt.Sprintf(`{"bp_id":%d,"masuk_ids":[%d],"keluar_ids":[]}`, lain, m)
	if w := panggil(pj, "POST", "/talangan/tukar", body); w.Code != http.StatusConflict && w.Code != http.StatusBadRequest {
		t.Errorf("BP berbeda: kode %d, seharusnya 400/409", w.Code)
	}
	// satu barang hanya boleh masuk satu penukaran
	if _, kode := buatTukar(t, a, []int{m}, nil); kode != http.StatusCreated {
		t.Fatalf("penukaran pertama: kode %d", kode)
	}
	if _, kode := buatTukar(t, a, []int{m}, nil); kode != http.StatusConflict {
		t.Errorf("penukaran kedua untuk barang sama: kode %d, seharusnya 409", kode)
	}
}

func TestTukarCreateBalapanHanyaSatuBerhasil(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	m, _ := masukDiBO(t, a, 250000)
	var wg sync.WaitGroup
	hasil := make(chan int, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, kode := buatTukar(t, a, []int{m}, nil)
			hasil <- kode
		}()
	}
	wg.Wait()
	close(hasil)
	ok := 0
	for k := range hasil {
		if k == http.StatusCreated {
			ok++
		}
	}
	if ok != 1 {
		t.Errorf("%d permintaan berhasil, seharusnya tepat 1", ok)
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM talangan_tukar_item WHERE barang_id = ?", m).Scan(&n)
	if n != 1 {
		t.Errorf("barang ada di %d penukaran", n)
	}
}
