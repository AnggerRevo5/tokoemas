package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	_ "time/tzdata"
)

type Location struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func main() {
		if len(os.Args) > 2 && os.Args[1] == "hash" {
		cetakHash(os.Args[2])
		return
	}
		dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		if os.Getenv("APP_ENV") == "production" {
			log.Fatal("DB_DSN wajib diisi di production")
		}
		// khusus pengembangan di laptop
		dsn = "root:rahasia123@tcp(127.0.0.1:3306)/toko_emas?parseTime=true&loc=Asia%2FJakarta&time_zone=%27%2B07%3A00%27"
	}

	db, err := sql.Open("mysql", dsn)
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("gagal konek database: ", err)
	}
	log.Println("database terhubung")
	seedUsers(db)

	// Mode khusus: membuat akun lewat terminal
	if len(os.Args) > 1 && os.Args[1] == "buatuser" {
		buatUser(db, os.Args[2:])
		return
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/login", loginHandler(db))

	// Semua route di grup ini wajib login
	auth := r.Group("/")
	auth.Use(authMiddleware())
	auth.GET("/me", meHandler)
	auth.POST("/kas", kasCreate(db))
	auth.GET("/kas", kasList(db))
	auth.POST("/stok", stokCreateV2(db))
	auth.GET("/stok", stokList(db))
	auth.POST("/kas/:id/void", voidHandler(db, "cash_transactions"))
	auth.POST("/stok/:id/void", voidHandler(db, "stock_movements"))
	auth.GET("/saldo", saldoHandler(db))
	auth.GET("/export", exportHandler(db))
	auth.POST("/hutang", hutangCreate(db))
	auth.GET("/hutang", hutangList(db))
	auth.POST("/hutang/:id/bayar", hutangBayar(db))
	auth.GET("/laporan/labarugi", labaRugiHandler(db))
	auth.GET("/locations", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, name, type FROM locations")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		result := []Location{}
		for rows.Next() {
			var l Location
			if err := rows.Scan(&l.ID, &l.Name, &l.Type); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			result = append(result, l)
		}
		c.JSON(http.StatusOK, result)
	})

	r.Run(":8080")
}