package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func barangListV3(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)

		q := `SELECT i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), i.weight_gram,
		             i.jenis, IFNULL(i.brand,''), i.kondisi, i.harga_beli,
		             i.location_id, l.name, i.origin_location_id, o.name, i.status, i.entry_date, IFNULL(i.note,''),
		             i.kirim_tgl, i.belum_sampai, IFNULL(i.belum_sampai_note,''),
		             IF(i.status = 'cuci', IFNULL(w.washer_name,''), ''),
		             IF(i.status = 'cuci', w.start_date, NULL),
		             IFNULL(i.kirim_ke_location_id,0), IFNULL(k.name,'')
		      FROM items i
		      JOIN locations l ON l.id = i.location_id
		      JOIN locations o ON o.id = i.origin_location_id
		      LEFT JOIN locations k ON k.id = i.kirim_ke_location_id
		      LEFT JOIN washing_jobs w ON w.id = i.washing_job_id
		      WHERE i.is_void = FALSE`
		args := []interface{}{}

		if cl.Role == "cabang" {
			q += " AND (i.location_id = ? OR i.kirim_ke_location_id = ?)"
			args = append(args, cl.LocationID, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" && loc != "0" {
			q += " AND i.location_id = ?"
			args = append(args, loc)
		}
		if s := c.Query("status"); s == "dikirim" {
			q += " AND i.status IN ('dikirim','kembali')"
		} else if s != "" && s != "semua" {
			q += " AND i.status = ?"
			args = append(args, s)
		}
		if j := c.Query("jenis"); j == "perhiasan" || j == "logam_mulia" || j == "hasil_lebur" {
			q += " AND i.jenis = ?"
			args = append(args, j)
		}
		if s := strings.TrimSpace(c.Query("q")); s != "" {
			q += " AND (i.item_name LIKE ? OR IFNULL(i.note,'') LIKE ? OR (i.status = 'cuci' AND IFNULL(w.washer_name,'') LIKE ?))"
			args = append(args, "%"+s+"%", "%"+s+"%", "%"+s+"%")
		}
		q += " ORDER BY i.id DESC LIMIT 500"

		rows, err := db.Query(q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type row struct {
			ID          int        `json:"id"`
			ItemNo      string     `json:"item_no"`
			ItemName    string     `json:"item_name"`
			Karat       string     `json:"karat"`
			WeightGram  float64    `json:"weight_gram"`
			Jenis       string     `json:"jenis"`
			Brand       string     `json:"brand"`
			Kondisi     string     `json:"kondisi"`
			HargaBeli   float64    `json:"harga_beli"`
			LocationID  int        `json:"location_id"`
			Lokasi      string     `json:"lokasi"`
			OriginID    int        `json:"origin_location_id"`
			Asal        string     `json:"asal"`
			Status      string     `json:"status"`
			EntryDate   time.Time  `json:"entry_date"`
			Note        string     `json:"note"`
			KirimTgl    *time.Time `json:"kirim_tgl"`
			BelumSampai bool       `json:"belum_sampai"`
			BelumNote   string     `json:"belum_sampai_note"`
			Pencuci     string     `json:"pencuci"`
			CuciTgl     *time.Time `json:"cuci_tgl"`
			TujuanID    int        `json:"tujuan_id"`
			Tujuan      string     `json:"tujuan"`
		}
		out := []row{}
		var totalGram, totalBeli float64
		for rows.Next() {
			var r row
			var kirim, cuciTgl sql.NullTime
			if err := rows.Scan(&r.ID, &r.ItemNo, &r.ItemName, &r.Karat, &r.WeightGram,
				&r.Jenis, &r.Brand, &r.Kondisi, &r.HargaBeli,
				&r.LocationID, &r.Lokasi, &r.OriginID, &r.Asal, &r.Status, &r.EntryDate, &r.Note,
				&kirim, &r.BelumSampai, &r.BelumNote, &r.Pencuci, &cuciTgl, &r.TujuanID, &r.Tujuan); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if kirim.Valid {
				t := kirim.Time
				r.KirimTgl = &t
			}
			if cuciTgl.Valid {
				t := cuciTgl.Time
				r.CuciTgl = &t
			}
			totalGram += r.WeightGram
			totalBeli += r.HargaBeli
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "jumlah": len(out), "total_gram": totalGram, "total_beli": totalBeli})
	}
}