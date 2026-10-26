package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
)

// hashSamaran dipakai saat username tidak ada, agar waktu respons sama dengan password salah
// (penyerang tidak bisa menebak username mana yang terdaftar).
var hashSamaran, _ = bcrypt.GenerateFromPassword([]byte("bukan-password-siapa-pun"), services.BcryptCost)

const pesanLoginGagal = "username atau password salah"

func Login(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		username := strings.TrimSpace(req.Username)
		if !middleware.CekBatasLogin(c, strings.ToLower(username)) {
			services.CatatKeamanan(db, eventDari(c, "login_dibatasi", username, 0, 0, "terlalu banyak percobaan"))
			return
		}

		var id int
		var hash, role string
		var locID sql.NullInt64
		var active bool
		err := db.QueryRow(
			"SELECT id, password_hash, role, location_id, is_active FROM users WHERE username = ?",
			username,
		).Scan(&id, &hash, &role, &locID, &active)

		alasan := ""
		switch {
		case err == sql.ErrNoRows:
			bcrypt.CompareHashAndPassword(hashSamaran, []byte(req.Password))
			alasan = "username tidak terdaftar"
		case err != nil:
			log.Printf("login: gagal membaca akun: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan di server"})
			return
		case bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil:
			alasan = "password salah"
		case !active:
			alasan = "akun nonaktif"
		}
		if alasan != "" {
			if middleware.CatatLoginGagal(c, strings.ToLower(username)) {
				alasan += ", dikunci sementara"
			}
			services.CatatKeamanan(db, eventDari(c, "login_gagal", username, id, int(locID.Int64), alasan))
			// pesan sama untuk semua alasan agar tidak membocorkan username yang terdaftar
			c.JSON(http.StatusUnauthorized, gin.H{"error": pesanLoginGagal})
			return
		}
		middleware.CatatLoginBerhasil(c, strings.ToLower(username))

		// perbarui hash lama ke cost bcrypt yang lebih kuat (password tidak berubah)
		if services.PerluHashUlang(hash) {
			if baru, err := bcrypt.GenerateFromPassword([]byte(req.Password), services.BcryptCost); err == nil {
				db.Exec("UPDATE users SET password_hash = ? WHERE id = ? AND password_hash = ?", string(baru), id, hash)
			}
		}

		sesiID, err := services.BuatSesi(db, id, c.ClientIP(), c.Request.UserAgent())
		if err != nil {
			log.Printf("login: gagal membuat sesi: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan di server"})
			return
		}
		token, err := middleware.BuatToken(id, username, role, int(locID.Int64), sesiID)
		if err != nil {
			log.Printf("login: gagal membuat token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan di server"})
			return
		}
		middleware.PasangCookieSesi(c, token, sesiID)
		services.CatatKeamanan(db, eventDari(c, "login_berhasil", username, id, int(locID.Int64), ""))

		// token tidak dikirim di body: tersimpan di cookie HttpOnly yang tidak bisa dibaca JavaScript
		c.JSON(http.StatusOK, gin.H{
			"username":    username,
			"role":        role,
			"location_id": int(locID.Int64),
		})
	}
}

// Logout mencabut sesi di server dan menghapus cookie di browser.
func Logout(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if err := services.CabutSesi(db, cl.ID); err != nil {
			log.Printf("logout: gagal mencabut sesi: %v", err)
		}
		middleware.HapusCookieSesi(c)
		services.CatatKeamanan(db, eventDari(c, "logout", cl.Username, cl.UserID, cl.LocationID, ""))
		c.JSON(http.StatusOK, gin.H{"message": "berhasil keluar"})
	}
}

func Me(c *gin.Context) {
	claims := c.MustGet("claims").(*models.Claims)
	c.JSON(http.StatusOK, gin.H{
		"user_id":     claims.UserID,
		"username":    claims.Username,
		"role":        claims.Role,
		"location_id": claims.LocationID,
	})
}

func eventDari(c *gin.Context, event, username string, userID, locID int, detail string) services.EventKeamanan {
	return services.EventKeamanan{
		Event: event, Username: username, UserID: userID, LocationID: locID,
		IP: c.ClientIP(), UserAgent: c.Request.UserAgent(), Detail: detail,
	}
}
