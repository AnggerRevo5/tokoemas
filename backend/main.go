package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "time/tzdata"

	"toko-emas/config"
	"toko-emas/middleware"
	"toko-emas/routes"
	"toko-emas/services"
)

func main() {
	// Mode khusus: mencetak hash password lewat terminal
	if len(os.Args) > 2 && os.Args[1] == "hash" {
		services.CetakHash(os.Args[2])
		return
	}

	if config.Produksi() {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := config.OpenDB()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("database terhubung")
	if err := services.SiapkanSkemaKeamanan(db); err != nil {
		log.Fatal(err)
	}
	services.SeedUsers(db)

	// Mode khusus: membuat akun lewat terminal
	if len(os.Args) > 1 && os.Args[1] == "buatuser" {
		services.BuatUser(db, os.Args[2:])
		return
	}

	r := gin.New()
	// IP asli pengguna hanya dipercaya dari proxy internal (nginx, cloudflared)
	if err := r.SetTrustedProxies(config.TrustedProxies()); err != nil {
		log.Fatal("TRUSTED_PROXIES tidak valid: ", err)
	}
	r.RemoteIPHeaders = []string{"CF-Connecting-IP", "X-Forwarded-For", "X-Real-IP"}
	r.Use(gin.Logger(), middleware.Pemulihan(), middleware.HeaderKeamanan(), middleware.BatasRequest(), middleware.BatasBody())
	routes.Setup(r, db)

	// batas waktu agar koneksi lambat (slowloris) tidak menahan server
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second, // laporan PDF periode panjang butuh waktu
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}
	log.Println("server berjalan di :8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
