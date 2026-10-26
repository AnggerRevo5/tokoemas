package services

import (
	"database/sql"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// KesalahanInput menandai kesalahan isian pengguna (dibalas 400), berbeda dari kegagalan sistem.
type KesalahanInput struct{ Pesan string }

func (e KesalahanInput) Error() string { return e.Pesan }

// AdalahKesalahanInput memeriksa apakah err berasal dari isian pengguna.
func AdalahKesalahanInput(err error) bool {
	var ki KesalahanInput
	return errors.As(err, &ki)
}

func salah(pesan string) error { return KesalahanInput{Pesan: pesan} }

const (
	minPanjangPassword = 6
	minPanjangUsername = 3
	maksPanjangNama    = 50
)

// DaftarRole adalah role yang tersedia. Fitur tiap role sama untuk semua akun dengan role itu.
var DaftarRole = []string{"super_admin", "owner", "cabang", "pj"}

func RoleValid(role string) bool {
	for _, r := range DaftarRole {
		if r == role {
			return true
		}
	}
	return false
}

// BcryptCost adalah tingkat kesulitan hash password. Hash lama dengan cost lebih rendah
// diperbarui otomatis saat pemiliknya berhasil login.
const BcryptCost = 12

// maksPanjangPassword: bcrypt hanya memakai 72 byte pertama, jadi password lebih panjang ditolak.
const maksPanjangPassword = 72

// HashPassword memeriksa panjang password lalu mengenkripsinya.
func HashPassword(password string) (string, error) {
	if len(password) < minPanjangPassword {
		return "", salah("password minimal 6 karakter")
	}
	if len(password) > maksPanjangPassword {
		return "", salah("password maksimal 72 karakter")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// PerluHashUlang bernilai true bila hash dibuat dengan cost lebih rendah dari BcryptCost.
func PerluHashUlang(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	return err == nil && cost < BcryptCost
}

// KesalahanIsian membuat kesalahan isian (dibalas 400) dari luar paket services.
func KesalahanIsian(pesan string) error { return salah(pesan) }

// LokasiUntukRole: role cabang wajib punya lokasi bertipe cabang, role lain tidak terikat lokasi (nil).
func LokasiUntukRole(db *sql.DB, role string, locID int) (interface{}, error) {
	if role != "cabang" {
		return nil, nil
	}
	if locID <= 0 {
		return nil, salah("akun cabang wajib memilih cabang")
	}
	if _, ok := CekCabang(db, locID); !ok {
		return nil, salah("cabang tidak ditemukan")
	}
	return locID, nil
}

// TambahUser membuat akun baru (dipakai halaman Akun dan perintah buatuser di terminal).
func TambahUser(db *sql.DB, username, password, role string, locID int) error {
	username = strings.TrimSpace(username)
	if len(username) < minPanjangUsername || len(username) > maksPanjangNama || strings.ContainsAny(username, " \t") {
		return salah("username 3 sampai 50 karakter dan tanpa spasi")
	}
	if !RoleValid(role) {
		return salah("role harus: super_admin, owner, cabang, atau pj")
	}
	lokasi, err := LokasiUntukRole(db, role, locID)
	if err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", username).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return salah("username sudah dipakai")
	}
	_, err = db.Exec(
		"INSERT INTO users (username, password_hash, role, location_id) VALUES (?, ?, ?, ?)",
		username, hash, role, lokasi,
	)
	return err
}
