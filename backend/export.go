package main

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func labelArah(a string) string {
	if a == "in" {
		return "Masuk"
	}
	return "Keluar"
}

func exportHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)

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
				tgl.Format("2006-01-02"), lok, labelArah(arah), strings.ReplaceAll(kat, "_", " "), amt, note, user,
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
				tgl.Format("2006-01-02"), lok, labelArah(arah), strings.ReplaceAll(kat, "_", " "), item, berat, karat, qty, note, user,
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