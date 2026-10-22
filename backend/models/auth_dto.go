package models

// LoginRequest adalah payload JSON untuk Login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
