package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"

	"toko-emas/config"
	"toko-emas/utils"
)

// BuatSesi mencatat sesi login baru di server dan mengembalikan ID-nya (dipakai sebagai jti token).
func BuatSesi(db *sql.DB, userID int, ip, userAgent string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	id := hex.EncodeToString(b)
	_, err := db.Exec(
		`INSERT INTO user_sessions (id, user_id, created_at, last_seen, expires_at, ip, user_agent)
		 VALUES (?, ?, NOW(), NOW(), NOW() + INTERVAL ? HOUR, ?, ?)`,
		id, userID, config.SesiJam(), utils.BatasTeks(ip, 45), utils.BatasTeks(userAgent, 255),
	)
	return id, err
}

// CabutSesi mengakhiri satu sesi (logout).
func CabutSesi(db *sql.DB, id string) error {
	_, err := db.Exec("UPDATE user_sessions SET revoked_at = NOW() WHERE id = ? AND revoked_at IS NULL", id)
	return err
}

// CabutSemuaSesi mengakhiri semua sesi aktif milik satu akun, kecuali sesi `kecuali` (boleh kosong).
// Dipakai saat password diganti atau akun dinonaktifkan/diubah oleh Super Admin.
func CabutSemuaSesi(db *sql.DB, userID int, kecuali string) (int64, error) {
	res, err := db.Exec(
		"UPDATE user_sessions SET revoked_at = NOW() WHERE user_id = ? AND revoked_at IS NULL AND id <> ?",
		userID, kecuali,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// TokenCSRF diturunkan dari ID sesi dengan HMAC, sehingga tidak perlu disimpan dan tidak bisa ditebak tanpa secret.
func TokenCSRF(sesiID string) string {
	m := hmac.New(sha256.New, config.JWTSecret)
	m.Write([]byte("csrf:" + sesiID))
	return hex.EncodeToString(m.Sum(nil))
}

// CSRFCocok membandingkan token CSRF dari header dengan yang diharapkan (waktu konstan).
func CSRFCocok(sesiID, dariHeader string) bool {
	return dariHeader != "" && hmac.Equal([]byte(TokenCSRF(sesiID)), []byte(dariHeader))
}
