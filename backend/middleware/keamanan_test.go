package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"toko-emas/config"
	"toko-emas/models"
	"toko-emas/services"
)

func init() { gin.SetMode(gin.TestMode) }

func klaimUji(sesi string) models.Claims {
	return models.Claims{
		UserID: 7, Username: "dinoyo", Role: "cabang", LocationID: 2,
		RegisteredClaims: jwt.RegisteredClaims{
			ID: sesi, Issuer: penerbitToken, Audience: jwt.ClaimStrings{audiensToken},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
}

func TestBacaTokenMenerimaTokenSah(t *testing.T) {
	tok, err := BuatToken(7, "dinoyo", "cabang", 2, "abc123")
	if err != nil {
		t.Fatal(err)
	}
	cl, ok := BacaToken(tok)
	if !ok || cl.UserID != 7 || cl.ID != "abc123" {
		t.Fatalf("token sah ditolak: %+v", cl)
	}
}

func TestBacaTokenMenolakAlgoritmaNone(t *testing.T) {
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodNone, klaimUji("x")).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, ok := BacaToken(tok); ok {
		t.Fatal("token alg=none harus ditolak")
	}
}

func TestBacaTokenMenolakAlgoritmaLain(t *testing.T) {
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, klaimUji("x")).SignedString(config.JWTSecret)
	if _, ok := BacaToken(tok); ok {
		t.Fatal("token HS512 harus ditolak (hanya HS256)")
	}
}

func TestBacaTokenMenolakSecretLainDanTanpaSesi(t *testing.T) {
	palsu, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, klaimUji("x")).SignedString([]byte("secret-penyerang-yang-bukan-milik-server"))
	if _, ok := BacaToken(palsu); ok {
		t.Fatal("token dengan secret lain harus ditolak")
	}
	tanpaSesi, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, klaimUji("")).SignedString(config.JWTSecret)
	if _, ok := BacaToken(tanpaSesi); ok {
		t.Fatal("token lama tanpa ID sesi harus ditolak")
	}
}

func TestBacaTokenMenolakKedaluwarsaDanAudiensLain(t *testing.T) {
	kl := klaimUji("x")
	kl.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour))
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, kl).SignedString(config.JWTSecret)
	if _, ok := BacaToken(tok); ok {
		t.Fatal("token kedaluwarsa harus ditolak")
	}
	kl = klaimUji("x")
	kl.Audience = jwt.ClaimStrings{"aplikasi-lain"}
	tok, _ = jwt.NewWithClaims(jwt.SigningMethodHS256, kl).SignedString(config.JWTSecret)
	if _, ok := BacaToken(tok); ok {
		t.Fatal("token untuk audiens lain harus ditolak")
	}
}

func TestCSRFTerikatKeSesi(t *testing.T) {
	if !services.CSRFCocok("sesi-a", services.TokenCSRF("sesi-a")) {
		t.Fatal("token CSRF sesi sendiri harus diterima")
	}
	if services.CSRFCocok("sesi-b", services.TokenCSRF("sesi-a")) || services.CSRFCocok("sesi-a", "") {
		t.Fatal("token CSRF sesi lain atau kosong harus ditolak")
	}
}

func jalankanPJ(path, pola string) int {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("claims", &models.Claims{Role: "pj"}) }, BatasiPJ())
	r.GET(pola, func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code
}

func TestBatasiPJ(t *testing.T) {
	boleh := map[string]string{
		"/me": "/me", "/locations": "/locations", "/barang/notif": "/barang/notif", "/cuci": "/cuci",
		"/cucian/barang": "/cucian/barang", "/lebur": "/lebur", "/penjualan/siap": "/penjualan/siap", "/service": "/service",
	}
	for path, pola := range boleh {
		if kode := jalankanPJ(path, pola); kode != http.StatusOK {
			t.Errorf("PJ seharusnya boleh %s, dapat %d", path, kode)
		}
	}
	dilarang := []string{"/kas", "/stok", "/saldo", "/barang", "/hutang", "/akun", "/laporan/preview", "/laporan/labarugi"}
	for _, p := range dilarang {
		if kode := jalankanPJ(p, p); kode != http.StatusForbidden {
			t.Errorf("PJ seharusnya ditolak di %s, dapat %d", p, kode)
		}
	}
}

func TestKunciLoginPerAkunDanIP(t *testing.T) {
	p := pembatasBaru(time.Minute, 5, time.Minute)
	for i := 0; i < 5; i++ {
		if p.tambah("dinoyo|1.2.3.4") {
			t.Fatalf("belum boleh terkunci pada percobaan ke-%d", i+1)
		}
	}
	if !p.tambah("dinoyo|1.2.3.4") || p.terkunci("dinoyo|1.2.3.4") == 0 {
		t.Fatal("harus terkunci setelah melewati batas")
	}
	if p.terkunci("dinoyo|5.6.7.8") != 0 {
		t.Fatal("IP lain tidak boleh ikut terkunci (mencegah penguncian akun toko dari luar)")
	}
	p.reset("dinoyo|1.2.3.4")
	if p.terkunci("dinoyo|1.2.3.4") != 0 {
		t.Fatal("login berhasil harus menghapus kunci")
	}
}

func TestBatasBody(t *testing.T) {
	r := gin.New()
	r.Use(BatasBody())
	r.POST("/x", func(c *gin.Context) {
		var v map[string]interface{}
		if err := c.ShouldBindJSON(&v); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		c.Status(http.StatusOK)
	})
	besar := `{"a":"` + strings.Repeat("x", maksBody+10) + `"}`
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", bytes.NewBufferString(besar)))
	if w.Code == http.StatusOK {
		t.Fatal("body lebih dari 1 MB harus ditolak")
	}
}

func TestHeaderKeamanan(t *testing.T) {
	r := gin.New()
	r.Use(HeaderKeamanan())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	for _, h := range []string{"X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy", "Cache-Control"} {
		if w.Header().Get(h) == "" {
			t.Errorf("header %s belum dipasang", h)
		}
	}
}
