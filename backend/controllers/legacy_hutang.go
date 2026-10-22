// File ini berisi kode lama yang sudah tidak terdaftar di routes.
// Dibiarkan dulu sampai ada konfirmasi boleh dihapus.

package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/services"
	"toko-emas/utils"
)

func LegacyHutangList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
			return
		}
		rows, err := db.Query(`
			SELECT d.id, d.creditor_type, d.creditor_name, d.principal,
			       IFNULL((SELECT SUM(amount) FROM debt_payments WHERE debt_id = d.id), 0),
			       IFNULL(d.note,''), d.start_date, d.due_date, d.status
			FROM debts d ORDER BY d.status, d.id DESC`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type row struct {
			ID           int        `json:"id"`
			CreditorType string     `json:"creditor_type"`
			CreditorName string     `json:"creditor_name"`
			Principal    float64    `json:"principal"`
			Terbayar     float64    `json:"terbayar"`
			Sisa         float64    `json:"sisa"`
			Note         string     `json:"note"`
			StartDate    time.Time  `json:"start_date"`
			DueDate      *time.Time `json:"due_date"`
			Status       string     `json:"status"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var due sql.NullTime
			if err := rows.Scan(&r.ID, &r.CreditorType, &r.CreditorName, &r.Principal, &r.Terbayar, &r.Note, &r.StartDate, &due, &r.Status); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if due.Valid {
				t := due.Time
				r.DueDate = &t
			}
			r.Sisa = r.Principal - r.Terbayar
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

func LegacyHutangBayar(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
			return
		}
		var req struct {
			Amount   float64 `json:"amount"`
			Note     string  `json:"note"`
			PaidDate string  `json:"paid_date"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nominal harus lebih dari 0"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.PaidDate)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var principal float64
		var status, nama string
		err = tx.QueryRow("SELECT principal, status, creditor_name FROM debts WHERE id = ? FOR UPDATE", id).
			Scan(&principal, &status, &nama)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "hutang tidak ditemukan"})
			return
		}
		if status == "lunas" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hutang sudah lunas"})
			return
		}
		var terbayar float64
		tx.QueryRow("SELECT IFNULL(SUM(amount),0) FROM debt_payments WHERE debt_id = ?", id).Scan(&terbayar)
		sisa := principal - terbayar
		if req.Amount > sisa+0.005 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("melebihi sisa hutang (sisa %.0f)", sisa)})
			return
		}

		res, err := tx.Exec(
			"INSERT INTO debt_payments (debt_id, amount, paid_date, note, created_by) VALUES (?,?,?,?,?)",
			id, req.Amount, tgl, req.Note, cl.UserID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		payID, _ := res.LastInsertId()
		services.CatatAudit(tx, cl.UserID, "create", "debt_payments", payID, req)

		// Otomatis: kas Back Office keluar
		var boID int
		if err := tx.QueryRow("SELECT id FROM locations WHERE type='back_office' LIMIT 1").Scan(&boID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lokasi back office tidak ditemukan"})
			return
		}
		res2, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			boID, "out", "bayar_hutang", req.Amount, fmt.Sprintf("Bayar hutang #%d - %s", id, nama), tgl, cl.UserID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cashID, _ := res2.LastInsertId()
		services.CatatAudit(tx, cl.UserID, "create_auto", "cash_transactions", cashID, gin.H{"dari_pembayaran_hutang": payID})

		lunas := sisa-req.Amount < 0.005
		if lunas {
			tx.Exec("UPDATE debts SET status = 'lunas' WHERE id = ?", id)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": "pembayaran tersimpan", "sisa": sisa - req.Amount, "lunas": lunas})
	}
}
