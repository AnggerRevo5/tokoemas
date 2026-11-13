package controllers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
