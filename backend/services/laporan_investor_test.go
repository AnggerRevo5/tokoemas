package services

import (
	"testing"
	"time"
)

func tglUji(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestHitungRingkasanRentang(t *testing.T) {
	acuan := tglUji(2026, 2, 5)
	gerak := []Gerak{
		{ID: 1, Jenis: GerakSetor, Jumlah: 100_000_000, Tanggal: tglUji(2026, 2, 5)},
		{ID: 2, Jenis: GerakBayarBunga, Jumlah: 1_000_000, Tanggal: tglUji(2026, 3, 6)},
		{ID: 3, Jenis: GerakTarikPokok, Jumlah: 50_000_000, Tanggal: tglUji(2026, 3, 20)},
	}

	// Maret sampai April: bunga 5 Mar (1.000.000) dan 5 Apr (dasar 50 juta = 500.000) jatuh tempo di periode ini
	r := HitungRingkasanRentang(acuan, 100, gerak, tglUji(2026, 3, 1), tglUji(2026, 4, 30))
	want := RingkasanInvestorRentang{
		PokokAwal: 100_000_000, Setor: 0, TarikPokok: 50_000_000, PokokAkhir: 50_000_000,
		BungaTempo: 1_500_000, BungaDibayar: 1_000_000, BungaBelum: 500_000,
	}
	if r != want {
		t.Errorf("Maret-April: dapat %+v, seharusnya %+v", r, want)
	}

	// Februari: hanya setoran awal, belum ada bunga yang jatuh tempo
	r = HitungRingkasanRentang(acuan, 100, gerak, tglUji(2026, 2, 1), tglUji(2026, 2, 28))
	want = RingkasanInvestorRentang{PokokAwal: 0, Setor: 100_000_000, PokokAkhir: 100_000_000}
	if r != want {
		t.Errorf("Februari: dapat %+v, seharusnya %+v", r, want)
	}

	// Mei saja: bunga 5 Mei (dasar 50 juta) jatuh tempo; bunga Maret-April ikut terkumpul tapi tidak dihitung di periode
	r = HitungRingkasanRentang(acuan, 100, gerak, tglUji(2026, 5, 1), tglUji(2026, 5, 31))
	want = RingkasanInvestorRentang{
		PokokAwal: 50_000_000, PokokAkhir: 50_000_000,
		BungaTempo: 500_000, BungaDibayar: 0, BungaBelum: 2_000_000 - 1_000_000,
	}
	if r != want {
		t.Errorf("Mei: dapat %+v, seharusnya %+v", r, want)
	}
}
