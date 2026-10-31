package services

import (
	"fmt"
	"sort"
	"time"
)

// Jenis pergerakan uang investor.
const (
	GerakSetor      = "setor"       // pokok bertambah (setor awal atau top-up)
	GerakTarikPokok = "tarik_pokok" // pokok berkurang
	GerakBayarBunga = "bayar_bunga" // bunga yang sudah dicairkan ke investor (tidak mengubah pokok)
)

// Gerak adalah satu catatan uang investor yang belum dibatalkan. Jumlah dalam rupiah bulat.
type Gerak struct {
	ID      int
	Jenis   string
	Jumlah  int64
	Tanggal time.Time // hanya tanggal (UTC, jam 00:00)
}

// BarisBunga adalah bunga satu siklus: dari Mulai (tidak ikut dihitung) sampai JatuhTempo.
type BarisBunga struct {
	Siklus     int
	Mulai      time.Time
	JatuhTempo time.Time
	Dasar      int64 // pokok yang dihitung bunganya
	Bunga      int64
}

const maksSiklus = 1200 // 100 tahun, pengaman agar perulangan selalu berhenti

// TanggalSaja membuang jam dari t dan mengembalikan tanggalnya dalam UTC.
func TanggalSaja(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// HariIniWIB mengembalikan tanggal hari ini menurut waktu Jakarta.
func HariIniWIB() time.Time {
	wib, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return TanggalSaja(time.Now())
	}
	return TanggalSaja(time.Now().In(wib))
}

// JatuhTempoKe mengembalikan tanggal jatuh tempo siklus ke-k (k=0 adalah tanggal acuan itu sendiri).
// Bila bulannya tidak punya tanggal itu (29/30/31), dipakai hari terakhir bulan tersebut,
// tetapi tanggal acuan aslinya tetap dipakai untuk bulan-bulan berikutnya.
func JatuhTempoKe(acuan time.Time, k int) time.Time {
	bulan := int(acuan.Month()) - 1 + k
	tahun := acuan.Year() + bulan/12
	bulan = bulan%12 + 1
	hari := acuan.Day()
	terakhir := time.Date(tahun, time.Month(bulan)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if hari > terakhir {
		hari = terakhir
	}
	return time.Date(tahun, time.Month(bulan), hari, 0, 0, 0, 0, time.UTC)
}

// jumlahSampai menjumlah gerak berjenis tertentu yang tanggalnya <= batas.
func jumlahSampai(gerak []Gerak, jenis string, batas time.Time) int64 {
	var total int64
	for _, g := range gerak {
		if g.Jenis == jenis && !g.Tanggal.After(batas) {
			total += g.Jumlah
		}
	}
	return total
}

// Pokok mengembalikan sisa pokok pada akhir tanggal sampai.
func Pokok(gerak []Gerak, sampai time.Time) int64 {
	return jumlahSampai(gerak, GerakSetor, sampai) - jumlahSampai(gerak, GerakTarikPokok, sampai)
}

// dasarSiklus: pokok awal siklus (setor sampai tanggal mulai, top-up di tengah siklus belum ikut)
// dikurangi semua penarikan pokok sampai jatuh tempo (penarikan langsung mengurangi bunga siklus itu).
func dasarSiklus(gerak []Gerak, mulai, jatuhTempo time.Time) int64 {
	d := jumlahSampai(gerak, GerakSetor, mulai) - jumlahSampai(gerak, GerakTarikPokok, jatuhTempo)
	if d < 0 {
		return 0
	}
	return d
}

// BungaSiklus menghitung bunga sederhana (tanpa bunga atas bunga), dibulatkan ke rupiah terdekat.
// bp = basis poin per bulan (100 = 1%).
func BungaSiklus(dasar, bp int64) int64 {
	return (dasar*bp + 5000) / 10000
}

// HitungBunga mengembalikan bunga semua siklus yang sudah jatuh tempo sampai tanggal hingga.
// Hasilnya diturunkan dari catatan gerak setiap kali dihitung, sehingga koreksi (pembatalan) otomatis ikut.
func HitungBunga(acuan time.Time, bp int64, gerak []Gerak, hingga time.Time) []BarisBunga {
	out := []BarisBunga{}
	for k := 1; k <= maksSiklus; k++ {
		akhir := JatuhTempoKe(acuan, k)
		if akhir.After(hingga) {
			break
		}
		awal := JatuhTempoKe(acuan, k-1)
		dasar := dasarSiklus(gerak, awal, akhir)
		out = append(out, BarisBunga{Siklus: k, Mulai: awal, JatuhTempo: akhir, Dasar: dasar, Bunga: BungaSiklus(dasar, bp)})
	}
	return out
}

// SiklusBerikut mengembalikan siklus pertama yang belum jatuh tempo pada tanggal hingga, dengan perkiraan
// bunganya berdasarkan catatan saat ini (bisa berubah bila ada penarikan sebelum jatuh tempo).
func SiklusBerikut(acuan time.Time, bp int64, gerak []Gerak, hingga time.Time) BarisBunga {
	for k := 1; k <= maksSiklus; k++ {
		akhir := JatuhTempoKe(acuan, k)
		if akhir.After(hingga) {
			awal := JatuhTempoKe(acuan, k-1)
			dasar := dasarSiklus(gerak, awal, akhir)
			return BarisBunga{Siklus: k, Mulai: awal, JatuhTempo: akhir, Dasar: dasar, Bunga: BungaSiklus(dasar, bp)}
		}
	}
	return BarisBunga{}
}

// TotalBunga menjumlah bunga semua baris.
func TotalBunga(baris []BarisBunga) int64 {
	var t int64
	for _, b := range baris {
		t += b.Bunga
	}
	return t
}

// ValidasiGerak memastikan seluruh catatan masih masuk akal:
// pokok tidak pernah minus di tanggal mana pun, dan bunga yang dibayar tidak melebihi bunga yang sudah terkumpul.
func ValidasiGerak(acuan time.Time, bp int64, gerak []Gerak, hingga time.Time) error {
	urut := append([]Gerak(nil), gerak...)
	sort.SliceStable(urut, func(i, j int) bool {
		if !urut[i].Tanggal.Equal(urut[j].Tanggal) {
			return urut[i].Tanggal.Before(urut[j].Tanggal)
		}
		// pada tanggal yang sama setor dihitung lebih dulu
		return urut[i].Jenis == GerakSetor && urut[j].Jenis != GerakSetor
	})
	var pokok, dibayar int64
	for _, g := range urut {
		switch g.Jenis {
		case GerakSetor:
			pokok += g.Jumlah
		case GerakTarikPokok:
			pokok -= g.Jumlah
			if pokok < 0 {
				return salah(fmt.Sprintf("pokok tidak cukup pada %s (kurang %s)", g.Tanggal.Format("2006-01-02"), FmtRp(float64(-pokok))))
			}
		case GerakBayarBunga:
			dibayar += g.Jumlah
		}
	}
	if total := TotalBunga(HitungBunga(acuan, bp, gerak, hingga)); dibayar > total {
		return salah(fmt.Sprintf("bunga yang dibayar (%s) melebihi bunga yang sudah terkumpul (%s)", FmtRp(float64(dibayar)), FmtRp(float64(total))))
	}
	return nil
}

// Dibayar menjumlah semua pembayaran bunga.
func Dibayar(gerak []Gerak) int64 {
	var t int64
	for _, g := range gerak {
		if g.Jenis == GerakBayarBunga {
			t += g.Jumlah
		}
	}
	return t
}
