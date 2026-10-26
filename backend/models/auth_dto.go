package models

// LoginRequest adalah payload JSON untuk Login.
type LoginRequest struct {
	Username string `json:"username" binding:"max=50"`
	Password string `json:"password" binding:"max=200"`
}
