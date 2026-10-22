package controllers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"toko-emas/config"
	"toko-emas/models"
)

func Login(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}

		var id int
		var hash, role string
		var locID sql.NullInt64
		var active bool
		err := db.QueryRow(
			"SELECT id, password_hash, role, location_id, is_active FROM users WHERE username = ?",
			req.Username,
		).Scan(&id, &hash, &role, &locID, &active)

		if err != nil || !active || bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "username atau password salah"})
			return
		}

		claims := models.Claims{
			UserID:     id,
			Username:   req.Username,
			Role:       role,
			LocationID: int(locID.Int64),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			},
		}
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(config.JWTSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal membuat token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":       token,
			"username":    req.Username,
			"role":        role,
			"location_id": int(locID.Int64),
		})
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
