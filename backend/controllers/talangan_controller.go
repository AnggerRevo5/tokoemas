package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// rapikanBarangTalangan memeriksa dan menormalkan jenis, bahan, dan kadar; menulis 400 bila tidak valid.
func rapikanBarangTalangan(c *gin.Context, jenis, bahan, karat string) (j, b, k string, ok bool) {
	j, b = strings.TrimSpace(jenis), strings.ToLower(strings.TrimSpace(bahan))
	if j == "" {
		j = "perhiasan"
	}
	if b == "" {
		b = "emas"
	}
	if (j != "perhiasan" && j != "logam_mulia") || (b != "emas" && b != "perak") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "jenis harus perhiasan atau logam mulia, bahan emas atau perak"})
		return "", "", "", false
	}
	switch {
	case b == "perak" && j == "logam_mulia":
		k = "999"
	case b == "perak":
		k = "925"
	case j == "logam_mulia":
		k = "24K"
	default:
		nk, valid := utils.NormalKarat(karat)
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kadar harus 1K sampai 24K"})
			return "", "", "", false
		}
		k = nk
	}
	return j, b, k, true
}

// bpAktif mengembalikan nama BP bila id ada dan aktif.
func bpAktif(c *gin.Context, tx *sql.Tx, id int) (string, bool) {
	var nama string
	var aktif bool
	if err := tx.QueryRow("SELECT nama, aktif FROM bp_cabang WHERE id = ?", id).Scan(&nama, &aktif); err != nil || !aktif {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cabang BP tidak ditemukan atau tidak aktif"})
		return "", false
	}
	return nama, true
}

// TalanganCreate: petugas toko mencatat barang MASUK (milik BP lain yang dibeli dari customer).
// Barang tidak masuk stok toko; kas toko berkurang seperti pembelian customer biasa.
func TalanganCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "super_admin")
		if !ok {
			return
		}
		var req models.TalanganCreateReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		req.NamaBarang = strings.TrimSpace(req.NamaBarang)
		if req.NamaBarang == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama barang wajib diisi"})
			return
		}
		jenis, bahan, karat, ok := rapikanBarangTalangan(c, req.Jenis, req.Bahan, req.Karat)
		if !ok {
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.Tanggal)
		if !ok {
			return
		}
		loc, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if tipe != "cabang" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "talangan masuk hanya diinput di toko"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		bp, ok := bpAktif(c, tx, req.BPID)
		if !ok {
			return
		}
		res, err := tx.Exec(
			`INSERT INTO talangan_barang (arah, cabang_pemilik_id, toko_input_id, nama_barang, jenis, bahan, berat, kadar,
			   harga_beli, tanggal_beli, no_surat, status, diinput_oleh, catatan)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			services.ArahMasuk, req.BPID, loc, req.NamaBarang, jenis, bahan, req.Berat, karat,
			req.HargaBeli, tgl, strings.TrimSpace(req.NoSurat), services.StTalangan, cl.UserID, utils.StrNull(strings.TrimSpace(req.Catatan)))
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()

		kas, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?,?,?,?,?,?,?)`,
			loc, "out", "pembelian_customer", req.HargaBeli,
			fmt.Sprintf("Talangan %s: %s", bp, req.NamaBarang), tgl, cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		kasID, _ := kas.LastInsertId()
		if _, err := tx.Exec("UPDATE talangan_barang SET kas_id = ? WHERE id = ?", kasID, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if err := services.CatatStatus(tx, int(id), "", services.StTalangan, cl.UserID, "barang dibeli dari customer"); err != nil {
			utils.Gagal(c, err)
			return
		}
		meta := middleware.Meta(c)
		services.CatatAudit(tx, meta, "talangan_baru", "talangan_barang", id, req)
		services.CatatAudit(tx, meta, "create_auto", "cash_transactions", kasID, gin.H{"talangan": id, "total": req.HargaBeli})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "talangan tercatat, kas toko berkurang " + utils.RupiahTeks(float64(req.HargaBeli))})
	}
}

// TalanganList: daftar talangan. Petugas toko hanya melihat catatan tokonya sendiri.
// Filter: arah, bp_id, status, lokasi (toko input), dan stok=1 (hanya yang sudah jadi barang di BO).
func TalanganList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if cl.Role != "cabang" && cl.Role != "owner" && cl.Role != "pj" && cl.Role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
			return
		}
		q := `SELECT t.id, t.arah, t.cabang_pemilik_id, bp.nama, IFNULL(t.toko_input_id,0), IFNULL(l.name,''),
		        IFNULL(t.stok_id,0), IFNULL(t.proses_id,0), IFNULL(t.proses_jenis,''), t.nama_barang, t.jenis, t.bahan,
		        t.berat, IFNULL(t.kadar,''), CAST(t.harga_beli AS SIGNED), t.ubah_harga, DATE_FORMAT(t.tanggal_beli,'%Y-%m-%d'),
		        t.no_surat, t.no_referensi, t.status, IFNULL(t.tukar_id,0), IFNULL(t.catatan,''), t.belum_sampai, IFNULL(t.belum_sampai_note,''),
		        IFNULL(u.username,''), t.created_at
		      FROM talangan_barang t
		      JOIN bp_cabang bp ON bp.id = t.cabang_pemilik_id
		      LEFT JOIN locations l ON l.id = t.toko_input_id
		      LEFT JOIN users u ON u.id = t.diinput_oleh
		      WHERE 1=1`
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND t.toko_input_id = ?"
			args = append(args, cl.LocationID)
		} else if v := c.Query("lokasi"); v != "" {
			q += " AND t.toko_input_id = ?"
			args = append(args, v)
		}
		if v := c.Query("arah"); v == services.ArahMasuk || v == services.ArahKeluar {
			q += " AND t.arah = ?"
			args = append(args, v)
		}
		if v := c.Query("bp_id"); v != "" {
			q += " AND t.cabang_pemilik_id = ?"
			args = append(args, v)
		}
		if v := c.Query("status"); v != "" {
			q += " AND t.status = ?"
			args = append(args, v)
		}
		q += " ORDER BY t.id DESC LIMIT 500"
		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		hasil := []models.TalanganRow{}
		for rows.Next() {
			var r models.TalanganRow
			var dibuat sql.NullTime
			if err := rows.Scan(&r.ID, &r.Arah, &r.BPID, &r.BP, &r.TokoInputID, &r.TokoInput, &r.StokID, &r.ProsesID, &r.ProsesJenis,
				&r.NamaBarang, &r.Jenis, &r.Bahan, &r.Berat, &r.Kadar, &r.HargaBeli, &r.UbahHarga, &r.TanggalBeli,
				&r.NoSurat, &r.NoReferensi, &r.Status, &r.TukarID, &r.Catatan, &r.BelumSampai, &r.BelumNote, &r.DiinputOleh, &dibuat); err != nil {
				utils.Gagal(c, err)
				return
			}
			if dibuat.Valid {
				r.DibuatPada = dibuat.Time
			} else {
				r.DibuatPada = time.Time{}
			}
			hasil = append(hasil, r)
		}
		c.JSON(http.StatusOK, hasil)
	}
}

// TalanganBPList: daftar BP aktif untuk dropdown.
func TalanganBPList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bp, err := services.DaftarBPAktif(db)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if bp == nil {
			bp = []services.BP{}
		}
		c.JSON(http.StatusOK, bp)
	}
}

// kejadianRiwayat: satu kejadian di riwayat talangan. Jenis "status" = perpindahan status talangan,
// "cuci" = kejadian pencucian barangnya (diambil dari antrean cuci dan gerak stok alur cuci lama, hanya dibaca).
type kejadianRiwayat struct {
	Jenis   string    `json:"jenis"`
	Dari    string    `json:"dari"`
	Ke      string    `json:"ke"`
	Oleh    string    `json:"oleh"`
	Waktu   time.Time `json:"waktu"`
	Catatan string    `json:"catatan"`
}

// TalanganRiwayat: log perpindahan status satu catatan ditambah riwayat cuci barangnya (hanya baca), urut menurut waktu.
func TalanganRiwayat(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var toko sql.NullInt64
		var stok int
		if err := db.QueryRow("SELECT toko_input_id, IFNULL(stok_id,0) FROM talangan_barang WHERE id = ?", id).Scan(&toko, &stok); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "data tidak ditemukan"})
			return
		}
		if cl.Role == "cabang" && (!toko.Valid || int(toko.Int64) != cl.LocationID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "data tidak ditemukan"})
			return
		}
		hasil := []kejadianRiwayat{}
		rows, err := db.Query(
			`SELECT s.dari, s.ke, IFNULL(u.username,''), s.waktu, IFNULL(s.catatan,'')
			 FROM talangan_status_log s LEFT JOIN users u ON u.id = s.oleh WHERE s.talangan_id = ? ORDER BY s.id`, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		for rows.Next() {
			b := kejadianRiwayat{Jenis: "status"}
			var w sql.NullTime
			if err := rows.Scan(&b.Dari, &b.Ke, &b.Oleh, &w, &b.Catatan); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			b.Waktu = w.Time
			hasil = append(hasil, b)
		}
		rows.Close()

		if stok > 0 {
			cuci, err := riwayatCuci(db, stok)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			hasil = append(hasil, cuci...)
			sort.SliceStable(hasil, func(i, j int) bool { return hasil[i].Waktu.Before(hasil[j].Waktu) })
		}
		c.JSON(http.StatusOK, hasil)
	}
}

// riwayatCuci: kejadian cuci satu barang stok. Antrean hanya tampil selama barang masih mengantre
// (barisnya dihapus alur cuci lama saat dikirim ke pencuci atau dibatalkan); sisanya dari gerak stok yang tidak dibatalkan.
func riwayatCuci(db *sql.DB, itemID int) ([]kejadianRiwayat, error) {
	var hasil []kejadianRiwayat
	rows, err := db.Query(
		`SELECT q.created_at, IFNULL(u.username,''), IFNULL(q.note,'') FROM washing_queue_items q
		 LEFT JOIN users u ON u.id = q.queued_by WHERE q.item_id = ?`, itemID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		b := kejadianRiwayat{Jenis: "cuci", Ke: "Masuk antrean cuci"}
		var w sql.NullTime
		if err := rows.Scan(&w, &b.Oleh, &b.Catatan); err != nil {
			rows.Close()
			return nil, err
		}
		b.Waktu = w.Time
		hasil = append(hasil, b)
	}
	rows.Close()

	rows, err = db.Query(
		`SELECT m.category, m.created_at, IFNULL(u.username,''), IFNULL(m.note,''), IFNULL(w.washer_name,'')
		 FROM stock_movements m
		 LEFT JOIN users u ON u.id = m.created_by
		 LEFT JOIN items i ON i.id = m.item_id
		 LEFT JOIN washing_jobs w ON w.id = i.washing_job_id
		 WHERE m.item_id = ? AND m.is_void = FALSE AND m.category IN ('cuci','cucian','batal_cuci')
		 ORDER BY m.id`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kat, pencuci string
		var w sql.NullTime
		b := kejadianRiwayat{Jenis: "cuci"}
		if err := rows.Scan(&kat, &w, &b.Oleh, &b.Catatan, &pencuci); err != nil {
			return nil, err
		}
		b.Waktu = w.Time
		switch kat {
		case "cuci":
			b.Ke = "Dikirim ke pencuci"
			if pencuci != "" {
				b.Catatan = "Pencuci: " + pencuci
			}
		case "cucian":
			b.Ke = "Selesai cuci, diterima back office"
		default:
			b.Ke = "Cuci dibatalkan, diterima back office"
		}
		hasil = append(hasil, b)
	}
	return hasil, rows.Err()
}
