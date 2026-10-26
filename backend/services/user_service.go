package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func BuatUser(db *sql.DB, args []string) {
	if len(args) < 3 {
		fmt.Println("cara pakai: go run . buatuser <username> <password> <role> [location_id]")
		return
	}
	username, password, role := args[0], args[1], args[2]

	locID := 0
	if role == "cabang" {
		if len(args) < 4 {
			fmt.Println("role cabang wajib isi location_id (id lokasi cabang, lihat tabel locations)")
			return
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			fmt.Println("location_id harus angka")
			return
		}
		locID = id
	}

	if err := TambahUser(db, username, password, role, locID); err != nil {
		fmt.Println("gagal membuat user:", err)
		return
	}
	fmt.Println("user berhasil dibuat:", username, "(", role, ")")
}

func SeedUsers(db *sql.DB) {
	akun := []struct {
		username string
		envName  string
		role     string
		locID    interface{}
	}{
		{"admin", "SEED_ADMIN_PASSWORD", "super_admin", nil},
		{"owner", "SEED_OWNER_PASSWORD", "owner", nil},
		{"kepanjen", "SEED_KEPANJEN_PASSWORD", "cabang", 1},
		{"dinoyo", "SEED_DINOYO_PASSWORD", "cabang", 2},
	}

	for _, a := range akun {
		pw := os.Getenv(a.envName)
		if pw == "" {
			continue
		}

		var n int
		db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", a.username).Scan(&n)
		if n > 0 {
			continue
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(pw), BcryptCost)
		if err != nil {
			log.Println("gagal enkripsi password untuk", a.username)
			continue
		}
		_, err = db.Exec(
			"INSERT INTO users (username, password_hash, role, location_id) VALUES (?, ?, ?, ?)",
			a.username, string(hash), a.role, a.locID,
		)
		if err != nil {
			log.Println("gagal membuat akun", a.username, ":", err)
			continue
		}
		log.Println("akun dibuat:", a.username, "(", a.role, ")")
	}
}

func CetakHash(pw string) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), BcryptCost)
	if err != nil {
		fmt.Println("gagal:", err)
		return
	}
	fmt.Println(string(h))
}
