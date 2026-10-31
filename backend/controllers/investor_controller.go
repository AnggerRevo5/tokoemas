package controllers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

const (
	format         = "2006-01-02"
	bungaDefaultBP = 100 // 1% per bulan
)

var tanggalPalingAwal = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

var labelGerak = map[string]string{
	services.GerakSetor:      "setor",
	services.GerakTarikPokok: "tarik pokok",
	services.GerakBayarBunga: "bayar bunga",
}

type infoInvestor struct {
	ID    int
	Nama  string
	Note  string
	Acuan time.Time
	BP    int64
}

func ringkasInvestor(inf infoInvestor, gerak []services.Gerak, hari time.Time) models.InvestorRingkasan {
	bunga := services.HitungBunga(inf.Acuan, inf.BP, gerak, hari)
	total := services.TotalBunga(bunga)
	dibayar := services.Dibayar(gerak)
	r := models.InvestorRingkasan{
		ID: inf.ID, Name: inf.Nama, Note: inf.Note, StartDate: inf.Acuan.Format(format),
		RatePercent:  float64(inf.BP) / 100,
		Pokok:        services.Pokok(gerak, hari),
		BungaTotal:   total,
		BungaDibayar: dibayar,
		BungaBelum:   total - dibayar,
	}
	if r.Pokok > 0 {
		b := services.SiklusBerikut(inf.Acuan, inf.BP, gerak, hari)
		r.JatuhTempo, r.BungaBerikut, r.DasarBerikut = b.JatuhTempo.Format(format), b.Bunga, b.Dasar
	}
	return r
}

// ---------- DAFTAR INVESTOR ----------

func InvestorList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
			return
		}
		rows, err := db.Query("SELECT id, name, IFNULL(note,''), start_date, rate_bp FROM investors ORDER BY id DESC")
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		daftar := []infoInvestor{}
		for rows.Next() {
			var inf infoInvestor
			if err := rows.Scan(&inf.ID, &inf.Nama, &inf.Note, &inf.Acuan, &inf.BP); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			inf.Acuan = services.TanggalSaja(inf.Acuan)
			daftar = append(daftar, inf)
		}
		rows.Close()

		hari := services.HariIniWIB()
		out := make([]models.InvestorRingkasan, 0, len(daftar))
		for _, inf := range daftar {
			gerak, err := services.GerakInvestor(db, inf.ID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, ringkasInvestor(inf, gerak, hari))
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- DETAIL INVESTOR ----------

func InvestorDetail(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var inf infoInvestor
		err := db.QueryRow("SELECT id, name, IFNULL(note,''), start_date, rate_bp FROM investors WHERE id = ?", id).
			Scan(&inf.ID, &inf.Nama, &inf.Note, &inf.Acuan, &inf.BP)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "investor tidak ditemukan"})
			return
		}
		inf.Acuan = services.TanggalSaja(inf.Acuan)

		gerak, err := services.GerakInvestor(db, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		hari := services.HariIniWIB()

		rows, err := db.Query(
			`SELECT m.id, m.jenis, m.amount, m.tx_date, IFNULL(m.note,''), m.is_void, IFNULL(m.void_reason,''),
			        IFNULL(u.username,''), m.created_at
			 FROM investor_movements m LEFT JOIN users u ON u.id = m.created_by
			 WHERE m.investor_id = ? ORDER BY m.tx_date DESC, m.id DESC`, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		riwayat := []models.InvestorGerakRow{}
		for rows.Next() {
			var r models.InvestorGerakRow
			var tgl, dibuat time.Time
			if err := rows.Scan(&r.ID, &r.Jenis, &r.Amount, &tgl, &r.Note, &r.IsVoid, &r.VoidReason, &r.Pencatat, &dibuat); err != nil {
				utils.Gagal(c, err)
				return
			}
			r.TxDate, r.CreatedAt = tgl.Format(format), dibuat.Format("2006-01-02 15:04")
			riwayat = append(riwayat, r)
		}

		jadwal := []models.InvestorBungaRow{}
		for _, b := range services.HitungBunga(inf.Acuan, inf.BP, gerak, hari) {
			jadwal = append(jadwal, models.InvestorBungaRow{
				Siklus: b.Siklus, Mulai: b.Mulai.Format(format), JatuhTempo: b.JatuhTempo.Format(format),
				Dasar: b.Dasar, Bunga: b.Bunga,
			})
		}
		// terbaru di atas
		for i, j := 0, len(jadwal)-1; i < j; i, j = i+1, j-1 {
			jadwal[i], jadwal[j] = jadwal[j], jadwal[i]
		}
		c.JSON(http.StatusOK, gin.H{
			"investor": ringkasInvestor(inf, gerak, hari),
			"riwayat":  riwayat,
			"bunga":    jadwal,
		})
	}
}

// ---------- INVESTOR BARU (beserta setoran pertama) ----------

func InvestorCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		var req models.InvestorBaruReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		nama := strings.TrimSpace(req.Name)
		if nama == "" || req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama dan nominal setoran wajib diisi dengan benar"})
			return
		}
		tglStr, ok := utils.ParseTanggal(c, req.StartDate)
		if !ok {
			return
		}
		tgl, _ := time.Parse(format, tglStr)
		if tgl.After(services.HariIniWIB()) || tgl.Before(tanggalPalingAwal) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tanggal setor tidak boleh di masa depan"})
			return
		}
		bp := int64(bungaDefaultBP)
		if req.RatePercent > 0 {
			bp = int64(math.Round(req.RatePercent * 100))
		}
		if bp < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "persentase bunga terlalu kecil"})
			return
		}
		note := strings.TrimSpace(req.Note)

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec("INSERT INTO investors (name, note, start_date, rate_bp, created_by) VALUES (?,?,?,?,?)",
			nama, utils.StrNull(note), tglStr, bp, cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		res2, err := tx.Exec(
			"INSERT INTO investor_movements (investor_id, jenis, amount, tx_date, note, created_by) VALUES (?,?,?,?,?,?)",
			id, services.GerakSetor, req.Amount, tglStr, "Setoran awal", cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		gid, _ := res2.LastInsertId()
		services.CatatAudit(tx, middleware.Meta(c), "create", "investors", id,
			gin.H{"name": nama, "start_date": tglStr, "rate_bp": bp, "amount": req.Amount, "gerak_id": gid})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":      id,
			"message": fmt.Sprintf("Investor %s tersimpan, bunga pertama jatuh tempo %s", nama, services.JatuhTempoKe(tgl, 1).Format(format)),
		})
	}
}

// ---------- CATAT SETOR / TARIK POKOK / BAYAR BUNGA ----------

// gerakBaru adalah satu catatan yang akan disimpan.
type gerakBaru struct {
	Jenis  string
	Jumlah int64
}

// simpanGerak menyimpan satu atau beberapa catatan investor sekaligus dalam satu transaksi.
// Semua catatan diperiksa bersama-sama dengan riwayat yang ada; bila satu saja gagal, tidak ada yang tersimpan.
func simpanGerak(c *gin.Context, db *sql.DB, id int, tglStr, note string, baru []gerakBaru) ([]int64, bool) {
	cl := c.MustGet("claims").(*models.Claims)
	tgl, _ := time.Parse(format, tglStr)
	hari := services.HariIniWIB()
	if tgl.After(hari) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tanggal tidak boleh di masa depan"})
		return nil, false
	}

	tx, err := db.Begin()
	if err != nil {
		utils.Gagal(c, err)
		return nil, false
	}
	defer tx.Rollback()

	// kunci investor supaya dua pencatatan bersamaan tidak saling melewati pemeriksaan
	var acuan time.Time
	var bp int64
	if err := tx.QueryRow("SELECT start_date, rate_bp FROM investors WHERE id = ? FOR UPDATE", id).Scan(&acuan, &bp); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "investor tidak ditemukan"})
		return nil, false
	}
	acuan = services.TanggalSaja(acuan)
	if tgl.Before(acuan) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tanggal tidak boleh sebelum tanggal setor pertama (" + acuan.Format(format) + ")"})
		return nil, false
	}

	gerak, err := services.GerakInvestor(tx, id)
	if err != nil {
		utils.Gagal(c, err)
		return nil, false
	}
	for _, g := range baru {
		gerak = append(gerak, services.Gerak{Jenis: g.Jenis, Jumlah: g.Jumlah, Tanggal: tgl})
	}
	if err := services.ValidasiGerak(acuan, bp, gerak, hari); err != nil {
		balasKesalahan(c, err)
		return nil, false
	}

	ids := make([]int64, 0, len(baru))
	for _, g := range baru {
		res, err := tx.Exec(
			"INSERT INTO investor_movements (investor_id, jenis, amount, tx_date, note, created_by) VALUES (?,?,?,?,?,?)",
			id, g.Jenis, g.Jumlah, tglStr, utils.StrNull(note), cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return nil, false
		}
		gid, _ := res.LastInsertId()
		ids = append(ids, gid)
		services.CatatAudit(tx, middleware.Meta(c), "create", "investor_movements", gid,
			gin.H{"investor_id": id, "jenis": g.Jenis, "amount": g.Jumlah, "tx_date": tglStr, "note": note})
	}
	if err := tx.Commit(); err != nil {
		utils.Gagal(c, err)
		return nil, false
	}
	return ids, true
}

func InvestorGerak(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.InvestorGerakReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		if _, ada := labelGerak[req.Jenis]; !ada {
			c.JSON(http.StatusBadRequest, gin.H{"error": "jenis harus setor, tarik pokok, atau bayar bunga"})
			return
		}
		if req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nominal harus lebih dari 0 (rupiah bulat)"})
			return
		}
		tglStr, ok := utils.ParseTanggal(c, req.TxDate)
		if !ok {
			return
		}
		ids, ok := simpanGerak(c, db, id, tglStr, strings.TrimSpace(req.Note), []gerakBaru{{req.Jenis, req.Amount}})
		if !ok {
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"id":      ids[0],
			"message": fmt.Sprintf("%s %s tersimpan", strings.ToUpper(labelGerak[req.Jenis][:1])+labelGerak[req.Jenis][1:], services.FmtRp(float64(req.Amount))),
		})
	}
}

// ---------- AMBIL DANA: tarik pokok dan bayar bunga sekaligus ----------

func InvestorAmbil(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.InvestorAmbilReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		if req.Pokok < 0 || req.Bunga < 0 || req.Pokok+req.Bunga <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi nominal pokok dan/atau bunga yang diambil (rupiah bulat)"})
			return
		}
		tglStr, ok := utils.ParseTanggal(c, req.TxDate)
		if !ok {
			return
		}
		baru := []gerakBaru{}
		if req.Pokok > 0 {
			baru = append(baru, gerakBaru{services.GerakTarikPokok, req.Pokok})
		}
		if req.Bunga > 0 {
			baru = append(baru, gerakBaru{services.GerakBayarBunga, req.Bunga})
		}
		if _, ok := simpanGerak(c, db, id, tglStr, strings.TrimSpace(req.Note), baru); !ok {
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"message": fmt.Sprintf("Pengambilan tersimpan: pokok %s, bunga %s (total %s)",
				services.FmtRp(float64(req.Pokok)), services.FmtRp(float64(req.Bunga)), services.FmtRp(float64(req.Pokok+req.Bunga))),
		})
	}
}

// ---------- BATALKAN SATU CATATAN (tetap tampil di riwayat) ----------

func InvestorGerakVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, ok := middleware.HanyaAdminOwner(c); !ok {
			return
		}
		mid, ok := utils.IDParam(c)
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

		var investorID int
		if err := tx.QueryRow("SELECT investor_id FROM investor_movements WHERE id = ?", mid).Scan(&investorID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "catatan tidak ditemukan"})
			return
		}
		var acuan time.Time
		var bp int64
		if err := tx.QueryRow("SELECT start_date, rate_bp FROM investors WHERE id = ? FOR UPDATE", investorID).Scan(&acuan, &bp); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "investor tidak ditemukan"})
			return
		}
		var isVoid bool
		var jenis string
		var jumlah int64
		if err := tx.QueryRow("SELECT is_void, jenis, amount FROM investor_movements WHERE id = ? FOR UPDATE", mid).Scan(&isVoid, &jenis, &jumlah); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "catatan tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}

		semua, err := services.GerakInvestor(tx, investorID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		sisa := make([]services.Gerak, 0, len(semua))
		for _, g := range semua {
			if g.ID != mid {
				sisa = append(sisa, g)
			}
		}
		if err := services.ValidasiGerak(services.TanggalSaja(acuan), bp, sisa, services.HariIniWIB()); err != nil {
			balasKesalahan(c, fmt.Errorf("tidak bisa dibatalkan: %w", err))
			return
		}

		if _, err := tx.Exec("UPDATE investor_movements SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, mid); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "void", "investor_movements", int64(mid),
			gin.H{"investor_id": investorID, "jenis": jenis, "amount": jumlah, "reason": reason})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "catatan dibatalkan"})
	}
}
