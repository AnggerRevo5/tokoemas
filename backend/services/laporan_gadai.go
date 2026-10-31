package services

import (
	"database/sql"
	"strconv"
	"time"

	"toko-emas/models"
)

func fmtInt(n int64) string { return strconv.FormatInt(n, 10) }

// ---------- GADAI (ringkasan dan jadwal cicilan) ----------

var labelStatusGadai = map[string]string{
	GadaiAktif:        "Berjalan",
	GadaiLunas:        "Lunas, menunggu diambil",
	GadaiKembali:      "Sudah kembali ke toko",
	GadaiTidakDitebus: "Tidak ditebus",
}

// TabelGadai memuat semua gadai yang dimulai pada rentang tanggal, ditambah gadai yang masih berjalan atau
// menunggu diambil walau dimulai sebelum rentang itu, supaya jadwal cicilan yang belum selesai tidak hilang.
func TabelGadai(db *sql.DB, dari, sampai, loc string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "gadai",
		Judul: "Gadai (ringkasan dan jadwal cicilan)",
		Kolom: []string{"Tanggal gadai", "Lokasi", "Barang", "Berat (g)", "Pegadaian", "Dana diterima (Rp)", "Tenor", "Cicilan (Rp)",
			"Dibayar (Rp)", "Denda (Rp)", "Diskon (Rp)", "Sisa (Rp)", "Sisa bulan", "Jatuh tempo berikut", "Status"},
		Tipe: []string{"teks", "teks", "teks", "gram", "teks", "rupiah", "teks", "rupiah", "rupiah", "rupiah", "rupiah", "rupiah", "teks", "teks", "teks"},
	}
	q := `SELECT g.start_date, l.name, CONCAT(IFNULL(i.item_no,''), ' ', i.item_name, ' ', IFNULL(i.karat,'')), i.weight_gram,
	             g.pegadaian, g.dana, g.tenor, g.cicilan, g.total, g.first_due, g.status,
	             IFNULL(p.dibayar,0), IFNULL(p.denda,0), IFNULL(p.diskon,0), IFNULL(p.bulan,0)
	      FROM gadai g
	      JOIN items i ON i.id = g.item_id
	      JOIN locations l ON l.id = g.location_id
	      LEFT JOIN (SELECT gadai_id, SUM(amount) AS dibayar, SUM(denda) AS denda, SUM(diskon) AS diskon,
	                        SUM(jenis = 'cicilan') AS bulan
	                 FROM gadai_pembayaran WHERE is_void = FALSE GROUP BY gadai_id) p ON p.gadai_id = g.id
	      WHERE g.is_void = FALSE AND (g.start_date BETWEEN ? AND ? OR g.status IN ('aktif','lunas'))`
	args := []interface{}{dari, sampai}
	if loc != "" {
		q += " AND g.location_id = ?"
		args = append(args, loc)
	}
	q += " ORDER BY g.start_date, g.id"
	rows, err := db.Query(q, args...)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	var sumDana, sumDibayar, sumDenda, sumDiskon, sumSisa float64
	for rows.Next() {
		var mulai, tempo time.Time
		var lok, barang, pegadaian, status string
		var berat float64
		var dana, tenor, cicilan, total, dibayar, denda, diskon, bulan int64
		if err := rows.Scan(&mulai, &lok, &barang, &berat, &pegadaian, &dana, &tenor, &cicilan, &total, &tempo, &status,
			&dibayar, &denda, &diskon, &bulan); err != nil {
			return t, err
		}
		sisa := SisaGadai(total, dibayar, diskon)
		sisaBulan, berikut := "-", "-"
		if status == GadaiAktif && sisa > 0 {
			sb := tenor - bulan
			if sb < 0 {
				sb = 0
			}
			sisaBulan = fmtInt(sb)
			berikut = fmtTgl(JatuhTempoKe(TanggalSaja(tempo), int(bulan)))
		}
		if status == GadaiTidakDitebus || status == GadaiKembali {
			sisa = 0
		}
		sumDana += float64(dana)
		sumDibayar += float64(dibayar)
		sumDenda += float64(denda)
		sumDiskon += float64(diskon)
		sumSisa += float64(sisa)
		t.Baris = append(t.Baris, []interface{}{fmtTgl(mulai), lok, barang, berat, pegadaian, float64(dana), fmtInt(tenor) + " bln", float64(cicilan),
			float64(dibayar), float64(denda), float64(diskon), float64(sisa), sisaBulan, berikut, labelStatusGadai[status]})
	}
	if err := rows.Err(); err != nil {
		return t, err
	}
	t.Baris = append(t.Baris, []interface{}{"TOTAL", "", "", nil, "", sumDana, "", nil, sumDibayar, sumDenda, sumDiskon, sumSisa, "", "", ""})
	t.Tebal = []int{len(t.Baris) - 1}
	return t, nil
}
