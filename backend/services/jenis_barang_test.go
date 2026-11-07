package services

import "testing"

func TestAturanJenis(t *testing.T) {
	kasus := []struct {
		jenis                   string
		tanpaGram, gratis, baki bool
		awalan, nama            string
	}{
		{"susuk", true, false, true, "SSK", "Susuk emas"},
		{"tempat_perhiasan", true, true, false, "TMP", "Tempat perhiasan"},
		{"kotak_perhiasan", true, true, false, "KTK", "Kotak perhiasan"},
		{"perhiasan", false, false, true, "BRG", ""},
		{"logam_mulia", false, false, true, "BRG", ""},
		{"hasil_lebur", false, false, false, "BRG", ""},
		{"", false, false, false, "BRG", ""},
	}
	for _, k := range kasus {
		if got := TanpaGram(k.jenis); got != k.tanpaGram {
			t.Errorf("TanpaGram(%q) = %v, mau %v", k.jenis, got, k.tanpaGram)
		}
		if got := BolehGratis(k.jenis); got != k.gratis {
			t.Errorf("BolehGratis(%q) = %v, mau %v", k.jenis, got, k.gratis)
		}
		if got := BolehBaki(k.jenis); got != k.baki {
			t.Errorf("BolehBaki(%q) = %v, mau %v", k.jenis, got, k.baki)
		}
		if got := AwalanKode(k.jenis); got != k.awalan {
			t.Errorf("AwalanKode(%q) = %q, mau %q", k.jenis, got, k.awalan)
		}
		if got := NamaBawaan(k.jenis); got != k.nama {
			t.Errorf("NamaBawaan(%q) = %q, mau %q", k.jenis, got, k.nama)
		}
	}
}
