package controllers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/models"
	"toko-emas/utils"
)

// kelompokStok: pengelompokan kategori pergerakan stok untuk filter "Jenis" di riwayat stok.
var kelompokStok = map[string][]string{
	"oper":  {"ke_toko", "kembali_bo", "kembali_cabang", "ke_bp", "kembali_bp"},
	"masuk": {"barang_masuk", "beli_customer", "stok_awal", "pabrik", "konsinyasi", "barang_toko"},
	"jual":  {"laku", "jual", "jual_penadah", "pegadaian"},
	"olah":  {"cuci", "cucian", "lebur", "leburan_kotor", "murnikan", "cukim", "batal_cuci", "batal_lebur"},
}

// kiriman dari back office tercatat sebagai barang_masuk di toko; dikenali dari catatannya
const catatKiriman = "m.note LIKE 'Kiriman dari%'"

// RiwayatStok: riwayat pergerakan stok (oper, barang masuk, jual, cuci, lebur) dengan filter dan ringkasan.
// Akun cabang hanya melihat lokasinya sendiri.
func RiwayatStok(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if !utils.TanggalOpsional(c, "from", "to") {
			return
		}
		lokasi := c.Query("location_id")
		if cl.Role == "cabang" {
			lokasi = strconv.Itoa(cl.LocationID)
		}
		if lokasi == "0" {
			lokasi = ""
		}

		fil, args := "", []interface{}{}
		if lokasi != "" {
			fil += " AND m.location_id = ?"
			args = append(args, lokasi)
		}
		if v := c.Query("from"); v != "" {
			fil += " AND m.transaction_date >= ?"
			args = append(args, v)
		}
		if v := c.Query("to"); v != "" {
			fil += " AND m.transaction_date <= ?"
			args = append(args, v)
		}
		if v := c.Query("arah"); v == "in" || v == "out" {
			fil += " AND m.direction = ?"
			args = append(args, v)
		}
		switch k := c.Query("kelompok"); {
		case kelompokStok[k] != nil:
			fil += " AND (m.category IN (" + utils.Placeholders(len(kelompokStok[k])) + ")"
			for _, kat := range kelompokStok[k] {
				args = append(args, kat)
			}
			switch k {
			case "oper":
				fil += " OR (m.category = 'barang_masuk' AND " + catatKiriman + "))"
			case "masuk":
				fil += ") AND NOT (m.category = 'barang_masuk' AND " + catatKiriman + ")"
			default:
				fil += ")"
			}
		}
		if v := strings.TrimSpace(c.Query("q")); v != "" {
			pola := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(utils.BatasTeks(v, 60)) + "%"
			fil += " AND (m.item_name LIKE ? OR IFNULL(m.note,'') LIKE ? OR IFNULL(i.item_no,'') LIKE ?)"
			args = append(args, pola, pola, pola)
		}

		batas := 200
		if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 {
			batas = v
		}
		if batas > 2000 {
			batas = 2000
		}

		const dari = ` FROM stock_movements m JOIN locations l ON l.id = m.location_id
		               LEFT JOIN items i ON i.id = m.item_id LEFT JOIN users u ON u.id = m.created_by
		               WHERE m.is_void = FALSE`
		rows, err := db.Query(
			`SELECT m.id, m.location_id, l.name, m.direction, m.category, m.item_name, IFNULL(i.item_no,''), m.weight_gram,
			        IFNULL(m.karat,''), m.qty, IFNULL(m.note,''), DATE_FORMAT(m.transaction_date,'%Y-%m-%d'), IFNULL(u.username,'')`+
				dari+fil+" ORDER BY m.transaction_date DESC, m.id DESC LIMIT ?", append(append([]interface{}{}, args...), batas+1)...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []gin.H{}
		for rows.Next() {
			var id, loc, qty int
			var toko, arah, kat, nama, noItem, karat, note, tgl, oleh string
			var berat float64
			if err := rows.Scan(&id, &loc, &toko, &arah, &kat, &nama, &noItem, &berat, &karat, &qty, &note, &tgl, &oleh); err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, gin.H{"id": id, "location_id": loc, "lokasi": toko, "direction": arah, "category": kat, "item_name": nama,
				"item_no": noItem, "weight_gram": berat, "karat": karat, "qty": qty, "note": note, "transaction_date": tgl, "oleh": oleh})
		}
		adaLagi := len(out) > batas
		if adaLagi {
			out = out[:batas]
		}

		var masukPcs, keluarPcs, masukGram, keluarGram float64
		if err := db.QueryRow(
			`SELECT IFNULL(SUM(IF(m.direction='in', m.qty, 0)),0), IFNULL(SUM(IF(m.direction='out', m.qty, 0)),0),
			        IFNULL(SUM(IF(m.direction='in', m.weight_gram*m.qty, 0)),0), IFNULL(SUM(IF(m.direction='out', m.weight_gram*m.qty, 0)),0)`+
				dari+fil, args...).Scan(&masukPcs, &keluarPcs, &masukGram, &keluarGram); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "masuk_pcs": masukPcs, "keluar_pcs": keluarPcs, "masuk_gram": masukGram,
			"keluar_gram": keluarGram, "ada_lagi": adaLagi, "batas": batas})
	}
}
