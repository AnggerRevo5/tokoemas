package controllers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/utils"
)

const sqlTukar = `SELECT x.id, x.cabang_id, bp.nama, CAST(x.total_masuk AS SIGNED), CAST(x.total_keluar AS SIGNED), CAST(x.saldo AS SIGNED),
	x.arah_uang, IFNULL(x.metode_bayar,''), IFNULL(CAST(x.nominal_bayar AS SIGNED),0), x.no_referensi, x.status_uang,
	(x.metode_bayar = 'transfer' AND x.no_referensi = ''), IFNULL(x.koreksi_alasan,''), IFNULL(u.username,''),
	DATE_FORMAT(x.tanggal,'%Y-%m-%d'), (SELECT COUNT(*) FROM talangan_tukar_item i WHERE i.tukar_id = x.id), IFNULL(x.catatan,'')
	FROM talangan_tukar x JOIN bp_cabang bp ON bp.id = x.cabang_id LEFT JOIN users u ON u.id = x.pj_id`

func scanTukar(rows interface{ Scan(...interface{}) error }) (models.TukarRow, error) {
	var r models.TukarRow
	err := rows.Scan(&r.ID, &r.BPID, &r.BP, &r.TotalMasuk, &r.TotalKeluar, &r.Saldo, &r.ArahUang, &r.MetodeBayar, &r.NominalBayar,
		&r.NoReferensi, &r.StatusUang, &r.TanpaBukti, &r.KoreksiAlasan, &r.PJ, &r.Tanggal, &r.JumlahBarang, &r.Catatan)
	return r, err
}

// TukarList: daftar penukaran. Filter bp_id dan status_uang. Hanya PJ, owner, super admin.
func TukarList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaPeran(c, "pj", "owner", "super_admin"); !ok {
			return
		}
		q, args := sqlTukar+" WHERE 1=1", []interface{}{}
		if v := c.Query("bp_id"); v != "" {
			q += " AND x.cabang_id = ?"
			args = append(args, v)
		}
		if v := c.Query("status_uang"); v != "" {
			q += " AND x.status_uang = ?"
			args = append(args, v)
		}
		rows, err := db.Query(q+" ORDER BY x.id DESC LIMIT 500", args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		out := []models.TukarRow{}
		for rows.Next() {
			r, err := scanTukar(rows)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// TukarDetail: satu penukaran beserta barangnya.
func TukarDetail(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaPeran(c, "pj", "owner", "super_admin"); !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		r, err := scanTukar(db.QueryRow(sqlTukar+" WHERE x.id = ?", id))
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "penukaran tidak ditemukan"})
			return
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		rows, err := db.Query(
			`SELECT i.barang_id, i.arah, t.nama_barang, CAST(i.nilai AS SIGNED), t.status
			 FROM talangan_tukar_item i JOIN talangan_barang t ON t.id = i.barang_id WHERE i.tukar_id = ? ORDER BY i.id`, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		r.Items = []models.TukarItemRow{}
		for rows.Next() {
			var it models.TukarItemRow
			if err := rows.Scan(&it.BarangID, &it.Arah, &it.NamaBarang, &it.Nilai, &it.Status); err != nil {
				utils.Gagal(c, err)
				return
			}
			r.Items = append(r.Items, it)
		}
		c.JSON(http.StatusOK, r)
	}
}
