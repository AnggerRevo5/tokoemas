package services

import "testing"

func TestNormalMetode(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"", MetodeTunai, true},
		{"tunai", MetodeTunai, true},
		{" QRIS ", MetodeQRIS, true},
		{"Transfer", MetodeTransfer, true},
		{"cek", "", false},
	} {
		got, ok := NormalMetode(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NormalMetode(%q) = %q,%v; seharusnya %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestValidasiBiaya(t *testing.T) {
	cases := []struct {
		metode       string
		biaya, total float64
		galat        bool
	}{
		{MetodeTunai, 0, 1_000_000, false},
		{MetodeQRIS, 0, 1_000_000, false},
		{MetodeQRIS, 7_000, 1_000_000, false},
		{MetodeQRIS, 1_000_000, 1_000_000, false},
		{MetodeQRIS, 1_000_001, 1_000_000, true},
		{MetodeTunai, 5_000, 1_000_000, true},
		{MetodeTransfer, 0, 1_000_000, false},
		{MetodeTransfer, 5_000, 1_000_000, true},
		{MetodeQRIS, -1, 1_000_000, true},
	}
	for _, c := range cases {
		err := ValidasiBiaya(c.metode, c.biaya, c.total)
		if (err != nil) != c.galat {
			t.Errorf("ValidasiBiaya(%s, %v, %v) galat=%v, seharusnya %v", c.metode, c.biaya, c.total, err != nil, c.galat)
		}
	}
}
