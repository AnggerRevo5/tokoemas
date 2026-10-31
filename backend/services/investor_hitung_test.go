package services

import (
	"testing"
	"time"
)

func tgl(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

const juta = int64(1_000_000)

func bungaPada(baris []BarisBunga, jatuhTempo string) int64 {
	for _, b := range baris {
		if b.JatuhTempo.Equal(tgl(jatuhTempo)) {
			return b.Bunga
		}
	}
	return -1
}

func TestBungaPertamaSetelahSatuSiklusPenuh(t *testing.T) {
	g := []Gerak{{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-02-05")}}
	if b := HitungBunga(tgl("2026-02-05"), 100, g, tgl("2026-03-04")); len(b) != 0 {
		t.Fatalf("sebelum 5 Mar belum ada bunga, dapat %d baris", len(b))
	}
	b := HitungBunga(tgl("2026-02-05"), 100, g, tgl("2026-03-05"))
	if len(b) != 1 || b[0].Bunga != 1*juta {
		t.Fatalf("5 Mar harus 1 juta, dapat %+v", b)
	}
}

// Top-up di tengah siklus baru ikut bunga di siklus berikutnya.
func TestTopUpMulaiSiklusBerikutnya(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-02-05")},
		{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-03-10")},
	}
	b := HitungBunga(tgl("2026-02-05"), 100, g, tgl("2026-05-05"))
	for tempo, want := range map[string]int64{"2026-03-05": 1 * juta, "2026-04-05": 1 * juta, "2026-05-05": 2 * juta} {
		if got := bungaPada(b, tempo); got != want {
			t.Errorf("bunga %s = %d, mau %d", tempo, got, want)
		}
	}
}

// Top-up tepat di tanggal acuan dianggap awal siklus baru.
func TestTopUpDiTanggalAcuanIkutSiklusBaru(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-02-05")},
		{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-03-05")},
	}
	b := HitungBunga(tgl("2026-02-05"), 100, g, tgl("2026-04-05"))
	if got := bungaPada(b, "2026-03-05"); got != 1*juta {
		t.Errorf("5 Mar harus 1 juta (top-up belum ikut), dapat %d", got)
	}
	if got := bungaPada(b, "2026-04-05"); got != 2*juta {
		t.Errorf("5 Apr harus 2 juta, dapat %d", got)
	}
}

// Contoh pemilik: pokok 1 miliar, tarik 100 juta sebelum jatuh tempo, bunga jadi 9 juta.
func TestTarikPokokSebelumJatuhTempoMengurangiBunga(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakSetor, Jumlah: 1000 * juta, Tanggal: tgl("2026-01-01")},
		{Jenis: GerakTarikPokok, Jumlah: 100 * juta, Tanggal: tgl("2026-01-20")},
	}
	b := HitungBunga(tgl("2026-01-01"), 100, g, tgl("2026-02-01"))
	if got := bungaPada(b, "2026-02-01"); got != 9*juta {
		t.Fatalf("bunga 1 Feb harus 9 juta, dapat %d", got)
	}
}

// Contoh pemilik: tarik 50 juta dari 100 juta, bunga jadi 500 ribu. Penarikan tepat di tanggal jatuh tempo ikut mengurangi.
func TestTarikSetengahPokokBungaSetengah(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-02-05")},
		{Jenis: GerakTarikPokok, Jumlah: 50 * juta, Tanggal: tgl("2026-03-05")},
	}
	b := HitungBunga(tgl("2026-02-05"), 100, g, tgl("2026-03-05"))
	if got := bungaPada(b, "2026-03-05"); got != 500_000 {
		t.Fatalf("bunga harus 500 ribu, dapat %d", got)
	}
}

func TestBungaTunggalTanpaBungaAtasBunga(t *testing.T) {
	g := []Gerak{{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-01-10")}}
	b := HitungBunga(tgl("2026-01-10"), 100, g, tgl("2026-12-10"))
	if len(b) != 11 || TotalBunga(b) != 11*juta {
		t.Fatalf("11 siklus x 1 juta = 11 juta, dapat %d siklus total %d", len(b), TotalBunga(b))
	}
}

func TestTanggalAkhirBulanDigeserTapiAcuanAsliTetap(t *testing.T) {
	acuan := tgl("2026-01-31")
	for k, want := range map[int]string{1: "2026-02-28", 2: "2026-03-31", 3: "2026-04-30", 13: "2027-02-28"} {
		if got := JatuhTempoKe(acuan, k); !got.Equal(tgl(want)) {
			t.Errorf("siklus %d = %s, mau %s", k, got.Format("2006-01-02"), want)
		}
	}
	if got := JatuhTempoKe(tgl("2024-01-30"), 1); !got.Equal(tgl("2024-02-29")) {
		t.Errorf("tahun kabisat harus 29 Feb, dapat %s", got.Format("2006-01-02"))
	}
	if got := JatuhTempoKe(tgl("2026-11-15"), 2); !got.Equal(tgl("2027-01-15")) {
		t.Errorf("pergantian tahun salah: %s", got.Format("2006-01-02"))
	}
}

func TestPembulatanKeRupiahTerdekat(t *testing.T) {
	if got := BungaSiklus(1_000_050, 100); got != 10_001 { // 10000,5 dibulatkan ke atas
		t.Errorf("dapat %d", got)
	}
	if got := BungaSiklus(1_000_049, 100); got != 10_000 {
		t.Errorf("dapat %d", got)
	}
}

func TestPenarikanMelebihiTopUpTidakMembuatDasarMinus(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-02-05")},
		{Jenis: GerakSetor, Jumlah: 50 * juta, Tanggal: tgl("2026-02-10")},
		{Jenis: GerakTarikPokok, Jumlah: 120 * juta, Tanggal: tgl("2026-02-20")},
	}
	b := HitungBunga(tgl("2026-02-05"), 100, g, tgl("2026-04-05"))
	if got := bungaPada(b, "2026-03-05"); got != 0 {
		t.Errorf("dasar tidak boleh minus, bunga harus 0, dapat %d", got)
	}
	if got := bungaPada(b, "2026-04-05"); got != 30*juta/100 {
		t.Errorf("siklus berikutnya dari sisa pokok 30 juta, dapat %d", got)
	}
}

func TestSiklusBerikut(t *testing.T) {
	g := []Gerak{{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-02-05")}}
	s := SiklusBerikut(tgl("2026-02-05"), 100, g, tgl("2026-03-20"))
	if !s.JatuhTempo.Equal(tgl("2026-04-05")) || s.Bunga != 1*juta {
		t.Fatalf("dapat %+v", s)
	}
}

func TestValidasiPokokTidakBolehMinus(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakSetor, Jumlah: 10 * juta, Tanggal: tgl("2026-02-05")},
		{Jenis: GerakTarikPokok, Jumlah: 11 * juta, Tanggal: tgl("2026-02-06")},
	}
	if err := ValidasiGerak(tgl("2026-02-05"), 100, g, tgl("2026-02-10")); err == nil {
		t.Fatal("penarikan melebihi pokok harus ditolak")
	}
}

func TestValidasiSetorDanTarikDiTanggalSama(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakTarikPokok, Jumlah: 10 * juta, Tanggal: tgl("2026-02-06")},
		{Jenis: GerakSetor, Jumlah: 10 * juta, Tanggal: tgl("2026-02-06")},
	}
	if err := ValidasiGerak(tgl("2026-02-06"), 100, g, tgl("2026-02-10")); err != nil {
		t.Fatalf("setor dan tarik di hari yang sama harus lolos: %v", err)
	}
}

func TestValidasiBungaDibayarTidakBolehMelebihiTerkumpul(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-02-05")},
		{Jenis: GerakBayarBunga, Jumlah: 1 * juta, Tanggal: tgl("2026-03-06")},
	}
	if err := ValidasiGerak(tgl("2026-02-05"), 100, g, tgl("2026-03-10")); err != nil {
		t.Fatalf("membayar tepat 1 juta harus lolos: %v", err)
	}
	g[1].Jumlah = 1*juta + 1
	if err := ValidasiGerak(tgl("2026-02-05"), 100, g, tgl("2026-03-10")); err == nil {
		t.Fatal("membayar lebih dari bunga terkumpul harus ditolak")
	}
}

func TestPokokDanDibayar(t *testing.T) {
	g := []Gerak{
		{Jenis: GerakSetor, Jumlah: 100 * juta, Tanggal: tgl("2026-02-05")},
		{Jenis: GerakTarikPokok, Jumlah: 30 * juta, Tanggal: tgl("2026-02-20")},
		{Jenis: GerakBayarBunga, Jumlah: 5, Tanggal: tgl("2026-03-06")},
	}
	if p := Pokok(g, tgl("2026-02-19")); p != 100*juta {
		t.Errorf("pokok 19 Feb = %d", p)
	}
	if p := Pokok(g, tgl("2026-03-01")); p != 70*juta {
		t.Errorf("pokok 1 Mar = %d", p)
	}
	if Dibayar(g) != 5 {
		t.Errorf("dibayar = %d", Dibayar(g))
	}
}
