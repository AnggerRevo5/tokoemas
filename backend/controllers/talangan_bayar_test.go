package controllers

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
)

func pathTukar(id int, aksi string) string { return fmt.Sprintf("/talangan/tukar/%d/%s", id, aksi) }

// tukarBPBayar: saldo +300.000 (BP bayar). tukarKitaBayar: saldo -300.000.
func tukarBPBayar(t *testing.T, a akunUji) (tukar int, barang []int) {
	t.Helper()
	m, _ := masukDiBO(t, a, 1600000)
	k := keluarUji(t, a, 1300000)
	h, kode := buatTukar(t, a, []int{m}, []int{k})
	if h == nil || h.Saldo != 300000 {
		t.Fatalf("tukar BP bayar gagal: kode %d %+v", kode, h)
	}
	return h.ID, []int{m, k}
}

func tukarKitaBayar(t *testing.T, a akunUji) (tukar int, barang []int) {
	t.Helper()
	m, _ := masukDiBO(t, a, 1000000)
	k := keluarUji(t, a, 1300000)
	h, kode := buatTukar(t, a, []int{m}, []int{k})
	if h == nil || h.Saldo != -300000 {
		t.Fatalf("tukar kita bayar gagal: kode %d %+v", kode, h)
	}
	return h.ID, []int{m, k}
}

type kasTukar struct {
	Jumlah     int
	Arah       string
	Nominal    int64
	Lokasi     int
	Metode     string
	StatusUang string
}

func bacaKasTukar(t *testing.T, tukar int) kasTukar {
	t.Helper()
	db := dbUji(t)
	var k kasTukar
	pola := fmt.Sprintf("Tukar talangan #%d %%", tukar)
	db.QueryRow("SELECT COUNT(*) FROM cash_transactions WHERE note LIKE ? AND is_void = FALSE", pola).Scan(&k.Jumlah)
	db.QueryRow("SELECT direction, CAST(amount AS SIGNED), location_id, metode FROM cash_transactions WHERE note LIKE ? AND is_void = FALSE LIMIT 1", pola).
		Scan(&k.Arah, &k.Nominal, &k.Lokasi, &k.Metode)
	db.QueryRow("SELECT status_uang FROM talangan_tukar WHERE id = ?", tukar).Scan(&k.StatusUang)
	return k
}

func TestBayarHanyaPJDanValidasi(t *testing.T) {
	for _, role := range []string{"cabang", "owner"} {
		r := mesinTalangan(nil, role, 1, 1)
		for _, aksi := range []string{"terima-pj", "setor", "transfer"} {
			if w := panggil(r, "POST", pathTukar(1, aksi), `{"nominal":1}`); w.Code != http.StatusForbidden {
				t.Errorf("%s %s: kode %d, seharusnya 403", role, aksi, w.Code)
			}
		}
	}
	pj := mesinTalangan(nil, "pj", 1, 0)
	if w := panggil(pj, "POST", pathTukar(1, "koreksi-nominal"), `{"nominal":1,"alasan":"x"}`); w.Code != http.StatusForbidden {
		t.Errorf("pj koreksi-nominal: kode %d, seharusnya 403", w.Code)
	}
	for nama, b := range map[string]string{"nominal nol": `{"nominal":0}`, "nominal negatif": `{"nominal":-5}`, "kosong": `{}`} {
		if w := panggil(pj, "POST", pathTukar(1, "terima-pj"), b); w.Code != http.StatusBadRequest {
			t.Errorf("%s: kode %d, seharusnya 400", nama, w.Code)
		}
	}
	owner := mesinTalangan(nil, "owner", 1, 0)
	if w := panggil(owner, "POST", pathTukar(1, "koreksi-nominal"), `{"nominal":100}`); w.Code != http.StatusBadRequest {
		t.Errorf("koreksi tanpa alasan: kode %d, seharusnya 400", w.Code)
	}
}

func TestBayarCaraSatuViaPJLaluSetor(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	tukar, barang := tukarBPBayar(t, a)
	pj := mesinTalangan(db, "pj", a.User, 0)

	// nominal beda satu rupiah ditolak
	for _, n := range []int{299999, 300001} {
		if w := panggil(pj, "POST", pathTukar(tukar, "terima-pj"), fmt.Sprintf(`{"nominal":%d}`, n)); w.Code != http.StatusBadRequest {
			t.Errorf("nominal %d: kode %d, seharusnya 400", n, w.Code)
		}
	}
	// setor sebelum terima ditolak
	if w := panggil(pj, "POST", pathTukar(tukar, "setor"), `{}`); w.Code != http.StatusConflict {
		t.Errorf("setor sebelum terima: kode %d, seharusnya 409", w.Code)
	}
	if w := panggil(pj, "POST", pathTukar(tukar, "terima-pj"), `{"nominal":300000}`); w.Code != http.StatusOK {
		t.Fatalf("terima-pj: %d %s", w.Code, w.Body.String())
	}
	k := bacaKasTukar(t, tukar)
	if k.StatusUang != "DI_TANGAN_PJ" || k.Jumlah != 0 {
		t.Fatalf("setelah terima-pj kas BO belum boleh berubah: %+v", k)
	}
	for _, id := range barang {
		if s := statusTalangan(t, id); s != "DIHITUNG" {
			t.Errorf("barang %d = %s, harus tetap DIHITUNG sebelum setor", id, s)
		}
	}
	if w := panggil(pj, "POST", pathTukar(tukar, "setor"), `{}`); w.Code != http.StatusOK {
		t.Fatalf("setor: %d %s", w.Code, w.Body.String())
	}
	k = bacaKasTukar(t, tukar)
	if k.StatusUang != "MASUK_KAS_BO" || k.Jumlah != 1 || k.Arah != "in" || k.Nominal != 300000 || k.Lokasi != a.BO {
		t.Errorf("kas BO salah: %+v", k)
	}
	for _, id := range barang {
		if s := statusTalangan(t, id); s != "LUNAS" {
			t.Errorf("barang %d = %s, seharusnya LUNAS", id, s)
		}
	}
	// setor ulang ditolak
	if w := panggil(pj, "POST", pathTukar(tukar, "setor"), `{}`); w.Code != http.StatusConflict {
		t.Errorf("setor dua kali: kode %d, seharusnya 409", w.Code)
	}
}

func TestBayarCaraDuaTransferLangsung(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	tukar, barang := tukarBPBayar(t, a)
	pj := mesinTalangan(db, "pj", a.User, 0)
	if w := panggil(pj, "POST", pathTukar(tukar, "transfer"), `{"nominal":299999,"no_referensi":"TRX1"}`); w.Code != http.StatusBadRequest {
		t.Errorf("nominal beda: kode %d, seharusnya 400", w.Code)
	}
	if w := panggil(pj, "POST", pathTukar(tukar, "transfer"), `{"nominal":300000}`); w.Code != http.StatusOK {
		t.Fatalf("transfer: %d %s", w.Code, w.Body.String())
	}
	k := bacaKasTukar(t, tukar)
	if k.StatusUang != "MASUK_KAS_BO" || k.Jumlah != 1 || k.Arah != "in" || k.Nominal != 300000 || k.Metode != "transfer" {
		t.Errorf("kas transfer salah: %+v", k)
	}
	for _, id := range barang {
		if s := statusTalangan(t, id); s != "LUNAS" {
			t.Errorf("barang %d = %s", id, s)
		}
	}
	// tanpa nomor referensi ditandai agar owner bisa cek
	var d struct {
		TanpaBukti bool `json:"tanpa_bukti"`
	}
	owner := mesinTalangan(db, "owner", a.User, 0)
	jsonKe(t, panggil(owner, "GET", fmt.Sprintf("/talangan/tukar/%d", tukar), ""), &d)
	if !d.TanpaBukti {
		t.Error("transfer tanpa no_referensi harus ditandai tanpa_bukti")
	}
	tukar2, _ := tukarBPBayar(t, a)
	panggil(pj, "POST", pathTukar(tukar2, "transfer"), `{"nominal":300000,"no_referensi":"TRX-77"}`)
	jsonKe(t, panggil(owner, "GET", fmt.Sprintf("/talangan/tukar/%d", tukar2), ""), &d)
	if d.TanpaBukti {
		t.Error("transfer dengan no_referensi tidak boleh ditandai tanpa bukti")
	}
}

func TestBayarKitaBayarKasBOBerkurang(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	tukar, barang := tukarKitaBayar(t, a)
	pj := mesinTalangan(db, "pj", a.User, 0)
	if w := panggil(pj, "POST", pathTukar(tukar, "transfer"), `{"nominal":300000,"no_referensi":"OUT-1"}`); w.Code != http.StatusOK {
		t.Fatalf("transfer: %d %s", w.Code, w.Body.String())
	}
	k := bacaKasTukar(t, tukar)
	if k.Arah != "out" || k.Nominal != 300000 || k.Lokasi != a.BO || k.Jumlah != 1 {
		t.Errorf("kita bayar: kas BO harus berkurang 300000: %+v", k)
	}
	for _, id := range barang {
		if s := statusTalangan(t, id); s != "LUNAS" {
			t.Errorf("barang %d = %s", id, s)
		}
	}
}

func TestBayarImpasTidakButuhPembayaran(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	m, _ := masukDiBO(t, a, 900000)
	k := keluarUji(t, a, 900000)
	h, _ := buatTukar(t, a, []int{m}, []int{k})
	pj := mesinTalangan(db, "pj", a.User, 0)
	if w := panggil(pj, "POST", pathTukar(h.ID, "terima-pj"), `{"nominal":1}`); w.Code != http.StatusConflict {
		t.Errorf("impas terima-pj: kode %d, seharusnya 409", w.Code)
	}
}

// klik ganda / permintaan serentak: hanya satu baris kas yang boleh terbentuk
func TestBayarKlikGandaHanyaSatuBarisKas(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	pj := mesinTalangan(db, "pj", a.User, 0)

	serentak := func(path, body string) (sukses int) {
		var wg sync.WaitGroup
		hasil := make(chan int, 8)
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				hasil <- panggil(pj, "POST", path, body).Code
			}()
		}
		wg.Wait()
		close(hasil)
		for k := range hasil {
			if k == http.StatusOK {
				sukses++
			}
		}
		return
	}

	// transfer
	t1, _ := tukarBPBayar(t, a)
	if n := serentak(pathTukar(t1, "transfer"), `{"nominal":300000,"no_referensi":"X"}`); n != 1 {
		t.Errorf("transfer serentak: %d berhasil, seharusnya 1", n)
	}
	if k := bacaKasTukar(t, t1); k.Jumlah != 1 {
		t.Errorf("transfer serentak: %d baris kas, seharusnya 1", k.Jumlah)
	}
	// terima-pj lalu setor
	t2, _ := tukarBPBayar(t, a)
	if n := serentak(pathTukar(t2, "terima-pj"), `{"nominal":300000}`); n != 1 {
		t.Errorf("terima-pj serentak: %d berhasil, seharusnya 1", n)
	}
	if n := serentak(pathTukar(t2, "setor"), `{}`); n != 1 {
		t.Errorf("setor serentak: %d berhasil, seharusnya 1", n)
	}
	if k := bacaKasTukar(t, t2); k.Jumlah != 1 {
		t.Errorf("setor serentak: %d baris kas, seharusnya 1", k.Jumlah)
	}
	// koreksi nominal oleh owner serentak: satu catatan koreksi tersimpan konsisten
	t3, _ := tukarBPBayar(t, a)
	owner := mesinTalangan(db, "owner", a.User, 0)
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			panggil(owner, "POST", pathTukar(t3, "koreksi-nominal"), `{"nominal":250000,"alasan":"disepakati"}`)
		}()
	}
	wg.Wait()
	var nominal int64
	db.QueryRow("SELECT CAST(IFNULL(nominal_bayar,0) AS SIGNED) FROM talangan_tukar WHERE id = ?", t3).Scan(&nominal)
	if nominal != 250000 {
		t.Errorf("nominal koreksi = %d", nominal)
	}
}

func TestKoreksiNominalOwner(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	tukar, _ := tukarBPBayar(t, a)
	pj := mesinTalangan(db, "pj", a.User, 0)
	owner := mesinTalangan(db, "owner", a.User, 0)
	if w := panggil(owner, "POST", pathTukar(tukar, "koreksi-nominal"), `{"nominal":250000,"alasan":"BP minta potongan"}`); w.Code != http.StatusOK {
		t.Fatalf("koreksi: %d %s", w.Code, w.Body.String())
	}
	// sekarang nominal harus sama dengan hasil koreksi, bukan saldo
	if w := panggil(pj, "POST", pathTukar(tukar, "transfer"), `{"nominal":300000}`); w.Code != http.StatusBadRequest {
		t.Errorf("nominal lama sesudah koreksi: kode %d, seharusnya 400", w.Code)
	}
	if w := panggil(pj, "POST", pathTukar(tukar, "transfer"), `{"nominal":250000,"no_referensi":"K1"}`); w.Code != http.StatusOK {
		t.Fatalf("transfer nominal koreksi: %d %s", w.Code, w.Body.String())
	}
	if k := bacaKasTukar(t, tukar); k.Nominal != 250000 || k.Jumlah != 1 {
		t.Errorf("kas harus sesuai nominal koreksi: %+v", k)
	}
	var alasan string
	var oleh int
	db.QueryRow("SELECT IFNULL(koreksi_alasan,''), IFNULL(dikonfirmasi_oleh,0) FROM talangan_tukar WHERE id = ?", tukar).Scan(&alasan, &oleh)
	if alasan != "BP minta potongan" || oleh != a.User {
		t.Errorf("alasan %q oleh %d", alasan, oleh)
	}
	// setelah uang masuk kas, koreksi ditolak
	if w := panggil(owner, "POST", pathTukar(tukar, "koreksi-nominal"), `{"nominal":1,"alasan":"x"}`); w.Code != http.StatusConflict {
		t.Errorf("koreksi sesudah lunas: kode %d, seharusnya 409", w.Code)
	}
}
