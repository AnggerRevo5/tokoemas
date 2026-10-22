// File ini berisi kode lama yang sudah tidak terdaftar di routes.
// Dibiarkan dulu sampai ada konfirmasi boleh dihapus.

package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"

	"toko-emas/models"
	"toko-emas/services"
)

func LegacyExportHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)

		from := c.DefaultQuery("from", "2000-01-01")
		to := c.DefaultQuery("to", "2999-12-31")
		for _, s := range []string{from, to} {
			if _, err := time.Parse("2006-01-02", s); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "format tanggal harus YYYY-MM-DD"})
				return
			}
		}

		filter := ""
		args := []interface{}{from, to}
		if cl.Role == "cabang" {
			filter = " AND t.location_id = ?"
			args = append(args, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" && loc != "0" {
			filter = " AND t.location_id = ?"
			args = append(args, loc)
		}

		f := excelize.NewFile()
		bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})

		// ---------- Sheet Kas ----------
		f.SetSheetName("Sheet1", "Kas")
		f.SetSheetRow("Kas", "A1", &[]interface{}{"Tanggal", "Lokasi", "Arah", "Kategori", "Nominal (Rp)", "Catatan", "Diinput oleh"})
		f.SetCellStyle("Kas", "A1", "G1", bold)
		f.SetColWidth("Kas", "A", "G", 20)

		rows, err := db.Query(`
			SELECT t.transaction_date, l.name, t.direction, t.category, t.amount, IFNULL(t.note,''), u.username
			FROM cash_transactions t
			JOIN locations l ON l.id = t.location_id
			JOIN users u ON u.id = t.created_by
			WHERE t.is_void = FALSE AND t.transaction_date BETWEEN ? AND ?`+filter+`
			ORDER BY t.transaction_date, t.id`, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		n := 2
		for rows.Next() {
			var tgl time.Time
			var lok, arah, kat, note, user string
			var amt float64
			if err := rows.Scan(&tgl, &lok, &arah, &kat, &amt, &note, &user); err != nil {
				rows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			cell, _ := excelize.CoordinatesToCellName(1, n)
			f.SetSheetRow("Kas", cell, &[]interface{}{
				tgl.Format("2006-01-02"), lok, services.LabelArah(arah), strings.ReplaceAll(kat, "_", " "), amt, note, user,
			})
			n++
		}
		rows.Close()

		// ---------- Sheet Stok ----------
		f.NewSheet("Stok")
		f.SetSheetRow("Stok", "A1", &[]interface{}{"Tanggal", "Lokasi", "Arah", "Kategori", "Barang", "Berat (gram)", "Kadar", "Jumlah (pcs)", "Catatan", "Diinput oleh"})
		f.SetCellStyle("Stok", "A1", "J1", bold)
		f.SetColWidth("Stok", "A", "J", 20)

		rows2, err := db.Query(`
			SELECT t.transaction_date, l.name, t.direction, t.category, t.item_name, t.weight_gram,
			       IFNULL(t.karat,''), t.qty, IFNULL(t.note,''), u.username
			FROM stock_movements t
			JOIN locations l ON l.id = t.location_id
			JOIN users u ON u.id = t.created_by
			WHERE t.is_void = FALSE AND t.transaction_date BETWEEN ? AND ?`+filter+`
			ORDER BY t.transaction_date, t.id`, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		n = 2
		for rows2.Next() {
			var tgl time.Time
			var lok, arah, kat, item, karat, note, user string
			var berat float64
			var qty int
			if err := rows2.Scan(&tgl, &lok, &arah, &kat, &item, &berat, &karat, &qty, &note, &user); err != nil {
				rows2.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			cell, _ := excelize.CoordinatesToCellName(1, n)
			f.SetSheetRow("Stok", cell, &[]interface{}{
				tgl.Format("2006-01-02"), lok, services.LabelArah(arah), strings.ReplaceAll(kat, "_", " "), item, berat, karat, qty, note, user,
			})
			n++
		}
		rows2.Close()

		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", `attachment; filename="pembukuan.xlsx"`)
		if err := f.Write(c.Writer); err != nil {
			c.Status(http.StatusInternalServerError)
		}
	}
}

func namaSheet(s string) string {
	for _, ch := range []string{":", "\\", "/", "?", "*", "[", "]"} {
		s = strings.ReplaceAll(s, ch, " ")
	}
	r := []rune(strings.TrimSpace(s))
	if len(r) > 31 {
		r = r[:31]
	}
	return string(r)
}

func gayaExcel(f *excelize.File, cache map[string]int, tipe string, tebal bool) int {
	key := fmt.Sprintf("%s|%v", tipe, tebal)
	if id, ok := cache[key]; ok {
		return id
	}
	st := &excelize.Style{
		Font:      &excelize.Font{Bold: tebal},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
	}
	switch tipe {
	case "rupiah":
		st.NumFmt = 3
	case "gram":
		s := "#,##0.000"
		st.CustomNumFmt = &s
	}
	id, _ := f.NewStyle(st)
	cache[key] = id
	return id
}

func LegacyLaporanExcel(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		dari, sampai, hasil, ok := siapkanLaporan(c, db)
		if !ok {
			return
		}

		f := excelize.NewFile()
		judulGaya, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
		headGaya, _ := f.NewStyle(&excelize.Style{
			Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
			Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1F4E3D"}, Pattern: 1},
			Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		})
		cache := map[string]int{}

		for idx, t := range hasil {
			nama := namaSheet(t.Judul)
			if idx == 0 {
				f.SetSheetName("Sheet1", nama)
			} else {
				f.NewSheet(nama)
			}
			f.SetCellValue(nama, "A1", fmt.Sprintf("%s (%s s/d %s)", t.Judul, dari, sampai))
			f.SetCellStyle(nama, "A1", "A1", judulGaya)

			lebar := make([]int, len(t.Kolom))
			for j, k := range t.Kolom {
				cell, _ := excelize.CoordinatesToCellName(j+1, 3)
				f.SetCellValue(nama, cell, k)
				f.SetCellStyle(nama, cell, cell, headGaya)
				lebar[j] = len([]rune(k))
			}

			tebal := map[int]bool{}
			for _, i := range t.Tebal {
				tebal[i] = true
			}
			for i, b := range t.Baris {
				for j, v := range b {
					if v == nil || j >= len(t.Tipe) {
						continue
					}
					cell, _ := excelize.CoordinatesToCellName(j+1, 4+i)
					f.SetCellValue(nama, cell, v)
					f.SetCellStyle(nama, cell, cell, gayaExcel(f, cache, t.Tipe[j], tebal[i]))
					if n := len([]rune(fmt.Sprint(v))); n > lebar[j] {
						lebar[j] = n
					}
				}
			}
			for j, w := range lebar {
				if w > 45 {
					w = 45
				}
				if w < 10 {
					w = 10
				}
				col, _ := excelize.ColumnNumberToName(j + 1)
				f.SetColWidth(nama, col, col, float64(w+2))
			}
		}

		c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="laporan_%s_sd_%s.xlsx"`, dari, sampai))
		if err := f.Write(c.Writer); err != nil {
			c.Status(http.StatusInternalServerError)
		}
	}
}
