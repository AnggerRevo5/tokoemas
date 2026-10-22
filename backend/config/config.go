package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// JWTSecret diambil dari environment JWT_SECRET (wajib di production).
var JWTSecret = func() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		if os.Getenv("APP_ENV") == "production" {
			log.Fatal("JWT_SECRET wajib diisi di production")
		}
		s = "kunci-khusus-pengembangan-lokal"
	}
	return []byte(s)
}()

// OpenDB membuka koneksi MySQL memakai DB_DSN (default khusus pengembangan di laptop).
func OpenDB() (*sql.DB, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		if os.Getenv("APP_ENV") == "production" {
			log.Fatal("DB_DSN wajib diisi di production")
		}
		// khusus pengembangan di laptop
		dsn = "root:rahasia123@tcp(127.0.0.1:3306)/toko_emas?parseTime=true&loc=Asia%2FJakarta&time_zone=%27%2B07%3A00%27"
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
