package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

var jenisHutang = map[string]bool{"perorangan": true, "pabrik": true, "bank_pegadaian": true}

func HutangCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		if _, ok := wajibIDBackOffice(c, db); !ok {
			return
		}
		var req models.HutangCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		if !jenisHutang[req.CreditorType] || req.CreditorName == "" || req.Principal <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "jenis, nama kreditur, dan nominal wajib diisi dengan benar"})
			return
		}
		start, ok := utils.ParseTanggal(c, req.StartDate)
		if !ok {
			return
		}
		var due interface{}
		if req.DueDate != "" {
			if _, err := time.Parse("2006-01-02", req.DueDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "format jatuh tempo harus YYYY-MM-DD"})
				return
			}
			due = req.DueDate
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(
			`INSERT INTO debts (creditor_type, creditor_name, principal, note, start_date, due_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			req.CreditorType, req.CreditorName, req.Principal, req.Note, start, due, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "create", "debts", id, req)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "gagal menyimpan"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "hutang tersimpan"})
	}
}

func HutangTerimaDana(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req models.HutangTerimaDanaRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		nama := strings.TrimSpace(req.CreditorName)
		if !jenisHutang[req.CreditorType] || nama == "" || req.Principal <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "jenis, nama pemberi dana, dan nominal wajib diisi dengan benar"})
			return
		}
		start, ok := utils.ParseTanggal(c, req.StartDate)
		if !ok {
			return
		}
		var due interface{}
		if req.DueDate != "" {
			if _, err := time.Parse("2006-01-02", req.DueDate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "format jatuh tempo harus YYYY-MM-DD"})
				return
			}
			due = req.DueDate
		}
		boID, ok := wajibIDBackOffice(c, db)
		if !ok {
			return
		}
		kategori := "pinjaman"
		note := strings.TrimSpace(req.Note)

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			boID, "in", kategori, req.Principal, catatanDana(nama, note), start, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		cashID, _ := res.LastInsertId()

		res2, err := tx.Exec(
			`INSERT INTO debts (creditor_type, creditor_name, principal, note, start_date, due_date, cash_id, created_by)
			 VALUES (?,?,?,?,?,?,?,?)`,
			req.CreditorType, nama, req.Principal, sql.NullString{String: note, Valid: note != ""}, start, due, cashID, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		debtID, _ := res2.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "terima_dana", "debts", debtID, gin.H{"kas_id": cashID, "request": req})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":      debtID,
			"message": fmt.Sprintf("dana dari %s diterima: kas back office bertambah dan hutang tercatat", nama),
		})
	}
}

// ---------- DAFTAR HUTANG ----------

func HutangList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
			return
		}
		rows, err := db.Query(`
			SELECT d.id, d.creditor_type, d.creditor_name, d.principal,
			       IFNULL((SELECT SUM(amount) FROM debt_payments WHERE debt_id = d.id), 0),
			       IFNULL(d.note,''), d.start_date, d.due_date, d.status, d.cash_id IS NOT NULL
			FROM debts d
			WHERE d.is_void = FALSE
			ORDER BY d.status, d.id DESC`)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.HutangListRow{}
		for rows.Next() {
			var r models.HutangListRow
			var due sql.NullTime
			if err := rows.Scan(&r.ID, &r.CreditorType, &r.CreditorName, &r.Principal, &r.Terbayar,
				&r.Note, &r.StartDate, &due, &r.Status, &r.DariKas); err != nil {
				utils.Gagal(c, err)
				return
			}
			if due.Valid {
				t := due.Time
				r.DueDate = &t
			}
			r.Sisa = r.Principal - r.Terbayar
			r.Pembayaran = []models.HutangPembayaranRow{}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			utils.Gagal(c, err)
			return
		}
		rows.Close()

		if len(out) > 0 {
			ids := make([]int, 0, len(out))
			args := make([]interface{}, 0, len(out))
			for _, r := range out {
				ids = append(ids, r.ID)
				args = append(args, r.ID)
			}
			riwayat, err := db.Query(
				`SELECT p.id, p.debt_id, p.amount, p.paid_date, IFNULL(p.note,''), IFNULL(u.username,'')
				 FROM debt_payments p LEFT JOIN users u ON u.id = p.created_by
				 WHERE p.debt_id IN (`+utils.Placeholders(len(ids))+`) ORDER BY p.paid_date DESC, p.id DESC`,
				args...,
			)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			defer riwayat.Close()
			pembayaran := make(map[int][]models.HutangPembayaranRow, len(out))
			for riwayat.Next() {
				var debtID int
				var p models.HutangPembayaranRow
				if err := riwayat.Scan(&p.ID, &debtID, &p.Amount, &p.PaidDate, &p.Note, &p.Pencatat); err != nil {
					utils.Gagal(c, err)
					return
				}
				pembayaran[debtID] = append(pembayaran[debtID], p)
			}
			if err := riwayat.Err(); err != nil {
				utils.Gagal(c, err)
				return
			}
			for i := range out {
				out[i].Pembayaran = pembayaran[out[i].ID]
				if out[i].Pembayaran == nil {
					out[i].Pembayaran = []models.HutangPembayaranRow{}
				}
			}
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- BAYAR HUTANG (dengan peringatan saldo) ----------

func HutangBayar(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.HutangBayarRequest
		if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nominal harus lebih dari 0"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.PaidDate)
		if !ok {
			return
		}
		boID, ok := wajibIDBackOffice(c, db)
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var principal float64
		var status, nama string
		var isVoid bool
		err = tx.QueryRow("SELECT principal, status, creditor_name, is_void FROM debts WHERE id = ? FOR UPDATE", id).
			Scan(&principal, &status, &nama, &isVoid)
		if err != nil || isVoid {
			c.JSON(http.StatusNotFound, gin.H{"error": "hutang tidak ditemukan"})
			return
		}
		if status == "lunas" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hutang sudah lunas"})
			return
		}
		var terbayar float64
		if err := tx.QueryRow("SELECT IFNULL(SUM(amount),0) FROM debt_payments WHERE debt_id = ?", id).Scan(&terbayar); err != nil {
			utils.Gagal(c, err)
			return
		}
		sisa := principal - terbayar
		if req.Amount > sisa+0.005 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("melebihi sisa hutang (sisa %s)", services.FmtRp(sisa))})
			return
		}

		var saldo float64
		if err := tx.QueryRow(
			"SELECT IFNULL(SUM(IF(direction='in',amount,-amount)),0) FROM cash_transactions WHERE location_id = ? AND is_void = FALSE",
			boID,
		).Scan(&saldo); err != nil {
			utils.Gagal(c, err)
			return
		}
		peringatan := ""
		if saldo-req.Amount < 0 {
			peringatan = fmt.Sprintf("Saldo kas back office tidak cukup, menjadi minus %s", services.FmtRp(req.Amount-saldo))
		}

		res2, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			boID, "out", "bayar_hutang", req.Amount, catatanBayar(id, nama, req.Note), tgl, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		cashID, _ := res2.LastInsertId()
		res, err := tx.Exec(
			"INSERT INTO debt_payments (debt_id, amount, paid_date, note, cash_id, created_by) VALUES (?,?,?,?,?,?)",
			id, req.Amount, tgl, req.Note, cashID, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		payID, _ := res.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "create", "debt_payments", payID, gin.H{"request": req, "cash_id": cashID})
		services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", cashID, gin.H{"dari_pembayaran_hutang": payID})

		lunas := sisa-req.Amount < 0.005
		if lunas {
			if _, err := tx.Exec("UPDATE debts SET status = 'lunas' WHERE id = ?", id); err != nil {
				utils.Gagal(c, err)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message":    "pembayaran tersimpan",
			"sisa":       sisa - req.Amount,
			"lunas":      lunas,
			"peringatan": peringatan,
		})
	}
}

// ---------- BATALKAN HUTANG (Super Admin, hanya kalau belum ada pembayaran) ----------

func HutangVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, ok := middleware.HanyaSuperAdmin(c)
		if !ok {
			return
		}
		boID, ok := wajibIDBackOffice(c, db)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		reason, ok := utils.BacaAlasan(c)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		var isVoid bool
		var cashID sql.NullInt64
		err = tx.QueryRow("SELECT is_void, cash_id FROM debts WHERE id = ? FOR UPDATE", id).Scan(&isVoid, &cashID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "hutang tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}
		var n int
		if err := tx.QueryRow("SELECT COUNT(*) FROM debt_payments WHERE debt_id = ?", id).Scan(&n); err != nil {
			utils.Gagal(c, err)
			return
		}
		if n > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hutang sudah ada pembayaran, tidak bisa dibatalkan"})
			return
		}

		if cashID.Valid {
			res, err := tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ? AND location_id = ?", reason, cashID.Int64, boID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			n, err := res.RowsAffected()
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			if n != 1 {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "transaksi kas hutang tidak ditemukan di Back Office"})
				return
			}
			services.CatatAudit(tx, middleware.Meta(c), "void", "cash_transactions", cashID.Int64, gin.H{"reason": reason})
		}
		if _, err := tx.Exec("UPDATE debts SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "void", "debts", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "hutang dibatalkan, kas dikembalikan"})
	}
}

func catatanDana(nama, note string) string {
	catatan := "Dana dari " + nama
	if note != "" {
		catatan += ": " + note
	}
	return catatan
}

func catatanBayar(id int, nama, note string) string {
	catatan := fmt.Sprintf("Bayar hutang #%d - %s", id, nama)
	if note != "" {
		catatan += ": " + note
	}
	return catatan
}
