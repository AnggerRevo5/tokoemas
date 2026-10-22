package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "time/tzdata"

	"toko-emas/config"
	"toko-emas/routes"
	"toko-emas/services"
)

func main() {
	// Mode khusus: mencetak hash password lewat terminal
	if len(os.Args) > 2 && os.Args[1] == "hash" {
		services.CetakHash(os.Args[2])
		return
	}

	db, err := config.OpenDB()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("database terhubung")
	services.SeedUsers(db)

	// Mode khusus: membuat akun lewat terminal
	if len(os.Args) > 1 && os.Args[1] == "buatuser" {
		services.BuatUser(db, os.Args[2:])
		return
	}

	r := gin.Default()
	routes.Setup(r, db)
	r.Run(":8080")
}
