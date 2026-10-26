package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// Produksi bernilai true bila APP_ENV=production.
func Produksi() bool { return os.Getenv("APP_ENV") == "production" }

// JWTSecret diambil dari environment JWT_SECRET (wajib di production, minimal 32 karakter).
var JWTSecret = func() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		if Produksi() {
			log.Fatal("JWT_SECRET wajib diisi di production")
		}
		s = "kunci-khusus-pengembangan-lokal-bukan-untuk-produksi"
	}
	if Produksi() && len(s) < 32 {
		log.Fatal("JWT_SECRET minimal 32 karakter di production")
	}
	return []byte(s)
}()

// CookieSecure: cookie sesi hanya dikirim lewat HTTPS. Bawaan: aktif di production.
// Isi COOKIE_SECURE=false hanya untuk pengujian lokal lewat http.
func CookieSecure() bool {
	if v := os.Getenv("COOKIE_SECURE"); v != "" {
		b, _ := strconv.ParseBool(v)
		return b
	}
	return Produksi()
}

// CookiePath adalah path cookie sesi di browser (frontend memanggil API lewat /api).
func CookiePath() string {
	if v := os.Getenv("COOKIE_PATH"); v != "" {
		return v
	}
	return "/api"
}

// SesiJam adalah umur maksimal sesi login (bawaan 12 jam, satu hari kerja).
func SesiJam() int { return angkaEnv("SESI_JAM", 12, 1, 24) }

// SesiIdleMenit: sesi berakhir bila tidak dipakai selama ini (bawaan 240 menit).
func SesiIdleMenit() int { return angkaEnv("SESI_IDLE_MENIT", 240, 5, 1440) }

// TrustedProxies adalah alamat proxy (nginx, cloudflared) yang boleh menentukan IP asli pengguna.
func TrustedProxies() []string {
	v := os.Getenv("TRUSTED_PROXIES")
	if v == "" {
		v = "127.0.0.1/8,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,::1/128"
	}
	out := []string{}
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func angkaEnv(nama string, bawaan, min, maks int) int {
	n, err := strconv.Atoi(os.Getenv(nama))
	if err != nil || n < min || n > maks {
		return bawaan
	}
	return n
}

// OpenDB membuka koneksi MySQL memakai DB_DSN (default khusus pengembangan di laptop, tanpa password).
func OpenDB() (*sql.DB, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		if Produksi() {
			log.Fatal("DB_DSN wajib diisi di production")
		}
		// khusus pengembangan di laptop; isi DB_DSN bila MySQL lokal memakai password
		dsn = "root@tcp(127.0.0.1:3306)/toko_emas?parseTime=true&loc=Asia%2FJakarta&time_zone=%27%2B07%3A00%27"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("gagal konek database: %w", err)
	}
	return db, nil
}
