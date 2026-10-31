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
	t.Helper()
	tok, err := middleware.BuatToken(9, "uji", "apa saja", 0, sesiID)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", services.TokenCSRF(sesiID))
	req.AddCookie(&http.Cookie{Name: middleware.NamaCookieSesi, Value: tok})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
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
	if w := kirim(t, r, http.MethodGet, "/locations"); w.Code != http.StatusOK {
		t.Errorf("pj: /locations harus tetap 200, dapat %d", w.Code)
	}
}
