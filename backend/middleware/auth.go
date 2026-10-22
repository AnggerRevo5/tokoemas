package middleware

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"toko-emas/config"
	"toko-emas/models"
)

// AuthMiddleware memeriksa token, lalu mencocokkan akun ke database: akun yang dinonaktifkan langsung
// ditolak, dan perubahan role atau cabang oleh Super Admin langsung berlaku tanpa menunggu login ulang.
func AuthMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "harus login"})
			return
		}

		claims := &models.Claims{}
		token, err := jwt.ParseWithClaims(
			strings.TrimPrefix(header, "Bearer "),
			claims,
			func(t *jwt.Token) (interface{}, error) { return config.JWTSecret, nil },
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "sesi tidak valid atau habis"})
			return
		}

		var role string
		var locID sql.NullInt64
		var aktif bool
		err = db.QueryRow("SELECT role, location_id, is_active FROM users WHERE id = ?", claims.UserID).Scan(&role, &locID, &aktif)
		if err != nil || !aktif {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "akun tidak aktif, silakan hubungi Super Admin"})
			return
		}
		claims.Role = role
		claims.LocationID = int(locID.Int64)

		c.Set("claims", claims)
		c.Next()
	}
}
