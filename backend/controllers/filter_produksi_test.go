package controllers

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"toko-emas/utils"
)

func TestSyaratBarang(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?q=cincin&karat=17&berat_min=1.5&asal=2&berat_max=abc", nil)

	sy, args := syaratBarang(c, "i", "i.weight_gram", "i.origin_location_id", true)
	// nama (2 argumen), kadar, berat_min, asal; berat_max tidak valid diabaikan
	if len(args) != 5 || args[2] != "17K" {
		t.Fatalf("argumen salah: %v (%s)", args, sy)
	}

	c, _ = gin.CreateTestContext(httptest.NewRecorder()) // konteks baru: gin menyimpan cache query
	c.Request = httptest.NewRequest("GET", "/?q=cincin", nil)
	if sy, args := syaratBarang(c, "s", "s.weight_gram", "s.origin_location_id", false); sy != "" || len(args) != 0 {
		t.Fatalf("q harus diabaikan bila denganNama=false: %s %v", sy, args)
	}
}

func TestSyaratTanggalBarangMasuk(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?from=2026-01-01&to=2026-01-31", nil)
	if !utils.TanggalOpsional(c, "from", "to") {
		t.Fatal("tanggal masuk valid seharusnya diterima")
	}

	syarat, args := syaratTanggal(c, "i.entry_date")
	if syarat != " AND DATE(i.entry_date) >= ? AND DATE(i.entry_date) <= ?" {
		t.Fatalf("syarat tanggal masuk salah: %s", syarat)
	}
	if len(args) != 2 || args[0] != "2026-01-01" || args[1] != "2026-01-31" {
		t.Fatalf("argumen rentang tanggal masuk salah: %v", args)
	}
}

func TestSyaratCariStok(t *testing.T) {
	// "8k" ikut dicocokkan ke kadar (5 argumen per kata + 1 kadar), "mdl" dan "mp" hanya teks
	sy, args := syaratCariStok("cincin mdl mp 8k")
	if len(args) != 4*5+1 || args[len(args)-1] != "8K" || strings.Count(sy, "i.karat = ?") != 1 {
		t.Fatalf("argumen salah: %v (%s)", args, sy)
	}
	// angka tanpa K bukan kadar (agar "8" tidak ikut menemukan 18K)
	if _, args := syaratCariStok("cincin 8"); len(args) != 2*5 {
		t.Fatalf("angka tanpa K tidak boleh jadi kadar: %v", args)
	}
	if sy, args := syaratCariStok("   "); sy != "" || len(args) != 0 {
		t.Fatalf("teks kosong harus diabaikan: %s %v", sy, args)
	}
}

func TestSyaratKadarBerat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?karat=999&berat_min=1&berat_max=2.5", nil)
	sy, args := syaratKadarBerat(c, "i", "i.weight_gram")
	if len(args) != 3 || args[0] != "999" || args[1] != 1.0 || args[2] != 2.5 {
		t.Fatalf("argumen salah: %v (%s)", args, sy)
	}
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?karat=99", nil) // bukan kadar yang dikenal: diabaikan
	if sy, args := syaratKadarBerat(c, "i", "i.weight_gram"); sy != "" || len(args) != 0 {
		t.Fatalf("kadar tidak dikenal harus diabaikan: %s %v", sy, args)
	}
	c, _ = gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?karat=17K&karat=18&karat=17", nil)
	if sy, args := syaratKadarBerat(c, "i", "i.weight_gram"); sy != " AND i.karat IN (?,?)" || len(args) != 2 || args[0] != "17K" || args[1] != "18K" {
		t.Fatalf("filter beberapa kadar salah: %v (%s)", args, sy)
	}
}
