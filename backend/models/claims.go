package models

import (
	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID     int    `json:"user_id"`
	Username   string `json:"username"`
	Role       string `json:"role"`
	LocationID int    `json:"location_id"` // 0 = tidak terikat cabang
	jwt.RegisteredClaims
}
