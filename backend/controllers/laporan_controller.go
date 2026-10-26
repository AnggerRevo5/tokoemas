package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

func LabaRugi(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
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
			utils.Gagal(c, err)
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
				utils.Gagal(c, err)
				return
			}
			switch {
			case services.OpMasuk[kat]:
				masuk[kat] = jml
				totalMasuk += jml
			case services.OpKeluar[kat]:
				keluar[kat] = jml
				totalKeluar += jml
			case services.DanaNonOp[kat]:
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

func siapkanLaporan(c *gin.Context, db *sql.DB) (string, string, []models.TabelLaporan, bool) {
	if _, ok := middleware.HanyaAdminOwner(c); !ok {
		return "", "", nil, false
	}
	wib, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(wib)
	dari := c.DefaultQuery("from", time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, wib).Format("2006-01-02"))
	sampai := c.DefaultQuery("to", now.Format("2006-01-02"))
	for _, s := range []string{dari, sampai} {
		if _, err := time.Parse("2006-01-02", s); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "format tanggal harus YYYY-MM-DD"})
			return "", "", nil, false
		}
	}
	loc := c.Query("location_id")
	if _, err := strconv.Atoi(loc); err != nil || loc == "0" {
		loc = ""
	}

	jenis := strings.Split(c.DefaultQuery("jenis", strings.Join(services.DaftarLaporan, ",")), ",")
	hasil := []models.TabelLaporan{}
	// setiap jenis laporan hanya dibuat sekali, agar satu request tidak bisa memicu ribuan query berat
	sudah := map[string]bool{}
	for _, j := range jenis {
		j = strings.TrimSpace(j)
		if j == "" || sudah[j] {
			continue
		}
		sudah[j] = true
		if len(sudah) > len(services.DaftarLaporan) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "jenis laporan tidak dikenal"})
			return "", "", nil, false
		}
		var t models.TabelLaporan
		var err error
		switch j {
		case "kas":
			t, err = services.TabelKas(db, dari, sampai, loc)
		case "stok":
			t, err = services.TabelStok(db, dari, sampai, loc)
		case "cuci":
			t, err = services.TabelCuci(db, dari, sampai)
		case "lebur":
			t, err = services.TabelLebur(db, dari, sampai)
		case "penjualan":
			t, err = services.TabelPenjualan(db, dari, sampai)
		case "hutang":
			t, err = services.TabelHutang(db)
		case "labarugi":
			t, err = services.TabelLabaRugi(db, dari, sampai)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "jenis laporan tidak dikenal"})
			return "", "", nil, false
		}
		if err != nil {
			utils.Gagal(c, err)
			return "", "", nil, false
		}
		hasil = append(hasil, t)
	}
	if len(hasil) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu laporan"})
		return "", "", nil, false
	}
	return dari, sampai, hasil, true
}

func LaporanPreview(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dari, sampai, hasil, ok := siapkanLaporan(c, db)
		if !ok {
			return
		}
		c.JSON(http.StatusOK, gin.H{"dari": dari, "sampai": sampai, "tabel": hasil})
	}
}

func LaporanPDF(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dari, sampai, hasil, ok := siapkanLaporan(c, db)
		if !ok {
			return
		}
		cl := c.MustGet("claims").(*models.Claims)
		pdf := services.BuatPDF(dari, sampai, cl.Username, hasil)

		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="laporan_%s_sd_%s.pdf"`, dari, sampai))
		if err := pdf.Output(c.Writer); err != nil {
			c.Status(http.StatusInternalServerError)
		}
	}
}
