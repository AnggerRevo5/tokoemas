package routes

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/services"
)

func init() { gin.SetMode(gin.TestMode) }

// ---------- driver database palsu ----------
// Hanya menjawab query sesi milik AuthMiddleware; query lain mengembalikan hasil kosong dan Exec selalu berhasil.
// Cukup untuk menguji pembatasan per role sebelum controller pembukuan berjalan.

type sesiPalsu struct {
	mu   sync.Mutex
	role string
}

func (s *sesiPalsu) ambilRole() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.role
}

var sesiUji = &sesiPalsu{}

type driverPalsu struct{}

func (driverPalsu) Open(string) (driver.Conn, error) { return koneksiPalsu{}, nil }

type koneksiPalsu struct{}

func (koneksiPalsu) Prepare(q string) (driver.Stmt, error) { return stmtPalsu{q: q}, nil }
func (koneksiPalsu) Close() error                          { return nil }
func (koneksiPalsu) Begin() (driver.Tx, error)             { return txPalsu{}, nil }

type txPalsu struct{}

func (txPalsu) Commit() error   { return nil }
func (txPalsu) Rollback() error { return nil }

type stmtPalsu struct{ q string }

func (s stmtPalsu) Close() error  { return nil }
func (s stmtPalsu) NumInput() int { return -1 }
func (s stmtPalsu) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(1), nil
}
func (s stmtPalsu) Query([]driver.Value) (driver.Rows, error) {
	if strings.Contains(s.q, "FROM user_sessions s JOIN users u") {
		// role, location_id, is_active, username, sesi berlaku, idle
		return &barisPalsu{
			kolom: []string{"role", "location_id", "is_active", "username", "berlaku", "idle"},
			data:  [][]driver.Value{{sesiUji.ambilRole(), nil, true, "uji", true, false}},
		}, nil
	}
	return &barisPalsu{kolom: []string{"x"}}, nil
}

type barisPalsu struct {
	kolom []string
	data  [][]driver.Value
	i     int
}

func (b *barisPalsu) Columns() []string { return b.kolom }
func (b *barisPalsu) Close() error      { return nil }
func (b *barisPalsu) Next(dest []driver.Value) error {
	if b.i >= len(b.data) {
		return io.EOF
	}
	copy(dest, b.data[b.i])
	b.i++
	return nil
}

func init() { sql.Register("palsu-routes", driverPalsu{}) }

// ---------- uji ----------

const sesiID = "sesiuji"

func mesinUji(t *testing.T, role string) *gin.Engine {
	t.Helper()
	sesiUji.mu.Lock()
	sesiUji.role = role
	sesiUji.mu.Unlock()
	db, err := sql.Open("palsu-routes", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	r := gin.New()
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) { c.AbortWithStatus(http.StatusInternalServerError) }))
	Setup(r, db)
	return r
}

func kirim(t *testing.T, r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	return kirimDenganBody(t, r, method, path, "{}")
}

func kirimDenganBody(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	tok, err := middleware.BuatToken(9, "uji", "apa saja", 0, sesiID)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", services.TokenCSRF(sesiID))
	req.AddCookie(&http.Cookie{Name: middleware.NamaCookieSesi, Value: tok})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOperasiTulisInvestorDanHutangMemerlukanBackOffice(t *testing.T) {
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/investor", `{"name":"Investor Uji","amount":1000,"start_date":"2020-01-01","rate_percent":1,"catat_kas":false}`},
		{http.MethodPost, "/investor/1/gerak", `{"jenis":"setor","amount":1000,"tx_date":"2020-01-01","catat_kas":false}`},
		{http.MethodPost, "/hutang", `{"creditor_type":"perorangan","creditor_name":"Kreditur Uji","principal":1000,"start_date":"2020-01-01"}`},
		{http.MethodPost, "/dana/terima", `{"creditor_type":"perorangan","creditor_name":"Kreditur Uji","principal":1000,"start_date":"2020-01-01"}`},
		{http.MethodPost, "/hutang/1/bayar", `{"amount":1000,"paid_date":"2020-01-01"}`},
	}
	for _, role := range []string{"owner", "super_admin"} {
		r := mesinUji(t, role)
		for _, test := range tests {
			w := kirimDenganBody(t, r, test.method, test.path, test.body)
			if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "lokasi back office tidak ditemukan") {
				t.Errorf("%s %s sebagai %s harus berhenti tanpa lokasi Back Office, dapat %d %s",
					test.method, test.path, role, w.Code, w.Body.String())
			}
		}
	}
}

// Pembatasan per role: role biasa diteruskan, PJ tetap dibatasi ke menu Lebur, Cuci, dan Service.
func TestPembatasanRole(t *testing.T) {
	for _, role := range []string{"super_admin", "owner", "cabang"} {
		r := mesinUji(t, role)
		// endpoint daftar lokasi tidak dibatasi role; dengan database palsu hasilnya daftar kosong
		if w := kirim(t, r, http.MethodGet, "/locations"); w.Code != http.StatusOK {
			t.Errorf("%s: /locations harus 200, dapat %d %s", role, w.Code, w.Body.String())
		}
		if w := kirim(t, r, http.MethodGet, "/me"); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"role":"`+role+`"`) {
			t.Errorf("%s: /me harus 200, dapat %d %s", role, w.Code, w.Body.String())
		}
	}
	// PJ tetap dibatasi ke menu produksi
	r := mesinUji(t, "pj")
	if w := kirim(t, r, http.MethodGet, "/kas"); w.Code != http.StatusForbidden {
		t.Errorf("pj: /kas harus tetap 403, dapat %d", w.Code)
	}
	if w := kirim(t, r, http.MethodGet, "/gadai"); w.Code != http.StatusForbidden {
		t.Errorf("pj: /gadai harus 403, dapat %d", w.Code)
	}
	if w := kirim(t, r, http.MethodGet, "/locations"); w.Code != http.StatusOK {
		t.Errorf("pj: /locations harus tetap 200, dapat %d", w.Code)
	}
	// menu Talangan terbuka untuk PJ, tapi kas dan stok tetap tertutup
	if w := kirim(t, r, http.MethodGet, "/talangan/bp"); w.Code != http.StatusOK {
		t.Errorf("pj: /talangan/bp harus 200, dapat %d", w.Code)
	}
	if w := kirim(t, r, http.MethodGet, "/stok"); w.Code != http.StatusForbidden {
		t.Errorf("pj: /stok harus tetap 403, dapat %d", w.Code)
	}
}

func TestHanyaSuperAdminBisaMengubahAtauMembatalkanDataKasStokLama(t *testing.T) {
	ruteUbahLama := [][2]string{
		{http.MethodPost, "/kas/1/void"},
		{http.MethodPost, "/stok/1/void"},
		{http.MethodPut, "/barang/1"},
		{http.MethodPost, "/barang/void/1"},
		{http.MethodPost, "/barang/batal-kirim"},
		{http.MethodPost, "/jual-toko/1/batal"},
		{http.MethodPost, "/jual-toko/1/koreksi"},
		{http.MethodPost, "/gadai/1/batal"},
		{http.MethodPost, "/gadai-bayar/1/void"},
		{http.MethodPost, "/barang/tukar-tambah/1/batal"},
		{http.MethodPost, "/baki/1/nama"},
		{http.MethodPost, "/baki/1/hapus"},
		{http.MethodPost, "/baki/pindah"},
	}
	for _, role := range []string{"cabang", "owner"} {
		r := mesinUji(t, role)
		for _, route := range ruteUbahLama {
			if w := kirim(t, r, route[0], route[1]); w.Code != http.StatusForbidden {
				t.Errorf("%s: %s %s harus 403, dapat %d %s", role, route[0], route[1], w.Code, w.Body.String())
			}
		}
		for _, route := range [][2]string{
			{http.MethodPost, "/kas"},
			{http.MethodPost, "/stok"},
			{http.MethodPost, "/barang"},
		} {
			if w := kirim(t, r, route[0], route[1]); w.Code == http.StatusForbidden {
				t.Errorf("%s: input baru %s %s tetap harus diizinkan, dapat %d %s", role, route[0], route[1], w.Code, w.Body.String())
			}
		}
	}
}

// Rute talangan lewat router asli: wajib login + CSRF, dan tiap role hanya mencapai rute miliknya.
func TestRuteTalanganAuthDanRole(t *testing.T) {
	// tanpa cookie sesi: ditolak
	r := mesinUji(t, "pj")
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/talangan", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized && w.Code != http.StatusForbidden {
		t.Errorf("tanpa login: kode %d, seharusnya 401/403", w.Code)
	}
	// POST tanpa token CSRF: ditolak
	tok, _ := middleware.BuatToken(9, "uji", "apa saja", 0, sesiID)
	req = httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/talangan/tukar", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: middleware.NamaCookieSesi, Value: tok})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("tanpa CSRF: kode %d, seharusnya 403", w.Code)
	}
	// PJ tidak boleh input talangan masuk, penukaran bukan milik cabang/owner
	if w := kirim(t, r, http.MethodPost, "/talangan"); w.Code != http.StatusForbidden {
		t.Errorf("pj POST /talangan: kode %d, seharusnya 403", w.Code)
	}
	for _, role := range []string{"cabang", "owner"} {
		rr := mesinUji(t, role)
		if w := kirim(t, rr, http.MethodPost, "/talangan/tukar"); w.Code != http.StatusForbidden {
			t.Errorf("%s POST /talangan/tukar: kode %d, seharusnya 403", role, w.Code)
		}
	}
	if w := kirim(t, mesinUji(t, "cabang"), http.MethodGet, "/talangan/piutang"); w.Code != http.StatusForbidden {
		t.Errorf("cabang GET /talangan/piutang: kode %d, seharusnya 403", w.Code)
	}
	if w := kirim(t, mesinUji(t, "cabang"), http.MethodGet, "/talangan/tukar"); w.Code != http.StatusForbidden {
		t.Errorf("cabang GET /talangan/tukar: kode %d, seharusnya 403", w.Code)
	}
	// owner tidak boleh membuat/mengubah cabang BP
	if w := kirim(t, mesinUji(t, "owner"), http.MethodPost, "/talangan/bp"); w.Code != http.StatusForbidden {
		t.Errorf("owner POST /talangan/bp: kode %d, seharusnya 403", w.Code)
	}
}
