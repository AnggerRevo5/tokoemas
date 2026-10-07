package main

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var opMasuk = map[string]bool{"omzet": true, "jual_barang": true, "jual_aset": true}
var opKeluar = map[string]bool{"pengeluaran_toko": true, "pembelian_customer": true, "operasional": true, "bulanan_investor": true}
var danaNonOp = map[string]bool{"investor": true, "pinjaman": true, "bayar_hutang": true}

func labaRugiHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := hanyaAdminOwner(c); !ok {
			return
		}

		wib, _ := time.LoadLocation("Asia/Jakarta")
		now := time.Now().In(wib)
		from := c.DefaultQuery("from", time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, wib).Format("2006-01-02"))
		to := c.DefaultQuery("to", now.Format("2006-01-02"))
		for _, s := range []string{from, to} {
			if _, err := time.Parse("2006-01-02", s); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "format tanggal harus YYYY-MM-DD"})
				return
			}
		}

		rows, err := db.Query(`
			SELECT category, SUM(amount) FROM cash_transactions
			WHERE is_void = FALSE AND transaction_date BETWEEN ? AND ?
			GROUP BY category`, from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		masuk := map[string]float64{}
		keluar := map[string]float64{}
		dana := map[string]float64{}
		var totalMasuk, totalKeluar float64

		for rows.Next() {
			var kat string
			var jml float64
			if err := rows.Scan(&kat, &jml); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			switch {
			case opMasuk[kat]:
				masuk[kat] = jml
				totalMasuk += jml
			case opKeluar[kat]:
				keluar[kat] = jml
				totalKeluar += jml
			case danaNonOp[kat]:
				dana[kat] = jml
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"dari":                 from,
			"sampai":               to,
			"uang_masuk":           gin.H{"rincian": masuk, "total": totalMasuk},
			"uang_keluar":          gin.H{"rincian": keluar, "total": totalKeluar},
			"laba_rugi":            totalMasuk - totalKeluar,
			"dana_non_operasional": dana,
		})
	}
}