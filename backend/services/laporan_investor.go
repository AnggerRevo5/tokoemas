package services

import (
	"database/sql"
	"fmt"
	"time"

	"toko-emas/models"
)

var labelGerakLaporan = map[string]string{
	GerakSetor:      "Setor pokok",
	GerakTarikPokok: "Tarik pokok",
	GerakBayarBunga: "Bayar bunga",
}

// jumlahRentang menjumlah gerak berjenis tertentu yang tanggalnya dari <= tanggal <= sampai.
func jumlahRentang(gerak []Gerak, jenis string, dari, sampai time.Time) int64 {
	var total int64
	for _, g := range gerak {
		if g.Jenis == jenis && !g.Tanggal.Before(dari) && !g.Tanggal.After(sampai) {
			total += g.Jumlah
		}
	}
	return total
}

// RingkasanInvestorRentang adalah angka satu investor untuk periode dari sampai sampai (keduanya ikut dihitung).
type RingkasanInvestorRentang struct {
	PokokAwal    int64 // sisa pokok pada akhir hari sebelum dari
	Setor        int64
	TarikPokok   int64
	PokokAkhir   int64
	BungaTempo   int64 // bunga siklus yang jatuh temponya di dalam periode
	BungaDibayar int64 // bunga yang dicairkan di dalam periode
	BungaBelum   int64 // bunga terkumpul sampai akhir periode dikurangi semua bunga yang sudah dibayar sampai saat itu
}

// HitungRingkasanRentang menghitung angka satu investor dari catatannya. Fungsi murni agar mudah diuji.
func HitungRingkasanRentang(acuan time.Time, bp int64, gerak []Gerak, dari, sampai time.Time) RingkasanInvestorRentang {
	var tempo int64
	baris := HitungBunga(acuan, bp, gerak, sampai)
	for _, b := range baris {
		if !b.JatuhTempo.Before(dari) {
			tempo += b.Bunga
		}
	}
	return RingkasanInvestorRentang{
		PokokAwal:    Pokok(gerak, dari.AddDate(0, 0, -1)),
		Setor:        jumlahRentang(gerak, GerakSetor, dari, sampai),
		TarikPokok:   jumlahRentang(gerak, GerakTarikPokok, dari, sampai),
		PokokAkhir:   Pokok(gerak, sampai),
		BungaTempo:   tempo,
		BungaDibayar: jumlahRentang(gerak, GerakBayarBunga, dari, sampai),
		BungaBelum:   TotalBunga(baris) - jumlahSampai(gerak, GerakBayarBunga, sampai),
	}
}

func parseTanggalLaporan(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return t, salah("format tanggal harus YYYY-MM-DD")
	}
	return TanggalSaja(t), nil
}

// ---------- RINGKASAN INVESTOR (rentang tanggal) ----------

func TabelInvestor(db *sql.DB, dari, sampai string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "investor",
		Judul: "Investor (ringkasan periode)",
		Kolom: []string{"Investor", "Setor awal", "Pokok awal (Rp)", "Setor (Rp)", "Tarik pokok (Rp)", "Pokok akhir (Rp)",
			"Bunga jatuh tempo (Rp)", "Bunga dibayar (Rp)", "Bunga belum dibayar (Rp)"},
		Tipe: []string{"teks", "teks", "rupiah", "rupiah", "rupiah", "rupiah", "rupiah", "rupiah", "rupiah"},
	}
	tDari, err := parseTanggalLaporan(dari)
	if err != nil {
		return t, err
	}
	tSampai, err := parseTanggalLaporan(sampai)
	if err != nil {
		return t, err
	}
	rows, err := db.Query("SELECT id, name, start_date, rate_bp FROM investors WHERE start_date <= ? ORDER BY name, id", sampai)
	if err != nil {
		return t, err
	}
	type inv struct {
		id    int
		nama  string
		acuan time.Time
		bp    int64
	}
	daftar := []inv{}
	for rows.Next() {
		var x inv
		if err := rows.Scan(&x.id, &x.nama, &x.acuan, &x.bp); err != nil {
			rows.Close()
			return t, err
		}
		x.acuan = TanggalSaja(x.acuan)
		daftar = append(daftar, x)
	}
	rows.Close()

	var jml RingkasanInvestorRentang
	for _, x := range daftar {
		gerak, err := GerakInvestor(db, x.id)
		if err != nil {
			return t, err
		}
		r := HitungRingkasanRentang(x.acuan, x.bp, gerak, tDari, tSampai)
		jml.PokokAwal += r.PokokAwal
		jml.Setor += r.Setor
		jml.TarikPokok += r.TarikPokok
		jml.PokokAkhir += r.PokokAkhir
		jml.BungaTempo += r.BungaTempo
		jml.BungaDibayar += r.BungaDibayar
		jml.BungaBelum += r.BungaBelum
		t.Baris = append(t.Baris, []interface{}{x.nama, fmtTgl(x.acuan), float64(r.PokokAwal), float64(r.Setor), float64(r.TarikPokok),
			float64(r.PokokAkhir), float64(r.BungaTempo), float64(r.BungaDibayar), float64(r.BungaBelum)})
	}
	t.Baris = append(t.Baris,
		[]interface{}{"TOTAL", "", float64(jml.PokokAwal), float64(jml.Setor), float64(jml.TarikPokok), float64(jml.PokokAkhir),
			float64(jml.BungaTempo), float64(jml.BungaDibayar), float64(jml.BungaBelum)},
		[]interface{}{"Kewajiban owner di akhir periode (pokok akhir + bunga belum dibayar)", "", nil, nil, nil,
			float64(jml.PokokAkhir + jml.BungaBelum), nil, nil, nil},
	)
	t.Tebal = []int{len(t.Baris) - 2, len(t.Baris) - 1}
	return t, nil
}

// ---------- MUTASI INVESTOR (rentang tanggal) ----------

func TabelInvestorMutasi(db *sql.DB, dari, sampai string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "investor_mutasi",
		Judul: "Investor (mutasi periode)",
		Kolom: []string{"Tanggal", "Investor", "Jenis", "Nominal (Rp)", "Catatan", "Status", "Diinput oleh"},
		Tipe:  []string{"teks", "teks", "teks", "rupiah", "teks", "teks", "teks"},
	}
	rows, err := db.Query(`
		SELECT m.tx_date, i.name, m.jenis, m.amount, IFNULL(m.note,''), m.is_void, IFNULL(m.void_reason,''), IFNULL(u.username,'')
		FROM investor_movements m
		JOIN investors i ON i.id = m.investor_id
		LEFT JOIN users u ON u.id = m.created_by
		WHERE m.tx_date BETWEEN ? AND ?
		ORDER BY m.tx_date, m.id`, dari, sampai)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	total := map[string]int64{}
	for rows.Next() {
		var tgl time.Time
		var nama, jenis, note, alasan, user string
		var amt int64
		var batal bool
		if err := rows.Scan(&tgl, &nama, &jenis, &amt, &note, &batal, &alasan, &user); err != nil {
			return t, err
		}
		label := labelGerakLaporan[jenis]
		if label == "" {
			label = jenis
		}
		status := "Aktif"
		if batal {
			status = "Dibatalkan: " + alasan
		} else {
			total[jenis] += amt
		}
		t.Baris = append(t.Baris, []interface{}{fmtTgl(tgl), nama, label, float64(amt), note, status, user})
	}
	if err := rows.Err(); err != nil {
		return t, err
	}
	for _, j := range []string{GerakSetor, GerakTarikPokok, GerakBayarBunga} {
		t.Tebal = append(t.Tebal, len(t.Baris))
		t.Baris = append(t.Baris, []interface{}{fmt.Sprintf("TOTAL %s (tidak termasuk yang dibatalkan)", labelGerakLaporan[j]), "", "", float64(total[j]), "", "", ""})
	}
	return t, nil
}
