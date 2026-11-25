package services

import (
	"math"
	"testing"
)

func TestHitungTukarTambah(t *testing.T) {
	seratus := make([]float64, 101)
	for i := range seratus {
		seratus[i] = 1000
	}
	cases := []struct {
		nama       string
		lama, baru []float64
		metode     string
		biaya      float64
		selisih    float64
		galat      bool
	}{
		{"pelanggan nambah", []float64{5e6}, []float64{6e6}, MetodeTunai, 0, 1e6, false},
		{"banyak barang", []float64{5e6, 3e6}, []float64{4e6, 4e6, 2e6}, MetodeTunai, 0, 2e6, false},
		{"toko bayar", []float64{5e6}, []float64{4e6}, MetodeTunai, 0, -1e6, false},
		{"selisih nol", []float64{5e6}, []float64{5e6}, MetodeTunai, 0, 0, false},
		{"transfer pelanggan bayar", []float64{5e6}, []float64{6e6}, MetodeTransfer, 0, 1e6, false},
		{"transfer toko bayar", []float64{5e6}, []float64{4e6}, MetodeTransfer, 0, -1e6, false},
		{"qris ditolak", []float64{5e6}, []float64{6e6}, MetodeQRIS, 0, 0, true},
		{"potongan di tunai ditolak", []float64{5e6}, []float64{6e6}, MetodeTunai, 5000, 0, true},
		{"barang lama kosong", nil, []float64{6e6}, MetodeTunai, 0, 0, true},
		{"barang baru kosong", []float64{5e6}, nil, MetodeTunai, 0, 0, true},
		{"nilai tukar nol", []float64{0}, []float64{6e6}, MetodeTunai, 0, 0, true},
		{"harga baru negatif", []float64{5e6}, []float64{-1}, MetodeTunai, 0, 0, true},
		{"lebih dari 100 barang", seratus, []float64{6e6}, MetodeTunai, 0, 0, true},
		{"potongan di transfer ditolak", []float64{5e6}, []float64{6e6}, MetodeTransfer, 5000, 0, true},
	}
	for _, c := range cases {
		got, err := HitungTukarTambah(c.lama, c.baru, c.metode, c.biaya)
		if (err != nil) != c.galat {
			t.Errorf("%s: err = %v, galat seharusnya %v", c.nama, err, c.galat)
			continue
		}
		if err == nil && math.Abs(got-c.selisih) > 0.005 {
			t.Errorf("%s: selisih = %v, seharusnya %v", c.nama, got, c.selisih)
		}
		if err != nil && !AdalahKesalahanInput(err) {
			t.Errorf("%s: harus KesalahanInput agar dibalas 400", c.nama)
		}
	}
}
