package services

import "testing"

func TestHitungTukar(t *testing.T) {
	kasus := []struct {
		nama          string
		masuk, keluar []int64
		tm, tk, saldo int64
		arah          string
	}{
		{"BP bayar selisih", []int64{1_600_000}, []int64{1_300_000}, 1_600_000, 1_300_000, 300_000, "BP_BAYAR"},
		{"kita bayar selisih", []int64{1_300_000}, []int64{1_600_000}, 1_300_000, 1_600_000, -300_000, "KITA_BAYAR"},
		{"impas", []int64{500_000, 500_000}, []int64{1_000_000}, 1_000_000, 1_000_000, 0, "NOL"},
		{"BP tidak punya barang kita", []int64{700_000, 300_000}, nil, 1_000_000, 0, 1_000_000, "BP_BAYAR"},
		{"angka besar", []int64{9_000_000_000_000}, []int64{1}, 9_000_000_000_000, 1, 8_999_999_999_999, "BP_BAYAR"},
	}
	for _, k := range kasus {
		tm, tk, saldo, arah := HitungTukar(k.masuk, k.keluar)
		if tm != k.tm || tk != k.tk || saldo != k.saldo || arah != k.arah {
			t.Errorf("%s: dapat %d %d %d %s", k.nama, tm, tk, saldo, arah)
		}
	}
}

func TestNominalSesuai(t *testing.T) {
	if !NominalSesuai(300_000, 300_000) || !NominalSesuai(-300_000, 300_000) {
		t.Error("nominal sama dengan |saldo| harus sesuai")
	}
	if NominalSesuai(300_000, 300_001) || NominalSesuai(300_000, 299_999) || NominalSesuai(300_000, -300_000) {
		t.Error("selisih satu rupiah atau nominal negatif harus ditolak")
	}
	if NominalSesuai(0, 0) != true {
		t.Error("saldo 0 tidak butuh pembayaran")
	}
}

func TestRupiahInt(t *testing.T) {
	if RupiahInt(1599999.6) != 1_600_000 || RupiahInt(0.4) != 0 || RupiahInt(-2.5) != -3 {
		t.Error("pembulatan ke rupiah terdekat salah")
	}
}
