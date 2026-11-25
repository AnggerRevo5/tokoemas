package services

import "fmt"

// MaksBarangTukarTambah adalah batas barang per sisi (lama atau baru) dalam satu tukar tambah.
const MaksBarangTukarTambah = 100

// HitungTukarTambah memeriksa satu tukar tambah dan mengembalikan selisih = total harga barang baru - total nilai tukar
// barang lama. Positif: pelanggan membayar (kas toko masuk). Negatif: toko membayar (kas toko keluar).
// Metode hanya tunai atau transfer; keduanya masuk atau keluar dari kas toko. Tidak ada potongan.
func HitungTukarTambah(nilaiLama, hargaBaru []float64, metode string, biaya float64) (float64, error) {
	if len(nilaiLama) == 0 || len(hargaBaru) == 0 {
		return 0, salah("tukar tambah butuh minimal satu barang lama dan satu barang baru")
	}
	if len(nilaiLama) > MaksBarangTukarTambah || len(hargaBaru) > MaksBarangTukarTambah {
		return 0, salah(fmt.Sprintf("maksimal %d barang di tiap sisi", MaksBarangTukarTambah))
	}
	if metode != MetodeTunai && metode != MetodeTransfer {
		return 0, salah("tukar tambah hanya bisa tunai atau transfer")
	}
	var lama, baru float64
	for _, n := range nilaiLama {
		if n <= 0 {
			return 0, salah("nilai tukar barang lama harus lebih dari 0")
		}
		lama += n
	}
	for _, h := range hargaBaru {
		if h < 0 {
			return 0, salah("harga barang baru tidak boleh negatif")
		}
		baru += h
	}
	selisih := baru - lama
	if err := ValidasiBiaya(metode, biaya, selisih); err != nil {
		return 0, err
	}
	return selisih, nil
}
