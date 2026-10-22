package models

import "time"

// AkunCreateRequest adalah payload untuk menambah akun baru.
type AkunCreateRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Role       string `json:"role"`
	LocationID int    `json:"location_id"`
}

// AkunUpdateRequest adalah payload untuk mengubah role, lokasi, dan status aktif akun.
type AkunUpdateRequest struct {
	Role       string `json:"role"`
	LocationID int    `json:"location_id"`
	IsActive   bool   `json:"is_active"`
}

// AkunPasswordRequest adalah payload untuk mengganti password akun oleh Super Admin.
type AkunPasswordRequest struct {
	Password string `json:"password"`
}

// LokasiCreateRequest adalah payload untuk menambah cabang baru.
type LokasiCreateRequest struct {
	Name string `json:"name"`
}

// AkunRow adalah satu baris daftar akun.
type AkunRow struct {
	ID         int       `json:"id"`
	Username   string    `json:"username"`
	Role       string    `json:"role"`
	LocationID int       `json:"location_id"`
	Lokasi     string    `json:"lokasi"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

// RingkasanRole adalah jumlah akun aktif dan nonaktif untuk satu role.
type RingkasanRole struct {
	Role     string `json:"role"`
	Aktif    int    `json:"aktif"`
	Nonaktif int    `json:"nonaktif"`
}
