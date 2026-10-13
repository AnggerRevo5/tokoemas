package main

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func serviceRingkas(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)

		locRows, err := db.Query("SELECT id, name, type FROM locations ORDER BY id")
		if err != nil {
			gagal(c, err)
			return
		}
		defer locRows.Close()

		type ring struct {
			LocationID     int    `json:"location_id"`
			Name           string `json:"name"`
			Type           string `json:"type"`
			DiSini         int    `json:"di_sini"`
			DiToko         int    `json:"di_toko"`
			MenungguServis int    `json:"menunggu_servis"`
			Diproses       int    `json:"diproses"`
			Selesai        int    `json:"selesai"`
			SiapDiambil    int    `json:"siap_diambil"`
			Keluar         int    `json:"keluar"`
			Masuk          int    `json:"masuk"`
		}
		m := map[int]*ring{}
		urut := []int{}
		boID := 0
		for locRows.Next() {
			var r ring
			if err := locRows.Scan(&r.LocationID, &r.Name, &r.Type); err != nil {
				gagal(c, err)
				return
			}
			if r.Type == "back_office" {
				boID = r.LocationID
			}
			if cl.Role == "cabang" && r.LocationID != cl.LocationID {
				continue
			}
			rr := r
			m[r.LocationID] = &rr
			urut = append(urut, r.LocationID)
		}

		// posisi barang service saat ini
		stRows, err := db.Query(`SELECT location_id, status, COUNT(*) FROM service_jobs
		                         WHERE is_void = FALSE AND status <> 'diambil' GROUP BY location_id, status`)
		if err != nil {
			gagal(c, err)
			return
		}
		for stRows.Next() {
			var loc, n int
			var st string
			if err := stRows.Scan(&loc, &st, &n); err != nil {
				stRows.Close()
				gagal(c, err)
				return
			}
			r := m[loc]
			if r == nil {
				continue
			}
			switch st {
			case "di_toko":
				r.DiToko = n
			case "di_bo":
				r.MenungguServis = n
			case "diproses":
				r.Diproses = n
			case "selesai":
				r.Selesai = n
			case "siap_diambil":
				r.SiapDiambil = n
			case "dikirim_bo", "dikirim_toko":
				r.Keluar += n
			}
		}
		stRows.Close()

		// barang yang sedang menuju lokasi
		if r := m[boID]; r != nil {
			var n int
			db.QueryRow("SELECT COUNT(*) FROM service_jobs WHERE is_void = FALSE AND status = 'dikirim_bo'").Scan(&n)
			r.Masuk += n
		}
		inRows, err := db.Query(`SELECT pickup_location_id, COUNT(*) FROM service_jobs
		                         WHERE is_void = FALSE AND status = 'dikirim_toko' AND pickup_location_id IS NOT NULL
		                         GROUP BY pickup_location_id`)
		if err == nil {
			for inRows.Next() {
				var loc, n int
				if inRows.Scan(&loc, &n) == nil {
					if r := m[loc]; r != nil {
						r.Masuk += n
					}
				}
			}
			inRows.Close()
		}

		out := []*ring{}
		for _, id := range urut {
			r := m[id]
			r.DiSini = r.DiToko + r.MenungguServis + r.Diproses + r.Selesai + r.SiapDiambil
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}