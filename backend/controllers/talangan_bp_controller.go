package controllers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/services"
	"toko-emas/utils"
)

// BPCreate: super admin menambah cabang BP; langsung bisa dipakai di Talangan tanpa ubah kode.
func BPCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaSuperAdmin(c); !ok {
			return
		}
		var req struct {
			Nama string `json:"nama" binding:"required,max=30"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Nama) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama cabang BP wajib diisi (maksimal 30 huruf)"})
			return
		}
		nama := strings.TrimSpace(req.Nama)
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		res, err := tx.Exec("INSERT INTO bp_cabang (nama) VALUES (?)", nama)
		if err != nil {
			if kunciDuplikat(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "nama cabang BP sudah ada"})
				return
			}
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "bp_baru", "bp_cabang", id, gin.H{"nama": nama})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "nama": nama, "aktif": true})
	}
}

// BPUpdate: ubah nama atau aktif/nonaktif. BP nonaktif tidak bisa dipilih lagi, datanya tetap terbaca.
func BPUpdate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaSuperAdmin(c); !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req struct {
			Nama  *string `json:"nama" binding:"omitempty,max=30"`
			Aktif *bool   `json:"aktif"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || (req.Nama == nil && req.Aktif == nil) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi nama atau aktif"})
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		var lamaNama string
		var lamaAktif bool
		if err := tx.QueryRow("SELECT nama, aktif FROM bp_cabang WHERE id = ? FOR UPDATE", id).Scan(&lamaNama, &lamaAktif); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "cabang BP tidak ditemukan"})
			return
		}
		nama, aktif := lamaNama, lamaAktif
		if req.Nama != nil {
			nama = strings.TrimSpace(*req.Nama)
			if nama == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "nama tidak boleh kosong"})
				return
			}
		}
		if req.Aktif != nil {
			aktif = *req.Aktif
		}
		if _, err := tx.Exec("UPDATE bp_cabang SET nama = ?, aktif = ? WHERE id = ?", nama, aktif, id); err != nil {
			if kunciDuplikat(err) {
				c.JSON(http.StatusConflict, gin.H{"error": "nama cabang BP sudah ada"})
				return
			}
			utils.Gagal(c, err)
			return
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "bp_ubah", "bp_cabang", int64(id),
			gin.H{"nama": lamaNama, "aktif": lamaAktif}, gin.H{"nama": nama, "aktif": aktif})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "cabang BP diperbarui"})
	}
}

type barisPiutang struct {
	BPID       int    `json:"bp_id"`
	BP         string `json:"bp"`
	Aktif      bool   `json:"aktif"`
	DiBOJumlah int    `json:"di_bo_jumlah"` // barang BP di stok BO yang belum ditukar
	DiBONilai  int64  `json:"di_bo_nilai"`
	TukarBelum int    `json:"tukar_belum_lunas"`
	Piutang    int64  `json:"piutang"` // BP masih harus bayar ke kita
	Hutang     int64  `json:"hutang"`  // kita masih harus bayar ke BP
}

// PiutangList: posisi per cabang BP (belum lunas). BP nonaktif tampil hanya bila masih ada saldo atau barang.
func PiutangList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaPeran(c, "pj", "owner", "super_admin"); !ok {
			return
		}
		const nominal = "IF(x.koreksi_alasan IS NOT NULL AND x.nominal_bayar IS NOT NULL, x.nominal_bayar, ABS(x.saldo))"
		rows, err := db.Query(`SELECT bp.id, bp.nama, bp.aktif,
			(SELECT COUNT(*) FROM talangan_barang t WHERE t.cabang_pemilik_id = bp.id AND t.arah = 'MASUK' AND t.status = 'DI_BO'),
			(SELECT IFNULL(CAST(SUM(t.harga_beli) AS SIGNED),0) FROM talangan_barang t WHERE t.cabang_pemilik_id = bp.id AND t.arah = 'MASUK' AND t.status = 'DI_BO'),
			(SELECT COUNT(*) FROM talangan_tukar x WHERE x.cabang_id = bp.id AND x.status_uang <> 'MASUK_KAS_BO'),
			(SELECT IFNULL(CAST(SUM(IF(x.arah_uang = 'BP_BAYAR', ` + nominal + `, 0)) AS SIGNED),0) FROM talangan_tukar x WHERE x.cabang_id = bp.id AND x.status_uang <> 'MASUK_KAS_BO'),
			(SELECT IFNULL(CAST(SUM(IF(x.arah_uang = 'KITA_BAYAR', ` + nominal + `, 0)) AS SIGNED),0) FROM talangan_tukar x WHERE x.cabang_id = bp.id AND x.status_uang <> 'MASUK_KAS_BO')
			FROM bp_cabang bp ORDER BY bp.nama`)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		out := []barisPiutang{}
		for rows.Next() {
			var r barisPiutang
			if err := rows.Scan(&r.BPID, &r.BP, &r.Aktif, &r.DiBOJumlah, &r.DiBONilai, &r.TukarBelum, &r.Piutang, &r.Hutang); err != nil {
				utils.Gagal(c, err)
				return
			}
			if !r.Aktif && r.DiBOJumlah == 0 && r.TukarBelum == 0 {
				continue
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}
