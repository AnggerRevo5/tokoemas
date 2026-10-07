package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// tabel hanya diisi dari kode kita sendiri (bukan dari input user)
func voidHandler(db *sql.DB, tabel string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		if cl.Role != "super_admin" && cl.Role != "cabang" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses membatalkan transaksi"})
			return
		}

		var req struct {
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan pembatalan wajib diisi"})
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var locID int
		var isVoid, recent bool
		var relID sql.NullInt64
		var kat string
		err = tx.QueryRow(
			"SELECT location_id, is_void, related_id, created_at >= NOW() - INTERVAL 1 DAY, category FROM "+tabel+" WHERE id = ? FOR UPDATE",
			id,
		).Scan(&locID, &isVoid, &relID, &recent, &kat)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "transaksi tidak ditemukan"})
			return
		}
		if cl.Role == "cabang" {
			if locID != cl.LocationID {
				c.JSON(http.StatusForbidden, gin.H{"error": "bukan transaksi lokasi kamu"})
				return
			}
			if !recent {
				c.JSON(http.StatusForbidden, gin.H{"error": "lebih dari 24 jam, minta Super Admin yang membatalkan"})
				return
			}
			if tabel == "stock_movements" && relID.Valid && kat == "barang_masuk" {
				c.JSON(http.StatusForbidden, gin.H{"error": "barang kiriman dari Back Office hanya bisa dibatalkan Super Admin"})
				return
			}
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "transaksi sudah dibatalkan"})
			return
		}
		if tabel == "cash_transactions" && kat == "bayar_hutang" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pembayaran hutang tidak bisa dibatalkan dari sini, hubungi Super Admin"})
			return
		}

		ids := []int{id}
		if relID.Valid {
			ids = append(ids, int(relID.Int64))
		}
		for _, x := range ids {
			if _, err := tx.Exec("UPDATE "+tabel+" SET is_void = TRUE, void_reason = ? WHERE id = ?", req.Reason, x); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			catatAudit(tx, cl.UserID, "void", tabel, int64(x), gin.H{"reason": req.Reason})
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "transaksi dibatalkan", "ids": ids})
	}
}

func saldoHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)

		q := `SELECT l.id, l.name,
		  IFNULL((SELECT SUM(IF(direction='in',amount,-amount)) FROM cash_transactions WHERE location_id=l.id AND is_void=FALSE),0),
		  IFNULL((SELECT SUM(IF(direction='in',weight_gram,-weight_gram)) FROM stock_movements WHERE location_id=l.id AND is_void=FALSE),0),
		  IFNULL((SELECT SUM(IF(direction='in',qty,-qty)) FROM stock_movements WHERE location_id=l.id AND is_void=FALSE),0)
		  FROM locations l`
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " WHERE l.id = ?"
			args = append(args, cl.LocationID)
		}
		q += " ORDER BY l.id"

		rows, err := db.Query(q, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type row struct {
			LocationID int     `json:"location_id"`
			Name       string  `json:"name"`
			SaldoKas   float64 `json:"saldo_kas"`
			StokGram   float64 `json:"stok_gram"`
			StokPcs    int     `json:"stok_pcs"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.LocationID, &r.Name, &r.SaldoKas, &r.StokGram, &r.StokPcs); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}