package services

import "math"

// Status gadai. Barangnya sendiri berstatus digadai selama gadai aktif atau lunas (menunggu diambil).
const (
	GadaiAktif        = "aktif"
	GadaiLunas        = "lunas"
	GadaiKembali      = "kembali"
	GadaiTidakDitebus = "tidak_ditebus"
)

// Kategori kas gadai. Dana dan pokok adalah pinjaman (tidak dihitung laba); hanya biaya yang masuk Laba Rugi.
const (
	KategoriGadaiDana  = "gadai_dana"
	KategoriGadaiPokok = "gadai_pokok"
	KategoriGadaiBiaya = "gadai_biaya"
)

// BatasHariUbahGadai: akun cabang hanya boleh membatalkan gadai atau pembayarannya dalam rentang ini.
const BatasHariUbahGadai = 7

// SisaGadai adalah kewajiban yang belum dibayar: total cicilan dikurangi yang sudah dibayar dan diskon pelunasan.
// Denda tidak ikut karena itu biaya tambahan di luar kewajiban.
func SisaGadai(total, dibayar, diskon int64) int64 {
	s := total - dibayar - diskon
	if s < 0 {
		return 0
	}
	return s
}

// AlokasiCicilan memecah satu pembayaran cicilan menjadi bagian pokok (mengembalikan dana yang diterima) dan
// bagian biaya (selisih total cicilan dengan dana). Pembagian sebanding; pembayaran yang melunasi sisa
// menggenapkan pokok supaya jumlah pokok seluruh cicilan tepat sama dengan dana.
func AlokasiCicilan(dana, total, pokokDibayar, bayar, sisaSetelah int64) (pokok, biaya int64) {
	sisaPokok := dana - pokokDibayar
	if sisaPokok < 0 {
		sisaPokok = 0
	}
	if sisaSetelah <= 0 || total <= 0 {
		pokok = sisaPokok
	} else {
		pokok = int64(math.Round(float64(bayar) * float64(dana) / float64(total)))
		if pokok > sisaPokok {
			pokok = sisaPokok
		}
	}
	if pokok > bayar {
		pokok = bayar
	}
	return pokok, bayar - pokok
}

// AlokasiPelunasan memecah pembayaran pelunasan. Yang dibayar minimal sebesar sisa pokok; kelebihannya adalah biaya.
func AlokasiPelunasan(dana, pokokDibayar, bayar int64) (pokok, biaya int64, ok bool) {
	sisaPokok := dana - pokokDibayar
	if sisaPokok < 0 {
		sisaPokok = 0
	}
	if bayar < sisaPokok {
		return 0, 0, false
	}
	return sisaPokok, bayar - sisaPokok, true
}
