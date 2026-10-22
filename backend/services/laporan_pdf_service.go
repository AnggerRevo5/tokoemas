package services

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/go-pdf/fpdf"

	"toko-emas/models"
)

func FmtRp(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := strconv.FormatInt(int64(math.Round(v)), 10)
	out := []byte{}
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, s[i])
	}
	r := "Rp " + string(out)
	if neg {
		r = "-" + r
	}
	return r
}

func teksSel(v interface{}, tipe string) string {
	switch x := v.(type) {
	case nil:
		return ""
	case float64:
		switch tipe {
		case "rupiah":
			return FmtRp(x)
		case "gram":
			return strconv.FormatFloat(math.Round(x*1000)/1000, 'f', -1, 64)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}

func gambarTabel(pdf *fpdf.Fpdf, tr func(string) string, t models.TabelLaporan) {
	const lebarTotal = 277.0
	const x0 = 10.0
	n := len(t.Kolom)
	ukuran, lh := 8.0, 4.0
	if n > 9 {
		ukuran, lh = 6.5, 3.2
	}

	bobot := make([]float64, n)
	for j, k := range t.Kolom {
		bobot[j] = float64(len([]rune(k)))
	}
	for _, b := range t.Baris {
		for j, v := range b {
			if j < n {
				if l := float64(len([]rune(teksSel(v, t.Tipe[j])))); l > bobot[j] {
					bobot[j] = l
				}
			}
		}
	}
	var jumlah float64
	for j := range bobot {
		if bobot[j] > 38 {
			bobot[j] = 38
		}
		if bobot[j] < 7 {
			bobot[j] = 7
		}
		jumlah += bobot[j]
	}
	lebar := make([]float64, n)
	for j := range bobot {
		lebar[j] = bobot[j] / jumlah * lebarTotal
	}

	tebal := map[int]bool{}
	for _, i := range t.Tebal {
		tebal[i] = true
	}

	var gambar func(sel []string, mode string)
	gambar = func(sel []string, mode string) {
		style := "B"
		if mode == "biasa" {
			style = ""
		}
		pdf.SetFont("Helvetica", style, ukuran)
		maks := 1
		for j, s := range sel {
			if l := len(pdf.SplitText(tr(s), lebar[j]-2)); l > maks {
				maks = l
			}
		}
		h := float64(maks)*lh + 1.5

		_, ph := pdf.GetPageSize()
		if pdf.GetY()+h > ph-14 {
			pdf.AddPage()
			if mode != "head" {
				gambar(t.Kolom, "head")
				pdf.SetFont("Helvetica", style, ukuran)
			}
		}

		y := pdf.GetY()
		pdf.SetDrawColor(180, 180, 180)
		gaya := "D"
		switch mode {
		case "head":
			pdf.SetFillColor(31, 78, 61)
			pdf.SetTextColor(255, 255, 255)
			gaya = "FD"
		case "tebal":
			pdf.SetFillColor(243, 244, 241)
			pdf.SetTextColor(0, 0, 0)
			gaya = "FD"
		default:
			pdf.SetTextColor(0, 0, 0)
		}

		x := x0
		for j, s := range sel {
			pdf.Rect(x, y, lebar[j], h, gaya)
			pdf.SetXY(x, y+0.75)
			align := "L"
			if t.Tipe[j] != "teks" {
				align = "R"
			}
			pdf.MultiCell(lebar[j], lh, tr(s), "", align, false)
			x += lebar[j]
		}
		pdf.SetXY(x0, y+h)
		pdf.SetTextColor(0, 0, 0)
	}

	gambar(t.Kolom, "head")
	for i, b := range t.Baris {
		sel := make([]string, n)
		for j := 0; j < n && j < len(b); j++ {
			sel[j] = teksSel(b[j], t.Tipe[j])
		}
		mode := "biasa"
		if tebal[i] {
			mode = "tebal"
		}
		gambar(sel, mode)
	}
}

func BuatPDF(dari, sampai, user string, hasil []models.TabelLaporan) *fpdf.Fpdf {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(10, 12, 10)
	pdf.SetAutoPageBreak(true, 12)
	pdf.SetTitle("Laporan Pembukuan Toko Emas", false)
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	pdf.SetFooterFunc(func() {
		pdf.SetY(-10)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(0, 5, tr("Halaman "+strconv.Itoa(pdf.PageNo())+" dari {nb}"), "", 0, "C", false, 0, "")
	})
	pdf.AliasNbPages("")

	wib, _ := time.LoadLocation("Asia/Jakarta")
	dicetak := time.Now().In(wib).Format("2006-01-02 15:04")

	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 15)
	pdf.CellFormat(0, 8, tr("Pembukuan Toko Emas - Bulan Purnama"), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(90, 90, 90)
	pdf.CellFormat(0, 5, tr(fmt.Sprintf("Periode %s sampai %s. Dicetak oleh %s pada %s.", dari, sampai, user, dicetak)), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(4)

	for i, t := range hasil {
		if i > 0 {
			pdf.AddPage()
		}
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(0, 7, tr(t.Judul), "", 1, "L", false, 0, "")
		gambarTabel(pdf, tr, t)
	}
	return pdf
}
