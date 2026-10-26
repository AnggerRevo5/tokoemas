package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// balasKesalahan membalas 400 untuk salah isian, 500 untuk kegagalan sistem.
func balasKesalahan(c *gin.Context, err error) {
	if services.AdalahKesalahanInput(err) {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	utils.Gagal(c, err)
}

// ---------- DAFTAR AKUN (Super Admin) ----------

func AkunList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaSuperAdmin(c); !ok {
			return
		}
		rows, err := db.Query(
			`SELECT u.id, u.username, u.role, IFNULL(u.location_id,0), IFNULL(l.name,''), u.is_active, u.created_at
			 FROM users u LEFT JOIN locations l ON l.id = u.location_id
			 ORDER BY u.is_active DESC, FIELD(u.role,'super_admin','owner','cabang','pj'), u.username`)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.AkunRow{}
		ringkasan := []models.RingkasanRole{}
		hitung := map[string]int{}
		for i, r := range services.DaftarRole {
			ringkasan = append(ringkasan, models.RingkasanRole{Role: r})
			hitung[r] = i
		}
		for rows.Next() {
			var r models.AkunRow
			if err := rows.Scan(&r.ID, &r.Username, &r.Role, &r.LocationID, &r.Lokasi, &r.IsActive, &r.CreatedAt); err != nil {
				utils.Gagal(c, err)
				return
			}
			if i, ada := hitung[r.Role]; ada {
				if r.IsActive {
					ringkasan[i].Aktif++
				} else {
					ringkasan[i].Nonaktif++
				}
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "ringkasan": ringkasan})
	}
}

// ---------- TAMBAH AKUN ----------

func AkunCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := middleware.HanyaSuperAdmin(c)
		if !ok {
			return
		}
		var req models.AkunCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		req.Username = strings.TrimSpace(req.Username)
		if err := services.TambahUser(db, req.Username, req.Password, req.Role, req.LocationID); err != nil {
			balasKesalahan(c, err)
			return
		}
		catatAuditAkun(c, db, "akun_tambah", "users", 0, gin.H{"username": req.Username, "role": req.Role, "location_id": req.LocationID})
		c.JSON(http.StatusCreated, gin.H{"message": "akun " + req.Username + " dibuat"})
	}
}

// ---------- UBAH ROLE / LOKASI / STATUS AKTIF ----------

func AkunUpdate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.AkunUpdateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		if !services.RoleValid(req.Role) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "role tidak valid"})
			return
		}
		lokasi, err := services.LokasiUntukRole(db, req.Role, req.LocationID)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		if id == cl.UserID && (req.Role != "super_admin" || !req.IsActive) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tidak bisa menurunkan atau menonaktifkan akun sendiri"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var username, roleLama string
		var aktifLama bool
		var lokasiLama sql.NullInt64
		if err := tx.QueryRow("SELECT username, role, is_active, location_id FROM users WHERE id = ? FOR UPDATE", id).
			Scan(&username, &roleLama, &aktifLama, &lokasiLama); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "akun tidak ditemukan"})
			return
		}
		// harus selalu ada minimal satu Super Admin aktif
		if roleLama == "super_admin" && aktifLama && (req.Role != "super_admin" || !req.IsActive) {
			var lain int
			tx.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'super_admin' AND is_active = TRUE AND id <> ?", id).Scan(&lain)
			if lain == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "harus ada minimal satu Super Admin aktif"})
				return
			}
		}
		if _, err := tx.Exec("UPDATE users SET role = ?, location_id = ?, is_active = ? WHERE id = ?",
			req.Role, lokasi, req.IsActive, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "akun_ubah", "users", int64(id),
			gin.H{"username": username, "role": roleLama, "location_id": lokasiLama.Int64, "is_active": aktifLama},
			gin.H{"username": username, "role": req.Role, "location_id": lokasi, "is_active": req.IsActive},
		)
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		// akun yang dinonaktifkan langsung keluar dari semua perangkat
		if !req.IsActive && aktifLama {
			if _, err := services.CabutSemuaSesi(db, id, ""); err != nil {
				log.Printf("gagal mencabut sesi akun #%d: %v", id, err)
			}
		}
		c.JSON(http.StatusOK, gin.H{"message": "akun " + username + " diperbarui"})
	}
}

// ---------- GANTI PASSWORD (Super Admin) ----------

func AkunPassword(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.AkunPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		hash, err := services.HashPassword(req.Password)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		var username string
		if err := db.QueryRow("SELECT username FROM users WHERE id = ?", id).Scan(&username); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "akun tidak ditemukan"})
			return
		}
		if _, err := db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		// sesi lain akun itu diakhiri agar password lama tidak bisa dipakai lagi (sesi admin yang sedang dipakai tetap)
		dicabut, err := services.CabutSemuaSesi(db, id, cl.ID)
		if err != nil {
			log.Printf("gagal mencabut sesi akun #%d: %v", id, err)
		}
		// password tidak pernah dicatat di audit
		catatAuditAkun(c, db, "akun_password", "users", int64(id), gin.H{"username": username, "sesi_dicabut": dicabut})
		c.JSON(http.StatusOK, gin.H{"message": "password " + username + " diganti"})
	}
}

// ---------- TAMBAH CABANG BARU ----------

func LokasiCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := middleware.HanyaSuperAdmin(c)
		if !ok {
			return
		}
		var req models.LokasiCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		nama := strings.TrimSpace(req.Name)
		if len(nama) < 3 || len(nama) > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama cabang 3 sampai 50 karakter"})
			return
		}
		var n int
		db.QueryRow("SELECT COUNT(*) FROM locations WHERE LOWER(name) = LOWER(?)", nama).Scan(&n)
		if n > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama lokasi sudah dipakai"})
			return
		}
		res, err := db.Exec("INSERT INTO locations (name, type) VALUES (?, 'cabang')", nama)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		catatAuditAkun(c, db, "lokasi_tambah", "locations", id, gin.H{"name": nama})
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "cabang " + nama + " ditambahkan"})
	}
}

// catatAuditAkun menyimpan jejak audit untuk operasi yang tidak memakai transaksi.
func catatAuditAkun(c *gin.Context, db *sql.DB, aksi, tabel string, recordID int64, data interface{}) {
	tx, err := db.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	if services.CatatAudit(tx, middleware.Meta(c), aksi, tabel, recordID, data) == nil {
		tx.Commit()
	}
}
