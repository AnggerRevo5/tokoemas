package services

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"toko-emas/models"
)

func LabelArah(a string) string {
	if a == "in" {
		return "Masuk"
	}
	return "Keluar"
}

var OpMasuk = map[string]bool{"omzet": true, "jual_barang": true, "jual_aset": true, "biaya_service": true}

var OpKeluar = map[string]bool{"pengeluaran_toko": true, "pembelian_customer": true, "operasional": true, "bulanan_investor": true}

var DanaNonOp = map[string]bool{"investor": true, "pinjaman": true, "bayar_hutang": true}

var DaftarLaporan = []string{"kas", "labarugi", "hutang", "penjualan", "cuci", "lebur", "stok"}

var namaKategori = map[string]string{
	"omzet": "Hasil jual (omzet)", "biaya_service": "Biaya service", "tambah_kas": "Tambahan kas", "pengeluaran_toko": "Pengeluaran toko",
	"pembelian_customer": "Pembelian dari customer", "transfer_bo": "Transfer kas ke back office",
	"dari_toko": "Pemasukan dari toko", "jual_barang": "Pemasukan dari jual barang",
	"investor": "Pemasukan dari investor", "pinjaman": "Pinjaman", "jual_aset": "Penjualan aset",
	"kembali_cabang": "Barang kembali cabang", "kas_toko": "Kas toko", "bulanan_investor": "Bulanan investor",
	"operasional": "Pengeluaran operasional", "bayar_hutang": "Pembayaran hutang",
	"beli_customer": "Beli dari customer", "stok_awal": "Stok awal", "pabrik": "Pabrik",
	"konsinyasi": "Konsinyasi", "barang_masuk": "Barang masuk", "ke_toko": "Oper ke toko",
	"kembali_bo": "Dikembalikan ke back office", "cuci": "Masuk proses cuci", "cucian": "Selesai cuci",
	"lebur": "Masuk proses lebur", "leburan_kotor": "Leburan kotor", "murnikan": "Dimurnikan jadi cukim",
	"cukim": "Cukim", "jual_penadah": "Jual ke penadah", "laku": "Laku", "pegadaian": "Pegadaian", "jual": "Jual",
}

func labelK(k string) string {
	if v, ok := namaKategori[k]; ok {
		return v
	}
	return strings.ReplaceAll(k, "_", " ")
}

func fmtTgl(t time.Time) string { return t.Format("2006-01-02") }

func nilF(n sql.NullFloat64) interface{} {
	if n.Valid {
		return n.Float64
	}
	return nil
}

func urutKunci(m map[string]float64) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

const sqlBarangGabung = `GROUP_CONCAT(CONCAT(i.item_name,' ',IFNULL(i.karat,''),' ',i.weight_gram,'g') SEPARATOR '; ')`

// ---------- KAS ----------

func TabelKas(db *sql.DB, dari, sampai, loc string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "kas",
		Judul: "Kas",
		Kolom: []string{"Tanggal", "Lokasi", "Kategori", "Masuk (Rp)", "Keluar (Rp)", "Catatan", "Diinput oleh"},
		Tipe:  []string{"teks", "teks", "teks", "rupiah", "rupiah", "teks", "teks"},
	}
	q := `SELECT x.transaction_date, l.name, x.direction, x.category, x.amount, IFNULL(x.note,''), IFNULL(u.username,'')
	      FROM cash_transactions x
	      JOIN locations l ON l.id = x.location_id
	      LEFT JOIN users u ON u.id = x.created_by
	      WHERE x.is_void = FALSE AND x.transaction_date BETWEEN ? AND ?`
	args := []interface{}{dari, sampai}
	if loc != "" {
		q += " AND x.location_id = ?"
		args = append(args, loc)
	}
	q += " ORDER BY x.transaction_date, x.id"
	rows, err := db.Query(q, args...)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	var masuk, keluar float64
	for rows.Next() {
		var tgl time.Time
		var lok, arah, kat, note, user string
		var amt float64
		if err := rows.Scan(&tgl, &lok, &arah, &kat, &amt, &note, &user); err != nil {
			return t, err
		}
		var m, k interface{}
		if arah == "in" {
			m = amt
			masuk += amt
		} else {
			k = amt
			keluar += amt
		}
		t.Baris = append(t.Baris, []interface{}{fmtTgl(tgl), lok, labelK(kat), m, k, note, user})
	}
	t.Baris = append(t.Baris,
		[]interface{}{"TOTAL", "", "", masuk, keluar, "", ""},
		[]interface{}{"SELISIH (masuk dikurangi keluar)", "", "", masuk - keluar, nil, "", ""},
	)
	t.Tebal = []int{len(t.Baris) - 2, len(t.Baris) - 1}
	return t, nil
}

// ---------- RIWAYAT STOK ----------

func TabelStok(db *sql.DB, dari, sampai, loc string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "stok",
		Judul: "Riwayat Stok",
		Kolom: []string{"Tanggal", "Lokasi", "Arah", "Kategori", "Barang", "Kadar", "Berat (g)", "Catatan", "Diinput oleh"},
		Tipe:  []string{"teks", "teks", "teks", "teks", "teks", "teks", "gram", "teks", "teks"},
	}
	q := `SELECT m.transaction_date, l.name, m.direction, m.category, m.item_name, IFNULL(m.karat,''),
	             m.weight_gram, IFNULL(m.note,''), IFNULL(u.username,'')
	      FROM stock_movements m
	      JOIN locations l ON l.id = m.location_id
	      LEFT JOIN users u ON u.id = m.created_by
	      WHERE m.is_void = FALSE AND m.transaction_date BETWEEN ? AND ?`
	args := []interface{}{dari, sampai}
	if loc != "" {
		q += " AND m.location_id = ?"
		args = append(args, loc)
	}
	q += " ORDER BY m.transaction_date, m.id"
	rows, err := db.Query(q, args...)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	var masuk, keluar float64
	for rows.Next() {
		var tgl time.Time
		var lok, arah, kat, item, karat, note, user string
		var berat float64
		if err := rows.Scan(&tgl, &lok, &arah, &kat, &item, &karat, &berat, &note, &user); err != nil {
			return t, err
		}
		if arah == "in" {
			masuk += berat
		} else {
			keluar += berat
		}
		t.Baris = append(t.Baris, []interface{}{fmtTgl(tgl), lok, LabelArah(arah), labelK(kat), item, karat, berat, note, user})
	}
	t.Baris = append(t.Baris,
		[]interface{}{"TOTAL MASUK", "", "", "", "", "", masuk, "", ""},
		[]interface{}{"TOTAL KELUAR", "", "", "", "", "", keluar, "", ""},
	)
	t.Tebal = []int{len(t.Baris) - 2, len(t.Baris) - 1}
	return t, nil
}

// ---------- CUCI ----------

func TabelCuci(db *sql.DB, dari, sampai string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "cuci",
		Judul: "Cuci",
		Kolom: []string{"No", "Mulai", "Asal", "Pencuci", "Barang", "Jumlah (pcs)", "Selesai (pcs)", "Berat sistem (g)",
			"Berat awal selesai (g)", "Berat sesudah (g)", "Susut (g)", "Status", "Tanggal selesai", "Catatan"},
		Tipe: []string{"teks", "teks", "teks", "teks", "teks", "angka", "angka", "gram", "gram", "gram", "gram", "teks", "teks", "teks"},
	}
	rows, err := db.Query(`
		SELECT w.id, w.start_date, l.name, w.washer_name, w.qty, w.weight_gram, w.status, w.finish_date, IFNULL(w.note,''),
		       (SELECT COUNT(*) FROM items i WHERE i.washing_job_id = w.id AND i.status = 'cuci' AND i.is_void = FALSE),
		       IFNULL((SELECT `+sqlBarangGabung+` FROM items i WHERE i.washing_job_id = w.id),''),
		       IFNULL(w.cancel_returned,0), IFNULL(w.cancel_reason,''),
		       IFNULL((SELECT SUM(x.weight_before) FROM washing_job_items x WHERE x.job_id = w.id AND x.weight_after IS NOT NULL),0),
		       IFNULL((SELECT SUM(x.weight_after) FROM washing_job_items x WHERE x.job_id = w.id),0)
		FROM washing_jobs w
		JOIN locations l ON l.id = w.origin_location_id
		WHERE w.is_void = FALSE AND w.start_date BETWEEN ? AND ?
		ORDER BY w.start_date, w.id`, dari, sampai)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	var totQty, totSelesai int
	var totSistem, totAwal, totSesudah float64
	for rows.Next() {
		var id, qty, sisa, kembali int
		var mulai time.Time
		var asal, pencuci, status, note, barang, alasan string
		var berat, awal, sesudah float64
		var fin sql.NullTime
		if err := rows.Scan(&id, &mulai, &asal, &pencuci, &qty, &berat, &status, &fin, &note, &sisa, &barang,
			&kembali, &alasan, &awal, &sesudah); err != nil {
			return t, err
		}
		selesai := qty - sisa
		st := "Proses"
		switch status {
		case "selesai":
			st = "Selesai"
		case "menunggu":
			st = "Menunggu PJ"
		case "batal":
			st = "Dibatalkan"
			selesai = qty - kembali
			if alasan != "" {
				if note != "" {
					note += " | "
				}
				note += "Batal: " + alasan
			}
		}
		tglSelesai := ""
		if fin.Valid {
			tglSelesai = fmtTgl(fin.Time)
		}
		var vAwal, vSesudah, vSusut interface{}
		if sesudah > 0 {
			vAwal, vSesudah, vSusut = awal, sesudah, awal-sesudah
			totAwal += awal
			totSesudah += sesudah
		}
		totQty += qty
		totSelesai += selesai
		totSistem += berat
		t.Baris = append(t.Baris, []interface{}{fmt.Sprintf("#%d", id), fmtTgl(mulai), asal, pencuci, barang, qty, selesai,
			berat, vAwal, vSesudah, vSusut, st, tglSelesai, note})
	}
	t.Baris = append(t.Baris, []interface{}{"TOTAL", "", "", "", "", totQty, totSelesai, totSistem, totAwal, totSesudah, totAwal - totSesudah, "", "", ""})
	t.Tebal = []int{len(t.Baris) - 1}
	return t, nil
}

// ---------- LEBUR ----------

func TabelLebur(db *sql.DB, dari, sampai string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "lebur",
		Judul: "Lebur",
		Kolom: []string{"No", "Mulai", "Barang", "Berat awal (g)", "Setelah dipotong (g)", "Leburan kotor (g)", "Cukim (g)",
			"Susut (g)", "Modal (Rp)", "Tahap", "PJ", "Penadah", "Harga jual (Rp)", "Untung / rugi (Rp)"},
		Tipe: []string{"teks", "teks", "teks", "gram", "gram", "gram", "gram", "gram", "rupiah", "teks", "teks", "teks", "rupiah", "rupiah"},
	}
	rows, err := db.Query(`
		SELECT b.id, b.start_date, b.status, b.source_weight, b.modal, b.cut_weight, b.leburan_weight, b.cukim_weight,
		       IFNULL(u1.username,''), IFNULL(u2.username,''),
		       IFNULL((SELECT `+sqlBarangGabung+` FROM items i WHERE i.melt_id = b.id AND i.jenis <> 'hasil_lebur'),''),
		       ps.sale_total, IFNULL(ps.penadah_name,'')
		FROM melt_batches b
		LEFT JOIN users u1 ON u1.id = b.leburan_by
		LEFT JOIN users u2 ON u2.id = b.cukim_by
		LEFT JOIN items ci ON ci.id = b.cukim_item_id
		LEFT JOIN penadah_sales ps ON ps.id = ci.sale_id AND ps.is_void = FALSE
		WHERE b.is_void = FALSE AND b.status <> 'batal' AND b.start_date BETWEEN ? AND ?
		ORDER BY b.start_date, b.id`, dari, sampai)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	tahap := map[string]string{"proses": "Proses lebur", "leburan_kotor": "Leburan kotor", "cukim": "Cukim siap jual", "terjual": "Terjual"}
	var sumSrc, sumModal, sumJual, sumUntung float64
	for rows.Next() {
		var id int
		var mulai time.Time
		var status, u1, u2, barang, penadah string
		var src, modal float64
		var cut, leb, cuk, sale sql.NullFloat64
		if err := rows.Scan(&id, &mulai, &status, &src, &modal, &cut, &leb, &cuk, &u1, &u2, &barang, &sale, &penadah); err != nil {
			return t, err
		}
		pj := u1
		if u2 != "" && u2 != u1 {
			if pj != "" {
				pj += " / "
			}
			pj += u2
		}
		var susut, untung interface{}
		if cuk.Valid {
			susut = src - cuk.Float64
		}
		if sale.Valid {
			untung = sale.Float64 - modal
			sumJual += sale.Float64
			sumUntung += sale.Float64 - modal
		}
		sumSrc += src
		sumModal += modal
		t.Baris = append(t.Baris, []interface{}{fmt.Sprintf("#%d", id), fmtTgl(mulai), barang, src, nilF(cut), nilF(leb), nilF(cuk),
			susut, modal, tahap[status], pj, penadah, nilF(sale), untung})
	}
	t.Baris = append(t.Baris, []interface{}{"TOTAL", "", "", sumSrc, nil, nil, nil, nil, sumModal, "", "", "", sumJual, sumUntung})
	t.Tebal = []int{len(t.Baris) - 1}
	return t, nil
}

// ---------- PENJUALAN KE PENADAH ----------

func TabelPenjualan(db *sql.DB, dari, sampai string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "penjualan",
		Judul: "Penjualan Penadah",
		Kolom: []string{"Tanggal", "Penadah", "Barang", "Berat (g)", "Modal (Rp)", "Harga jual (Rp)", "Untung / rugi (Rp)", "PJ", "Catatan"},
		Tipe:  []string{"teks", "teks", "teks", "gram", "rupiah", "rupiah", "rupiah", "teks", "teks"},
	}
	rows, err := db.Query(`
		SELECT s.sale_date, s.penadah_name,
		       IFNULL((SELECT `+sqlBarangGabung+` FROM items i WHERE i.sale_id = s.id),''),
		       s.weight_gram, s.modal, s.sale_total, IFNULL(u.username,''), IFNULL(s.note,'')
		FROM penadah_sales s
		LEFT JOIN users u ON u.id = s.created_by
		WHERE s.is_void = FALSE AND s.sale_date BETWEEN ? AND ?
		ORDER BY s.sale_date, s.id`, dari, sampai)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	var sumBerat, sumModal, sumJual float64
	for rows.Next() {
		var tgl time.Time
		var penadah, barang, pj, note string
		var berat, modal, jual float64
		if err := rows.Scan(&tgl, &penadah, &barang, &berat, &modal, &jual, &pj, &note); err != nil {
			return t, err
		}
		sumBerat += berat
		sumModal += modal
		sumJual += jual
		t.Baris = append(t.Baris, []interface{}{fmtTgl(tgl), penadah, barang, berat, modal, jual, jual - modal, pj, note})
	}
	t.Baris = append(t.Baris, []interface{}{"TOTAL", "", "", sumBerat, sumModal, sumJual, sumJual - sumModal, "", ""})
	t.Tebal = []int{len(t.Baris) - 1}
	return t, nil
}

// ---------- HUTANG (posisi saat ini) ----------

func TabelHutang(db *sql.DB) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "hutang",
		Judul: "Hutang",
		Kolom: []string{"Kreditur", "Jenis", "Total hutang (Rp)", "Terbayar (Rp)", "Sisa (Rp)", "Mulai", "Jatuh tempo", "Status"},
		Tipe:  []string{"teks", "teks", "rupiah", "rupiah", "rupiah", "teks", "teks", "teks"},
	}
	rows, err := db.Query(`
		SELECT d.creditor_name, d.creditor_type, d.principal,
		       IFNULL((SELECT SUM(p.amount) FROM debt_payments p WHERE p.debt_id = d.id),0),
		       d.start_date, d.due_date, d.status
			   FROM debts d WHERE d.is_void = FALSE ORDER BY d.status, d.id`)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	jenis := map[string]string{"investor": "Investor", "perorangan": "Perorangan", "pabrik": "Pabrik", "bank_pegadaian": "Bank / Pegadaian"}
	var sumP, sumT float64
	for rows.Next() {
		var nama, tipe, status string
		var pokok, terbayar float64
		var mulai time.Time
		var tempo sql.NullTime
		if err := rows.Scan(&nama, &tipe, &pokok, &terbayar, &mulai, &tempo, &status); err != nil {
			return t, err
		}
		tt := ""
		if tempo.Valid {
			tt = fmtTgl(tempo.Time)
		}
		st := "Aktif"
		if status == "lunas" {
			st = "Lunas"
		}
		sumP += pokok
		sumT += terbayar
		t.Baris = append(t.Baris, []interface{}{nama, jenis[tipe], pokok, terbayar, pokok - terbayar, fmtTgl(mulai), tt, st})
	}
	t.Baris = append(t.Baris, []interface{}{"TOTAL", "", sumP, sumT, sumP - sumT, "", "", ""})
	t.Tebal = []int{len(t.Baris) - 1}
	return t, nil
}

// ---------- LABA RUGI ----------

func TabelLabaRugi(db *sql.DB, dari, sampai string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "labarugi",
		Judul: "Laba Rugi",
		Kolom: []string{"Keterangan", "Jumlah (Rp)"},
		Tipe:  []string{"teks", "rupiah"},
	}
	rows, err := db.Query(`SELECT category, SUM(amount) FROM cash_transactions
	                       WHERE is_void = FALSE AND transaction_date BETWEEN ? AND ? GROUP BY category`, dari, sampai)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	masuk, keluar, dana := map[string]float64{}, map[string]float64{}, map[string]float64{}
	var totM, totK float64
	for rows.Next() {
		var k string
		var v float64
		if err := rows.Scan(&k, &v); err != nil {
			return t, err
		}
		switch {
		case OpMasuk[k]:
			masuk[k] = v
			totM += v
		case OpKeluar[k]:
			keluar[k] = v
			totK += v
		case DanaNonOp[k]:
			dana[k] = v
		}
	}

	add := func(ket string, v interface{}, tebal bool) {
		if tebal {
			t.Tebal = append(t.Tebal, len(t.Baris))
		}
		t.Baris = append(t.Baris, []interface{}{ket, v})
	}
	add("UANG MASUK (OPERASIONAL)", nil, true)
	for _, k := range urutKunci(masuk) {
		add("    "+labelK(k), masuk[k], false)
	}
	add("Total uang masuk", totM, true)
	add("", nil, false)
	add("UANG KELUAR (OPERASIONAL)", nil, true)
	for _, k := range urutKunci(keluar) {
		add("    "+labelK(k), keluar[k], false)
	}
	add("Total uang keluar", totK, true)
	add("", nil, false)
	add("LABA / RUGI", totM-totK, true)
	add("", nil, false)
	add("DANA NON-OPERASIONAL (tidak dihitung sebagai laba)", nil, true)
	for _, k := range urutKunci(dana) {
		add("    "+labelK(k), dana[k], false)
	}
	return t, nil
}

// ---------- PENYIAPAN DAN PENGIRIMAN ----------
