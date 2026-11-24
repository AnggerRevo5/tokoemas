package services

import "math"

const (
	ArahBPBayar   = "BP_BAYAR"
	ArahKitaBayar = "KITA_BAYAR"
	ArahNol       = "NOL"
)

func jumlah(n []int64) (s int64) {
	for _, v := range n {
		s += v
	}
	return
}

// HitungTukar menghitung penukaran dengan int64 rupiah: saldo = total masuk - total keluar.
// saldo > 0: BP bayar ke kita; saldo < 0: kita bayar ke BP; 0: tidak ada uang pindah.
func HitungTukar(masuk, keluar []int64) (totalMasuk, totalKeluar, saldo int64, arahUang string) {
	totalMasuk, totalKeluar = jumlah(masuk), jumlah(keluar)
	saldo = totalMasuk - totalKeluar
	switch {
	case saldo > 0:
		arahUang = ArahBPBayar
	case saldo < 0:
		arahUang = ArahKitaBayar
	default:
		arahUang = ArahNol
	}
	return
}

// NominalSesuai: nominal pembayaran harus persis |saldo|.
func NominalSesuai(saldo, nominal int64) bool {
	if saldo < 0 {
		saldo = -saldo
	}
	return nominal == saldo
}

// RupiahInt membulatkan ke rupiah terdekat (setengah menjauhi nol).
func RupiahInt(v float64) int64 { return int64(math.Round(v)) }
