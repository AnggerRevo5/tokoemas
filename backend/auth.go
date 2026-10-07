package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)	

// Nanti dipindah ke environment variable saat deploy
var jwtSecret = func() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		if os.Getenv("APP_ENV") == "production" {
			log.Fatal("JWT_SECRET wajib diisi di production")
		}
		s = "kunci-khusus-pengembangan-lokal"
	}
	return []byte(s)
}()

type Claims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	LocationID int    `json:"location_id"` // 0 = tidak terikat cabang
	jwt.RegisteredClaims
}

func buatUser(db *sql.DB, args []string) {
	if len(args) < 3 {
		fmt.Println("cara pakai: go run . buatuser <username> <password> <role> [location_id]")
		return
	}
	username, password, role := args[0], args[1], args[2]

	if role != "super_admin" && role != "owner" && role != "cabang" {
		fmt.Println("role harus: super_admin, owner, atau cabang")
		return
	}

	var locID interface{} = nil
	if role == "cabang" {
		if len(args) < 4 {
			fmt.Println("role cabang wajib isi location_id (1=Kepanjen, 2=Mall Dinoyo)")
			return
		}
		id, err := strconv.Atoi(args[3])
		if err != nil {
			fmt.Println("location_id harus angka")
			return
		}
		locID = id
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("gagal enkripsi password:", err)
		return
	}

	_, err = db.Exec(
		"INSERT INTO users (username, password_hash, role, location_id) VALUES (?, ?, ?, ?)",
		username, string(hash), role, locID,
	)
	if err != nil {
		fmt.Println("gagal membuat user:", err)
		return
	}
	fmt.Println("user berhasil dibuat:", username, "(", role, ")")
}

func loginHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
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

		claims := Claims{
			UserID:     id,
			Username:   req.Username,
			Role:       role,
			LocationID: int(locID.Int64),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(12 * time.Hour)),
			},
		}
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSecret)
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

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "harus login"})
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(
			strings.TrimPrefix(header, "Bearer "),
			claims,
			func(t *jwt.Token) (interface{}, error) { return jwtSecret, nil },
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "sesi tidak valid atau habis"})
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}

func meHandler(c *gin.Context) {
	claims := c.MustGet("claims").(*Claims)
	c.JSON(http.StatusOK, gin.H{
		"user_id":     claims.UserID,
		"username":    claims.Username,
		"role":        claims.Role,
		"location_id": claims.LocationID,
	})
}