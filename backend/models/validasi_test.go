package models

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// bind meniru handler: body JSON dibaca dengan ShouldBindJSON (validasi tag binding ikut berjalan).
func bind(t *testing.T, body string, v interface{}) error {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c.ShouldBindJSON(v)
}

func TestValidasiInputNormalDiterima(t *testing.T) {
	var kas KasReq
	if err := bind(t, `{"location_id":1,"direction":"in","category":"omzet","amount":1500000,"note":"","transaction_date":"2026-10-26"}`, &kas); err != nil {
		t.Fatalf("input kas normal ditolak: %v", err)
	}
	var barang BarangMasukReq
	if err := bind(t, `{"location_id":2,"category":"beli_customer","entry_date":"2026-10-26","posisi":"toko",
		"items":[{"jenis":"perhiasan","item_name":"Cincin","karat":"22K","weight_gram":3.25,"harga_beli":2500000}]}`, &barang); err != nil {
		t.Fatalf("input barang normal ditolak: %v", err)
	}
	var svc ServiceAksiRequest
	if err := bind(t, `{"action":"diambil","ids":[3],"fees":{"3":75000}}`, &svc); err != nil {
		t.Fatalf("aksi service normal ditolak: %v", err)
	}
}

func TestValidasiMenolakNilaiTidakMasukAkal(t *testing.T) {
	kasus := []struct {
		nama string
		body string
		v    interface{}
	}{
		{"nominal kas di atas 1 triliun", `{"amount":5000000000000}`, &KasReq{}},
		{"catatan terlalu panjang", `{"note":"` + strings.Repeat("a", 201) + `"}`, &KasReq{}},
		{"berat barang 1 ton", `{"items":[{"weight_gram":1000000}]}`, &BarangMasukReq{}},
		{"nama barang terlalu panjang", `{"items":[{"item_name":"` + strings.Repeat("b", 101) + `"}]}`, &BarangMasukReq{}},
		{"harga jual raksasa", `{"items":[{"item_id":1,"harga":1e15}]}`, &JualReq{}},
		{"id barang negatif", `{"item_ids":[-5]}`, &KirimBOReq{}},
		{"biaya service negatif", `{"ids":[1],"fees":{"1":-1000}}`, &ServiceAksiRequest{}},
		{"qty negatif", `{"qty":-3}`, &StokReq{}},
		{"username terlalu panjang", `{"username":"` + strings.Repeat("u", 51) + `","password":"x"}`, &LoginRequest{}},
	}
	for _, k := range kasus {
		if err := bind(t, k.body, k.v); err == nil {
			t.Errorf("%s: seharusnya ditolak", k.nama)
		}
	}
}
