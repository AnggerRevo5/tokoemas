package middleware

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"toko-emas/config"
	"toko-emas/models"
	"toko-emas/services"
)

const (
	// NamaCookieSesi menyimpan token sesi (HttpOnly: tidak bisa dibaca JavaScript).
	NamaCookieSesi = "te_sesi"
	// NamaCookieCSRF dibaca frontend lalu dikirim ulang di header X-CSRF-Token (double submit).
	NamaCookieCSRF = "te_csrf"
	headerCSRF     = "X-CSRF-Token"

	penerbitToken = "toko-emas"
	audiensToken  = "toko-emas-web"
)

// BuatToken membuat JWT HS256 yang terikat ke satu sesi di server (jti = ID sesi).
func BuatToken(userID int, username, role string, locID int, sesiID string) (string, error) {
	now := time.Now()
	claims := models.Claims{
		UserID:     userID,
		Username:   username,
		Role:       role,
		LocationID: locID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        sesiID,
			Issuer:    penerbitToken,
			Audience:  jwt.ClaimStrings{audiensToken},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(config.SesiJam()) * time.Hour)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(config.JWTSecret)
}

// PasangCookieSesi memasang cookie sesi (HttpOnly) dan cookie CSRF setelah login berhasil.
func PasangCookieSesi(c *gin.Context, token, sesiID string) {
	umur := config.SesiJam() * 3600
	http.SetCookie(c.Writer, &http.Cookie{
		Name: NamaCookieSesi, Value: token, Path: config.CookiePath(), MaxAge: umur,
		HttpOnly: true, Secure: config.CookieSecure(), SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name: NamaCookieCSRF, Value: services.TokenCSRF(sesiID), Path: "/", MaxAge: umur,
		HttpOnly: false, Secure: config.CookieSecure(), SameSite: http.SameSiteStrictMode,
	})
}

// HapusCookieSesi menghapus kedua cookie sesi di browser.
func HapusCookieSesi(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: NamaCookieSesi, Value: "", Path: config.CookiePath(), MaxAge: -1,
		HttpOnly: true, Secure: config.CookieSecure(), SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(c.Writer, &http.Cookie{
		Name: NamaCookieCSRF, Value: "", Path: "/", MaxAge: -1,
		Secure: config.CookieSecure(), SameSite: http.SameSiteStrictMode,
	})
}

// BacaToken memeriksa tanda tangan, algoritma (hanya HS256), penerbit, audiens, dan masa berlaku token.
func BacaToken(s string) (*models.Claims, bool) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(s, claims,
		func(t *jwt.Token) (interface{}, error) { return config.JWTSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(penerbitToken),
		jwt.WithAudience(audiensToken),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !token.Valid || claims.ID == "" || claims.UserID <= 0 {
		return nil, false
	}
	return claims, true
}

func tolakSesi(c *gin.Context, pesan string) {
	HapusCookieSesi(c)
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": pesan})
}

// AuthMiddleware memeriksa cookie sesi, mencocokkan sesi dan akun ke database, lalu memeriksa token CSRF
// untuk request yang mengubah data. Akun yang dinonaktifkan atau sesi yang dicabut langsung ditolak, dan
// perubahan role atau cabang oleh Super Admin langsung berlaku tanpa menunggu login ulang.
func AuthMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(NamaCookieSesi)
		if err != nil || cookie == "" {
			tolakSesi(c, "harus login")
			return
		}
		claims, ok := BacaToken(cookie)
		if !ok {
			tolakSesi(c, "sesi tidak valid atau habis")
			return
		}

		var role, username string
		var locID sql.NullInt64
		var aktif, sesiBerlaku, idle bool
		err = db.QueryRow(
			`SELECT u.role, u.location_id, u.is_active, u.username,
			        s.revoked_at IS NULL AND s.expires_at > NOW(),
			        s.last_seen < NOW() - INTERVAL ? MINUTE
			 FROM user_sessions s JOIN users u ON u.id = s.user_id
			 WHERE s.id = ? AND s.user_id = ?`,
			config.SesiIdleMenit(), claims.ID, claims.UserID,
		).Scan(&role, &locID, &aktif, &username, &sesiBerlaku, &idle)
		if err != nil || !sesiBerlaku {
			tolakSesi(c, "sesi tidak valid atau habis, silakan login lagi")
			return
		}
		if idle {
			services.CabutSesi(db, claims.ID)
			tolakSesi(c, "sesi berakhir karena lama tidak dipakai, silakan login lagi")
			return
		}
		if !aktif {
			services.CabutSesi(db, claims.ID)
			tolakSesi(c, "akun tidak aktif, silakan hubungi Super Admin")
			return
		}

		// request yang mengubah data wajib membawa token CSRF yang cocok dengan sesi ini
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if !services.CSRFCocok(claims.ID, c.GetHeader(headerCSRF)) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "sesi tidak valid, muat ulang halaman lalu coba lagi"})
				return
			}
		}

		// tandai sesi masih dipakai (paling sering sekali per menit)
		db.Exec("UPDATE user_sessions SET last_seen = NOW() WHERE id = ? AND last_seen < NOW() - INTERVAL 1 MINUTE", claims.ID)

		claims.Role = role
		claims.Username = username
		claims.LocationID = int(locID.Int64)
		c.Set("claims", claims)
		c.Next()
	}
}

// Meta mengambil identitas pelaku dari sesi (untuk audit). Tidak pernah dari isian pengguna.
func Meta(c *gin.Context) services.MetaAudit {
	m := services.MetaAudit{IP: c.ClientIP()}
	if v, ok := c.Get("claims"); ok {
		cl := v.(*models.Claims)
		m.UserID, m.Username, m.LocationID = cl.UserID, cl.Username, cl.LocationID
	}
	return m
}
