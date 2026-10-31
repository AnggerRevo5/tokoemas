package services

import "testing"

func TestSisaGadai(t *testing.T) {
	if got := SisaGadai(3_000_000, 1_000_000, 0); got != 2_000_000 {
		t.Errorf("sisa = %d, mau 2000000", got)
	}
	if got := SisaGadai(3_000_000, 1_000_000, 500_000); got != 1_500_000 {
		t.Errorf("sisa dengan diskon = %d, mau 1500000", got)
	}
	if got := SisaGadai(1_000_000, 2_000_000, 0); got != 0 {
		t.Errorf("sisa tidak boleh negatif, dapat %d", got)
	}
}

func TestAlokasiCicilan(t *testing.T) {
	// dana 2.500.000, 6 x 450.000 = 2.700.000: biaya total 200.000
	const dana, total, cicilan = int64(2_500_000), int64(2_700_000), int64(450_000)
	var pokokDibayar, biayaDibayar, sisa int64 = 0, 0, total
	for i := 1; i <= 6; i++ {
		sisa -= cicilan
		p, b := AlokasiCicilan(dana, total, pokokDibayar, cicilan, sisa)
		if p+b != cicilan {
			t.Fatalf("cicilan %d: pokok %d + biaya %d tidak sama dengan %d", i, p, b, cicilan)
		}
		pokokDibayar += p
		biayaDibayar += b
	}
	if pokokDibayar != dana {
		t.Errorf("total pokok = %d, mau %d", pokokDibayar, dana)
	}
	if biayaDibayar != total-dana {
		t.Errorf("total biaya = %d, mau %d", biayaDibayar, total-dana)
	}
}

func TestAlokasiCicilanTanpaBunga(t *testing.T) {
	// total sama dengan dana: seluruhnya pokok, tidak ada biaya
	p, b := AlokasiCicilan(1_000_000, 1_000_000, 0, 250_000, 750_000)
	if p != 250_000 || b != 0 {
		t.Errorf("pokok %d biaya %d, mau 250000 dan 0", p, b)
	}
}

func TestAlokasiPelunasan(t *testing.T) {
	// dana 2.500.000, sudah bayar pokok 1.000.000; pokok tersisa 1.500.000
	p, b, ok := AlokasiPelunasan(2_500_000, 1_000_000, 1_700_000)
	if !ok || p != 1_500_000 || b != 200_000 {
		t.Errorf("pokok %d biaya %d ok %v, mau 1500000, 200000, true", p, b, ok)
	}
	if _, _, ok := AlokasiPelunasan(2_500_000, 1_000_000, 1_000_000); ok {
		t.Error("bayar di bawah sisa pokok harus ditolak")
	}
}
