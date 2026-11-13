package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// Catatan talangan dan tukar barang dengan BP (pihak luar tanpa akun).
// Pegawai toko (cabang) menginput dan mengajukan tukar; owner memantau dan menyetujui, baru kas back office tercatat.

type TalanganReq struct {
	BPNama     string  `json:"bp_nama" binding:"max=30"`
	Arah       string  `json:"arah" binding:"max=12"` // kita_di_bp | bp_di_kita
	ItemName   string  `json:"item_name" binding:"max=100"`
	WeightGram float64 `json:"weight_gram" binding:"gte=0,lte=100000"`
	Karat      string  `json:"karat" binding:"max=10"`
	Nilai      float64 `json:"nilai" binding:"gte=0,lte=1000000000000"`
	EntryDate  string  `json:"entry_date" binding:"max=10"`
	Note       string  `json:"note" binding:"max=200"`
	LocationID int     `json:"location_id" binding:"gte=0"` // hanya dipakai super admin
}

// TalanganBatchReq: beberapa barang sekaligus untuk satu BP, arah, dan tanggal yang sama.
type TalanganBatchReq struct {
	BPNama     string `json:"bp_nama" binding:"max=30"`
	Arah       string `json:"arah" binding:"max=12"`
	EntryDate  string `json:"entry_date" binding:"max=10"`
	Note       string `json:"note" binding:"max=200"`
	LocationID int    `json:"location_id" binding:"gte=0"`
	Items      []struct {
		ItemName   string  `json:"item_name" binding:"max=100"`
		WeightGram float64 `json:"weight_gram" binding:"gte=0,lte=100000"`
		Karat      string  `json:"karat" binding:"max=10"`
		Nilai      float64 `json:"nilai" binding:"gte=0,lte=1000000000000"`
		Note       string  `json:"note" binding:"max=200"`
	} `json:"items" binding:"max=100,dive"`
}

type TalanganKonfirmasiReq struct {
	Konfirmasi bool   `json:"konfirmasi"`
	Note       string `json:"note" binding:"max=200"`
	Date       string `json:"date" binding:"max=10"`
}

type TukarReq struct {
	BPNama      string `json:"bp_nama" binding:"max=30"`
	TalanganIDs []int  `json:"talangan_ids" binding:"max=1000,dive,gt=0"`
	TukarDate   string `json:"tukar_date" binding:"max=10"`
	Note        string `json:"note" binding:"max=200"`
	LocationID  int    `json:"location_id" binding:"gte=0"`
}

// lokasiTulis: toko pemilik catatan. Cabang selalu tokonya sendiri; super admin menyebut location_id.
func lokasiTulis(c *gin.Context, cl *models.Claims, dariBody int) (int, bool) {
	if cl.Role == "cabang" {
		return cl.LocationID, true
	}
	if dariBody <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pilih toko"})
		return 0, false
	}
	return dariBody, true
}

func (r *TalanganReq) rapikan(c *gin.Context) (string, bool) {
	r.ItemName = strings.TrimSpace(r.ItemName)
	r.Karat = strings.TrimSpace(r.Karat)
	r.Note = strings.TrimSpace(r.Note)
	if !services.BPValid(r.BPNama) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pilih BP"})
		return "", false
	}
	if r.Arah != "kita_di_bp" && r.Arah != "bp_di_kita" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pilih arah barang"})
		return "", false
	}
	if r.ItemName == "" || r.Nilai <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama barang dan nilai wajib diisi"})
		return "", false
	}
	if r.Karat != "" {
		k, ok := utils.NormalKarat(r.Karat)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "kadar harus 1K sampai 24K"})
			return "", false
		}
		r.Karat = k
	}
	return utils.ParseTanggal(c, r.EntryDate)
}

// TalanganList: catatan + rekap selisih per BP. Cabang melihat tokonya, owner dan super admin semua toko.
func TalanganList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		q := `SELECT t.id, t.location_id, l.name, t.bp_nama, t.arah, t.item_name, t.weight_gram, IFNULL(t.karat,''), t.nilai,
		             DATE_FORMAT(t.entry_date,'%Y-%m-%d'), IFNULL(t.note,''), t.konfirmasi, IFNULL(t.konfirmasi_note,''),
		             IFNULL(DATE_FORMAT(t.konfirmasi_tgl,'%Y-%m-%d'),''), t.status, IFNULL(t.tukar_id,0), IFNULL(t.batal_alasan,''),
		             IFNULL(u.username,'')
		      FROM talangan t JOIN locations l ON l.id = t.location_id LEFT JOIN users u ON u.id = t.created_by
		      WHERE 1=1`
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND t.location_id = ?"
			args = append(args, cl.LocationID)
		}
		if bp := c.Query("bp"); bp != "" {
			q += " AND t.bp_nama = ?"
			args = append(args, bp)
		}
		switch s := c.Query("status"); s {
		case "semua":
		case "aktif", "tukar", "batal":
			q += " AND t.status = ?"
			args = append(args, s)
		default:
			q += " AND t.status IN ('aktif','tukar')"
		}
		q += " ORDER BY t.id DESC LIMIT 500"
		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []gin.H{}
		for rows.Next() {
			var id, loc, tukarID int
			var toko, bp, arah, nama, karat, tgl, note, kNote, kTgl, status, alasan, oleh string
			var berat, nilai float64
			var konf bool
			if err := rows.Scan(&id, &loc, &toko, &bp, &arah, &nama, &berat, &karat, &nilai, &tgl, &note, &konf, &kNote, &kTgl,
				&status, &tukarID, &alasan, &oleh); err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, gin.H{"id": id, "location_id": loc, "toko": toko, "bp_nama": bp, "arah": arah, "item_name": nama,
				"weight_gram": berat, "karat": karat, "nilai": nilai, "entry_date": tgl, "note": note, "konfirmasi": konf,
				"konfirmasi_note": kNote, "konfirmasi_tgl": kTgl, "status": status, "tukar_id": tukarID, "batal_alasan": alasan, "oleh": oleh})
		}

		// rekap: catatan aktif (belum masuk pengajuan tukar) per BP
		rq := `SELECT bp_nama, IFNULL(SUM(IF(arah='kita_di_bp', nilai, 0)),0), SUM(arah='kita_di_bp'),
		              IFNULL(SUM(IF(arah='bp_di_kita', nilai, 0)),0), SUM(arah='bp_di_kita')
		       FROM talangan WHERE status = 'aktif'`
		rargs := []interface{}{}
		if cl.Role == "cabang" {
			rq += " AND location_id = ?"
			rargs = append(rargs, cl.LocationID)
		}
		rq += " GROUP BY bp_nama"
		rrows, err := db.Query(rq, rargs...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rrows.Close()
		rekap := map[string]gin.H{}
		for rrows.Next() {
			var bp string
			var kita, bpv float64
			var nKita, nBP sql.NullInt64
			if err := rrows.Scan(&bp, &kita, &nKita, &bpv, &nBP); err != nil {
				utils.Gagal(c, err)
				return
			}
			rekap[bp] = gin.H{"bp_nama": bp, "total_kita": kita, "n_kita": nKita.Int64, "total_bp": bpv, "n_bp": nBP.Int64, "selisih": bpv - kita}
		}
		rekapList := []gin.H{}
		for _, bp := range services.DaftarBP {
			if r, ada := rekap[bp]; ada {
				rekapList = append(rekapList, r)
			} else {
				rekapList = append(rekapList, gin.H{"bp_nama": bp, "total_kita": 0, "n_kita": 0, "total_bp": 0, "n_bp": 0, "selisih": 0})
			}
		}
		c.JSON(http.StatusOK, gin.H{"rows": out, "rekap": rekapList, "daftar_bp": services.DaftarBP})
	}
}

func TalanganCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "super_admin")
		if !ok {
			return
		}
		var req TalanganBatchReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.Items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "isi minimal satu barang"})
			return
		}
		loc, ok := lokasiTulis(c, cl, req.LocationID)
		if !ok {
			return
		}
		note := strings.TrimSpace(req.Note)
		// tiap barang diperiksa dengan aturan yang sama seperti edit satu barang
		baris := make([]TalanganReq, len(req.Items))
		var tgl string
		for i, it := range req.Items {
			t := TalanganReq{BPNama: req.BPNama, Arah: req.Arah, ItemName: it.ItemName, WeightGram: it.WeightGram,
				Karat: it.Karat, Nilai: it.Nilai, EntryDate: req.EntryDate, Note: strings.TrimSpace(it.Note)}
			if t.Note == "" {
				t.Note = note
			}
			d, ok := t.rapikan(c)
			if !ok {
				return
			}
			tgl, baris[i] = d, t
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		for _, t := range baris {
			res, err := tx.Exec(
				`INSERT INTO talangan (location_id, bp_nama, arah, item_name, weight_gram, karat, nilai, entry_date, note, created_by)
				 VALUES (?,?,?,?,?,?,?,?,?,?)`,
				loc, t.BPNama, t.Arah, t.ItemName, t.WeightGram, utils.StrNull(t.Karat), t.Nilai, tgl, utils.StrNull(t.Note), cl.UserID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			id, _ := res.LastInsertId()
			services.CatatAudit(tx, middleware.Meta(c), "talangan_baru", "talangan", id, t)
		}
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("%d catatan talangan disimpan", len(baris))})
	}
}

// muatTalangan mengunci satu catatan dan memeriksa hak tulis toko pemiliknya.
func muatTalangan(c *gin.Context, tx *sql.Tx, cl *models.Claims, id int) (loc int, status, bp string, ok bool) {
	if err := tx.QueryRow("SELECT location_id, status, bp_nama FROM talangan WHERE id = ? FOR UPDATE", id).Scan(&loc, &status, &bp); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "catatan tidak ditemukan"})
		return 0, "", "", false
	}
	if cl.Role == "cabang" && loc != cl.LocationID {
		c.JSON(http.StatusForbidden, gin.H{"error": "bukan catatan toko kamu"})
		return 0, "", "", false
	}
	return loc, status, bp, true
}

func TalanganEdit(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req TalanganReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		tgl, ok := req.rapikan(c)
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		_, status, _, ok := muatTalangan(c, tx, cl, id)
		if !ok {
			return
		}
		if status != "aktif" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "catatan yang sudah masuk pengajuan tukar atau dibatalkan tidak bisa diedit"})
			return
		}
		var lama TalanganReq
		var tglLama string
		tx.QueryRow(`SELECT bp_nama, arah, item_name, weight_gram, IFNULL(karat,''), nilai, DATE_FORMAT(entry_date,'%Y-%m-%d'), IFNULL(note,'')
		             FROM talangan WHERE id = ?`, id).Scan(&lama.BPNama, &lama.Arah, &lama.ItemName, &lama.WeightGram, &lama.Karat, &lama.Nilai, &tglLama, &lama.Note)
		lama.EntryDate = tglLama
		// edit membatalkan konfirmasi telepon: nilai berubah berarti perlu dikonfirmasi ulang
		if _, err := tx.Exec(
			`UPDATE talangan SET bp_nama = ?, arah = ?, item_name = ?, weight_gram = ?, karat = ?, nilai = ?, entry_date = ?, note = ?,
			        konfirmasi = FALSE, konfirmasi_note = NULL, konfirmasi_tgl = NULL WHERE id = ?`,
			req.BPNama, req.Arah, req.ItemName, req.WeightGram, utils.StrNull(req.Karat), req.Nilai, tgl, utils.StrNull(req.Note), id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "talangan_edit", "talangan", int64(id), lama, req)
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "catatan talangan diperbarui, konfirmasi telepon perlu diulang"})
	}
}

func TalanganKonfirmasi(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req TalanganKonfirmasiReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		_, status, _, ok := muatTalangan(c, tx, cl, id)
		if !ok {
			return
		}
		if status != "aktif" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "catatan ini sudah tidak aktif"})
			return
		}
		note := strings.TrimSpace(req.Note)
		if req.Konfirmasi {
			_, err = tx.Exec("UPDATE talangan SET konfirmasi = TRUE, konfirmasi_note = ?, konfirmasi_tgl = ? WHERE id = ?", utils.StrNull(note), tgl, id)
		} else {
			_, err = tx.Exec("UPDATE talangan SET konfirmasi = FALSE, konfirmasi_note = NULL, konfirmasi_tgl = NULL WHERE id = ?", id)
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "talangan_konfirmasi", "talangan", int64(id), req)
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "konfirmasi telepon dicatat"})
	}
}

func TalanganBatal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		alasan, ok := utils.BacaAlasan(c)
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		_, status, _, ok := muatTalangan(c, tx, cl, id)
		if !ok {
			return
		}
		if status != "aktif" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "batalkan pengajuan tukarnya dulu bila catatan ini sudah diajukan"})
			return
		}
		if _, err := tx.Exec("UPDATE talangan SET status = 'batal', batal_alasan = ? WHERE id = ?", alasan, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "talangan_batal", "talangan", int64(id), gin.H{"reason": alasan})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "catatan talangan dibatalkan"})
	}
}

// ---------- TUKAR BARANG DENGAN BP ----------

func TukarList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		q := `SELECT x.id, x.bp_nama, l.name, DATE_FORMAT(x.tukar_date,'%Y-%m-%d'), x.total_kita, x.total_bp, x.selisih, x.status,
		             IFNULL(x.cash_id,0), IFNULL(x.note,''), IFNULL(x.batal_alasan,''), IFNULL(u.username,''), IFNULL(a.username,''),
		             (SELECT COUNT(*) FROM talangan t WHERE t.tukar_id = x.id)
		      FROM tukar_bp x JOIN locations l ON l.id = x.location_id
		      LEFT JOIN users u ON u.id = x.created_by LEFT JOIN users a ON a.id = x.approved_by
		      WHERE 1=1`
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND x.location_id = ?"
			args = append(args, cl.LocationID)
		}
		q += " ORDER BY x.id DESC LIMIT 200"
		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var id, cashID, n int
			var bp, toko, tgl, status, note, alasan, oleh, setuju string
			var kita, bpv, selisih float64
			if err := rows.Scan(&id, &bp, &toko, &tgl, &kita, &bpv, &selisih, &status, &cashID, &note, &alasan, &oleh, &setuju, &n); err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, gin.H{"id": id, "bp_nama": bp, "toko": toko, "tukar_date": tgl, "total_kita": kita, "total_bp": bpv,
				"selisih": selisih, "status": status, "cash_id": cashID, "note": note, "batal_alasan": alasan, "oleh": oleh,
				"disetujui_oleh": setuju, "jumlah": n})
		}
		c.JSON(http.StatusOK, out)
	}
}

func TukarAjukan(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "super_admin")
		if !ok {
			return
		}
		var req TukarReq
		if err := c.ShouldBindJSON(&req); err != nil || len(req.TalanganIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu catatan"})
			return
		}
		if !services.BPValid(req.BPNama) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih BP"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.TukarDate)
		if !ok {
			return
		}
		loc, ok := lokasiTulis(c, cl, req.LocationID)
		if !ok {
			return
		}
		uniq := map[int]bool{}
		args := []interface{}{}
		for _, id := range req.TalanganIDs {
			if !uniq[id] {
				uniq[id] = true
				args = append(args, id)
			}
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		rows, err := tx.Query(
			`SELECT id, item_name, arah, nilai, status, bp_nama, location_id, konfirmasi FROM talangan
			 WHERE id IN (`+utils.Placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		var kita, bpv float64
		n := 0
		for rows.Next() {
			var id, l int
			var nama, arah, status, bp string
			var nilai float64
			var konf bool
			if err := rows.Scan(&id, &nama, &arah, &nilai, &status, &bp, &l, &konf); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			n++
			switch {
			case status != "aktif":
				rows.Close()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sudah masuk pengajuan tukar atau dibatalkan", nama)})
				return
			case bp != req.BPNama:
				rows.Close()
				c.JSON(http.StatusBadRequest, gin.H{"error": "semua catatan harus untuk BP yang sama"})
				return
			case l != loc:
				rows.Close()
				c.JSON(http.StatusForbidden, gin.H{"error": "ada catatan yang bukan milik toko ini"})
				return
			case !konf:
				rows.Close()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s belum dikonfirmasi lewat telepon", nama)})
				return
			}
			if arah == "kita_di_bp" {
				kita += nilai
			} else {
				bpv += nilai
			}
		}
		rows.Close()
		if n != len(args) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ada catatan yang tidak ditemukan"})
			return
		}
		res, err := tx.Exec(
			`INSERT INTO tukar_bp (bp_nama, location_id, tukar_date, total_kita, total_bp, selisih, note, created_by)
			 VALUES (?,?,?,?,?,?,?,?)`,
			req.BPNama, loc, tgl, kita, bpv, bpv-kita, utils.StrNull(strings.TrimSpace(req.Note)), cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		tukarID, _ := res.LastInsertId()
		if _, err := tx.Exec("UPDATE talangan SET status = 'tukar', tukar_id = ? WHERE id IN ("+utils.Placeholders(len(args))+")",
			append([]interface{}{tukarID}, args...)...); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "tukar_ajukan", "tukar_bp", tukarID, gin.H{"bp": req.BPNama, "kita": kita, "bp_di_kita": bpv, "selisih": bpv - kita})
		for _, a := range args {
			services.CatatAudit(tx, middleware.Meta(c), "tukar_ajukan", "talangan", int64(a.(int)), gin.H{"tukar_id": tukarID})
		}
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"message": fmt.Sprintf("pengajuan tukar dengan %s dikirim ke owner, selisih %s", req.BPNama, services.FmtRp(bpv-kita))})
	}
}

// TukarSetujui: owner menyetujui, selisih masuk kas back office (kita bayar bila selisih positif, BP bayar bila negatif).
func TukarSetujui(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaAdminOwner(c)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		boID, err := services.IDBackOffice(db)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		var status, bp, tgl string
		var selisih float64
		if err := tx.QueryRow("SELECT status, bp_nama, DATE_FORMAT(tukar_date,'%Y-%m-%d'), selisih FROM tukar_bp WHERE id = ? FOR UPDATE", id).
			Scan(&status, &bp, &tgl, &selisih); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pengajuan tidak ditemukan"})
			return
		}
		if status != "diajukan" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pengajuan ini sudah diproses"})
			return
		}
		var cashID sql.NullInt64
		if selisih != 0 {
			arah, jumlah, catat := "out", selisih, fmt.Sprintf("Tukar barang dengan %s: kita bayar selisih", bp)
			if selisih < 0 {
				arah, jumlah, catat = "in", -selisih, fmt.Sprintf("Tukar barang dengan %s: %s bayar selisih", bp, bp)
			}
			res, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by, metode)
				 VALUES (?,?,?,?,?,?,?,?)`,
				boID, arah, "kembali_cabang", jumlah, catat, tgl, cl.UserID, services.MetodeTunai)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			kid, _ := res.LastInsertId()
			cashID = sql.NullInt64{Int64: kid, Valid: true}
		}
		if _, err := tx.Exec("UPDATE tukar_bp SET status = 'disetujui', cash_id = ?, approved_by = ? WHERE id = ?", cashID, cl.UserID, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "tukar_setujui", "tukar_bp", int64(id), gin.H{"bp": bp, "selisih": selisih})
		for _, tid := range idCatatanTukar(tx, id) {
			services.CatatAudit(tx, middleware.Meta(c), "tukar_setujui", "talangan", int64(tid), gin.H{"tukar_id": id, "selisih": selisih})
		}
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "tukar disetujui dan tercatat di kas back office"})
	}
}

// TukarBatal: toko membatalkan pengajuan yang belum disetujui; owner bisa membatalkan juga yang sudah masuk kas (kas dikembalikan).
// Catatan talangan dilepas kembali jadi aktif supaya bisa diedit lalu diajukan ulang.
func TukarBatal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		alasan, ok := utils.BacaAlasan(c)
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		var status string
		var loc int
		var cashID sql.NullInt64
		if err := tx.QueryRow("SELECT status, location_id, cash_id FROM tukar_bp WHERE id = ? FOR UPDATE", id).Scan(&status, &loc, &cashID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "pengajuan tidak ditemukan"})
			return
		}
		if status == "batal" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}
		if cl.Role == "cabang" && (loc != cl.LocationID || status != "diajukan") {
			c.JSON(http.StatusForbidden, gin.H{"error": "toko hanya bisa membatalkan pengajuan miliknya yang belum disetujui owner"})
			return
		}
		if cashID.Valid {
			tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", alasan, cashID.Int64)
			services.CatatAudit(tx, middleware.Meta(c), "void", "cash_transactions", cashID.Int64, gin.H{"reason": alasan})
		}
		for _, tid := range idCatatanTukar(tx, id) {
			services.CatatAudit(tx, middleware.Meta(c), "tukar_batal", "talangan", int64(tid), gin.H{"tukar_id": id, "reason": alasan})
		}
		if _, err := tx.Exec("UPDATE talangan SET status = 'aktif', tukar_id = NULL WHERE tukar_id = ?", id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec("UPDATE tukar_bp SET status = 'batal', batal_alasan = ? WHERE id = ?", alasan, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "tukar_batal", "tukar_bp", int64(id), gin.H{"reason": alasan, "oleh": cl.Role})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "pengajuan tukar dibatalkan, catatan talangan aktif kembali"})
	}
}

func idCatatanTukar(tx *sql.Tx, tukarID int) []int {
	out := []int{}
	rows, err := tx.Query("SELECT id FROM talangan WHERE tukar_id = ?", tukarID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if rows.Scan(&id) == nil {
			out = append(out, id)
		}
	}
	return out
}

// ---------- RIWAYAT (dari jejak audit): bisa dilihat toko pemilik, owner, dan super admin ----------

func riwayat(db *sql.DB, tabelAudit, tabelData string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "cabang", "owner", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var loc int
		if err := db.QueryRow("SELECT location_id FROM "+tabelData+" WHERE id = ?", id).Scan(&loc); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "data tidak ditemukan"})
			return
		}
		if cl.Role == "cabang" && loc != cl.LocationID {
			c.JSON(http.StatusForbidden, gin.H{"error": "bukan data toko kamu"})
			return
		}
		rows, err := db.Query(
			`SELECT DATE_FORMAT(a.created_at,'%Y-%m-%d %H:%i'), IFNULL(u.username,''), IFNULL(u.role,''), a.action, IFNULL(a.old_value,''), IFNULL(a.new_value,'')
			 FROM audit_logs a LEFT JOIN users u ON u.id = a.user_id
			 WHERE a.table_name = ? AND a.record_id = ? ORDER BY a.id`, tabelAudit, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var waktu, user, role, aksi, lama, baru string
			if err := rows.Scan(&waktu, &user, &role, &aksi, &lama, &baru); err != nil {
				utils.Gagal(c, err)
				return
			}
			e := gin.H{"waktu": waktu, "user": user, "role": role, "aksi": aksi}
			if lama != "" {
				e["lama"] = json.RawMessage(lama)
			}
			if baru != "" {
				e["baru"] = json.RawMessage(baru)
			}
			out = append(out, e)
		}
		c.JSON(http.StatusOK, out)
	}
}

func TalanganRiwayat(db *sql.DB) gin.HandlerFunc { return riwayat(db, "talangan", "talangan") }
func TukarRiwayat(db *sql.DB) gin.HandlerFunc    { return riwayat(db, "tukar_bp", "tukar_bp") }
