package services

import (
	"database/sql"
	"fmt"

	"toko-emas/models"
)

// TabelTalanganBP: satu laporan berisi tiga bagian: catatan talangan (periode), rekap selisih per BP (posisi saat ini),
// dan pengajuan tukar (periode). loc kosong = semua toko.
func TabelTalanganBP(db *sql.DB, dari, sampai, loc string) (models.TabelLaporan, error) {
	t := models.TabelLaporan{
		Kode:  "talangan_bp",
		Judul: "Talangan dan tukar BP",
		Kolom: []string{"Tanggal", "BP", "Jenis", "Barang / keterangan", "Kadar", "Berat (g)", "Nilai (Rp)", "Status", "Toko", "Catatan"},
		Tipe:  []string{"teks", "teks", "teks", "teks", "teks", "gram", "rupiah", "teks", "teks", "teks"},
	}
	judulBagian := func(teks string) {
		t.Baris = append(t.Baris, []interface{}{nil, nil, nil, teks, nil, nil, nil, nil, nil, nil})
		t.Tebal = append(t.Tebal, len(t.Baris)-1)
	}
	filterLok, argLok := "", []interface{}{}
	if loc != "" {
		filterLok = " AND x.location_id = ?"
		argLok = append(argLok, loc)
	}

	// 1. catatan talangan
	judulBagian("CATATAN TALANGAN (menurut tanggal catatan)")
	rows, err := db.Query(
		`SELECT x.entry_date, x.bp_nama, x.arah, x.item_name, IFNULL(x.karat,''), x.weight_gram, x.nilai, x.status, IFNULL(x.tukar_id,0),
		        l.name, IFNULL(x.note,''), x.konfirmasi, IFNULL(DATE_FORMAT(x.konfirmasi_tgl,'%Y-%m-%d'),''), IFNULL(x.konfirmasi_note,''),
		        IFNULL(x.batal_alasan,'')
		 FROM talangan x JOIN locations l ON l.id = x.location_id
		 WHERE x.entry_date BETWEEN ? AND ?`+filterLok+` ORDER BY x.bp_nama, x.id`, append([]interface{}{dari, sampai}, argLok...)...)
	if err != nil {
		return t, err
	}
	var kita, bpv float64
	for rows.Next() {
		var tgl sql.NullTime
		var bp, arah, nama, karat, status, toko, note, kTgl, kNote, alasan string
		var berat, nilai float64
		var tukarID int
		var konf bool
		if err := rows.Scan(&tgl, &bp, &arah, &nama, &karat, &berat, &nilai, &status, &tukarID, &toko, &note, &konf, &kTgl, &kNote, &alasan); err != nil {
			rows.Close()
			return t, err
		}
		jenis, st := "Barang kita di BP", "Aktif"
		if arah == "bp_di_kita" {
			jenis = "Barang BP di kita"
		}
		switch status {
		case "tukar":
			st = fmt.Sprintf("Diajukan tukar #%d", tukarID)
		case "batal":
			st = "Dibatalkan"
		}
		if status != "batal" {
			if arah == "kita_di_bp" {
				kita += nilai
			} else {
				bpv += nilai
			}
		}
		catat := note
		telp := "Telepon: belum"
		if konf {
			telp = "Telepon: sudah " + kTgl
			if kNote != "" {
				telp += " (" + kNote + ")"
			}
		}
		if catat != "" {
			catat += "; "
		}
		catat += telp
		if alasan != "" {
			catat += "; dibatalkan: " + alasan
		}
		var tg interface{}
		if tgl.Valid {
			tg = fmtTgl(tgl.Time)
		}
		var br interface{}
		if berat > 0 {
			br = berat
		}
		t.Baris = append(t.Baris, []interface{}{tg, bp, jenis, nama, karat, br, nilai, st, toko, catat})
	}
	rows.Close()
	t.Baris = append(t.Baris, []interface{}{nil, nil, "Total", "Barang kita di BP (tanpa yang dibatalkan)", nil, nil, kita, nil, nil, nil})
	t.Tebal = append(t.Tebal, len(t.Baris)-1)
	t.Baris = append(t.Baris, []interface{}{nil, nil, "Total", "Barang BP di kita (tanpa yang dibatalkan)", nil, nil, bpv, nil, nil, nil})
	t.Tebal = append(t.Tebal, len(t.Baris)-1)

	// 2. rekap selisih per BP (catatan aktif, belum diajukan tukar; posisi saat ini, tidak terikat periode)
	t.Baris = append(t.Baris, []interface{}{nil, nil, nil, nil, nil, nil, nil, nil, nil, nil})
	judulBagian("REKAP SELISIH PER BP (catatan aktif yang belum diajukan tukar, posisi saat ini)")
	rrows, err := db.Query(
		`SELECT x.bp_nama, IFNULL(SUM(IF(x.arah='kita_di_bp', x.nilai, 0)),0), IFNULL(SUM(IF(x.arah='bp_di_kita', x.nilai, 0)),0)
		 FROM talangan x WHERE x.status = 'aktif'`+filterLok+` GROUP BY x.bp_nama ORDER BY x.bp_nama`, argLok...)
	if err != nil {
		return t, err
	}
	for rrows.Next() {
		var bp string
		var a, b float64
		if err := rrows.Scan(&bp, &a, &b); err != nil {
			rrows.Close()
			return t, err
		}
		selisih := b - a
		st := "Seimbang"
		if selisih > 0 {
			st = "Kita bayar"
		} else if selisih < 0 {
			st = "BP bayar"
		}
		t.Baris = append(t.Baris, []interface{}{nil, bp, "Rekap",
			fmt.Sprintf("Barang kita di BP %s, barang BP di kita %s", FmtRp(a), FmtRp(b)), nil, nil, selisih, st, nil, nil})
	}
	rrows.Close()

	// 3. pengajuan tukar
	t.Baris = append(t.Baris, []interface{}{nil, nil, nil, nil, nil, nil, nil, nil, nil, nil})
	judulBagian("PENGAJUAN TUKAR (menurut tanggal tukar; nilai = selisih, positif berarti kita bayar)")
	prows, err := db.Query(
		`SELECT x.tukar_date, x.bp_nama, x.id, x.total_kita, x.total_bp, x.selisih, x.status, l.name, IFNULL(x.note,''), IFNULL(x.batal_alasan,''),
		        (SELECT COUNT(*) FROM talangan t WHERE t.tukar_id = x.id)
		 FROM tukar_bp x JOIN locations l ON l.id = x.location_id
		 WHERE x.tukar_date BETWEEN ? AND ?`+filterLok+` ORDER BY x.id`, append([]interface{}{dari, sampai}, argLok...)...)
	if err != nil {
		return t, err
	}
	defer prows.Close()
	for prows.Next() {
		var tgl sql.NullTime
		var bp, status, toko, note, alasan string
		var id, n int
		var a, b, selisih float64
		if err := prows.Scan(&tgl, &bp, &id, &a, &b, &selisih, &status, &toko, &note, &alasan, &n); err != nil {
			return t, err
		}
		st := map[string]string{"diajukan": "Menunggu owner", "disetujui": "Disetujui, masuk kas", "batal": "Dibatalkan"}[status]
		catat := note
		if alasan != "" {
			if catat != "" {
				catat += "; "
			}
			catat += "dibatalkan: " + alasan
		}
		var tg interface{}
		if tgl.Valid {
			tg = fmtTgl(tgl.Time)
		}
		t.Baris = append(t.Baris, []interface{}{tg, bp, fmt.Sprintf("Pengajuan tukar #%d", id),
			fmt.Sprintf("%d catatan: barang kita di BP %s, barang BP di kita %s", n, FmtRp(a), FmtRp(b)), nil, nil, selisih, st, toko, catat})
	}
	return t, nil
}
