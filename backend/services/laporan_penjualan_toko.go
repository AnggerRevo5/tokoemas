package services

import (
	"database/sql"
	"time"

	"toko-emas/models"
)

// dasar kedua laporan: barang yang dijual dari menu Stok (bukan ke penadah) pada rentang tanggal jual.
// Metode pembayaran diambil dari catatan kas penjualannya; penjualan lama tanpa catatan dianggap tunai.
const sqlDasarPenjualanToko = `
	FROM items i
	JOIN locations l ON l.id = i.location_id
	LEFT JOIN cash_transactions jc ON jc.id = i.jual_cash_id
	WHERE i.is_void = FALSE AND i.status = 'terjual' AND i.sale_id IS NULL
	  AND i.harga_jual IS NOT NULL AND i.tanggal_jual BETWEEN ? AND ?`

// ---------- PENJUALAN TOKO (rincian per barang) ----------

func TabelPenjualanToko(db *sql.DB, dari, sampai, loc string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "penjualan_toko",
		Judul: "Penjualan Toko (rincian)",
		Kolom: []string{"Tanggal", "Lokasi", "Barang", "Berat (g)", "Modal (Rp)", "Harga jual (Rp)", "Untung / rugi (Rp)", "Metode"},
		Tipe:  []string{"teks", "teks", "teks", "gram", "rupiah", "rupiah", "rupiah", "teks"},
	}
	q := `SELECT i.tanggal_jual, l.name, CONCAT(IFNULL(i.item_no,''), ' ', i.item_name, ' ', IFNULL(i.karat,'')),
	             i.weight_gram, i.harga_beli, i.harga_jual, IFNULL(jc.metode, 'tunai')` + sqlDasarPenjualanToko
	args := []interface{}{dari, sampai}
	if loc != "" {
		q += " AND i.location_id = ?"
		args = append(args, loc)
	}
	q += " ORDER BY i.tanggal_jual, i.id"
	rows, err := db.Query(q, args...)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	var sumBerat, sumModal, sumJual float64
	for rows.Next() {
		var tgl time.Time
		var lok, barang, metode string
		var berat, modal, jual float64
		if err := rows.Scan(&tgl, &lok, &barang, &berat, &modal, &jual, &metode); err != nil {
			return t, err
		}
		sumBerat += berat
		sumModal += modal
		sumJual += jual
		t.Baris = append(t.Baris, []interface{}{fmtTgl(tgl), lok, barang, berat, modal, jual, jual - modal, LabelMetode(metode)})
	}
	if err := rows.Err(); err != nil {
		return t, err
	}
	t.Baris = append(t.Baris, []interface{}{"TOTAL", "", "", sumBerat, sumModal, sumJual, sumJual - sumModal, ""})
	t.Tebal = []int{len(t.Baris) - 1}
	return t, nil
}

// ---------- PENJUALAN TOKO (ringkasan per lokasi dan metode) ----------

func TabelPenjualanTokoRingkas(db *sql.DB, dari, sampai, loc string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "penjualan_toko_ringkas",
		Judul: "Penjualan Toko (ringkasan per lokasi dan metode)",
		Kolom: []string{"Lokasi", "Tunai (Rp)", "QRIS/Debit (Rp)", "Total penjualan (Rp)", "Modal (Rp)", "Untung / rugi (Rp)"},
		Tipe:  []string{"teks", "rupiah", "rupiah", "rupiah", "rupiah", "rupiah"},
	}
	q := `SELECT l.name,
	             IFNULL(SUM(IF(IFNULL(jc.metode, 'tunai') = 'qris', 0, i.harga_jual)), 0),
	             IFNULL(SUM(IF(jc.metode = 'qris', i.harga_jual, 0)), 0),
	             IFNULL(SUM(i.harga_jual), 0), IFNULL(SUM(i.harga_beli), 0)` + sqlDasarPenjualanToko
	args := []interface{}{dari, sampai}
	if loc != "" {
		q += " AND i.location_id = ?"
		args = append(args, loc)
	}
	q += " GROUP BY l.id, l.name ORDER BY l.name"
	rows, err := db.Query(q, args...)
	if err != nil {
		return t, err
	}
	defer rows.Close()

	var sumTunai, sumQRIS, sumJual, sumModal float64
	for rows.Next() {
		var lok string
		var tunai, qris, jual, modal float64
		if err := rows.Scan(&lok, &tunai, &qris, &jual, &modal); err != nil {
			return t, err
		}
		sumTunai += tunai
		sumQRIS += qris
		sumJual += jual
		sumModal += modal
		t.Baris = append(t.Baris, []interface{}{lok, tunai, qris, jual, modal, jual - modal})
	}
	if err := rows.Err(); err != nil {
		return t, err
	}
	t.Baris = append(t.Baris, []interface{}{"TOTAL", sumTunai, sumQRIS, sumJual, sumModal, sumJual - sumModal})
	t.Tebal = []int{len(t.Baris) - 1}
	return t, nil
}
