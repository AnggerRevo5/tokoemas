package models

import "time"

// User mencerminkan tabel users.
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"` // super_admin | owner | cabang | pj
	LocationID   *int      `json:"location_id"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
}
