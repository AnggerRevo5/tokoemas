package controllers

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"toko-emas/models"
	"toko-emas/services"
)

// mesinTalangan membuat router uji dengan claims palsu; semua rute talangan terdaftar seperti di routes.Setup.
func mesinTalangan(db *sql.DB, role string, userID, lokasi int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("claims", &models.Claims{UserID: userID, Username: "uji", Role: role, LocationID: lokasi})
		c.Next()
	})
	RuteTalangan(r, db)
	return r
}

func panggil(r *gin.Engine, metode, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(metode, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func jsonKe(t *testing.T, w *httptest.ResponseRecorder, v interface{}) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("bukan JSON: %s", w.Body.String())
	}
}

// dbUji membuka database uji (TEST_DB_DSN) dan menyiapkan skema. Dilewati bila DSN tidak diisi.
func dbUji(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN tidak diisi")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	// skema yang sama dengan main.go supaya query rute lama (antrian cuci/lebur, baki, dsb.) bisa jalan
	for _, f := range []func(*sql.DB) error{
		services.SiapkanSkemaKeamanan, services.SiapkanSkemaBaki, services.SiapkanSkemaInvestor, services.SiapkanSkemaMetode,
		services.SiapkanSkemaGadai, services.SiapkanSkemaBahan, services.SiapkanSkemaJenis, services.SiapkanSkemaCuciKirim,
		services.SiapkanSkemaAntrianLebur, services.SiapkanSkemaAntrianCuci, services.SiapkanSkemaKirim, services.SiapkanSkemaBP,
		services.SiapkanSkemaTukarTambah, services.SiapkanSkemaTalanganBO,
	} {
		if err := f(db); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

type akunUji struct{ User, Toko, BO, BPID int }

// akunDariDB mengambil satu user, satu toko (cabang), back office, dan satu BP aktif dari database uji.
func akunDariDB(t *testing.T, db *sql.DB) akunUji {
	t.Helper()
	var a akunUji
	must := func(q string, dest *int) {
		if err := db.QueryRow(q).Scan(dest); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	must("SELECT id FROM users ORDER BY id LIMIT 1", &a.User)
	must("SELECT id FROM locations WHERE type='cabang' ORDER BY id LIMIT 1", &a.Toko)
	must("SELECT id FROM locations WHERE type='back_office' ORDER BY id LIMIT 1", &a.BO)
	must("SELECT id FROM bp_cabang WHERE aktif = TRUE ORDER BY id LIMIT 1", &a.BPID)
	return a
}
