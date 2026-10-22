package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

func ServiceCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if cl.Role == "pj" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
			return
		}
		var req models.ServiceReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		locID, tipe, ok := resolveLocation(c, db, req.LocationID)
		if !ok {
			return
		}
		if tipe != "cabang" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang service diterima di toko cabang"})
			return
		}
		cust := strings.TrimSpace(req.CustomerName)
		item := strings.TrimSpace(req.ItemName)
		keluhan := strings.TrimSpace(req.Complaint)
		if cust == "" || item == "" || keluhan == "" || req.WeightGram <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama customer, nama barang, berat, dan pekerjaan service wajib diisi"})
			return
		}
		karat := sql.NullString{}
		if strings.TrimSpace(req.Karat) != "" {
			k, ok := utils.NormalKarat(req.Karat)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "kadar harus 1K sampai 24K"})
				return
			}
			karat = sql.NullString{String: k, Valid: true}
		}
		tgl, ok := utils.ParseTanggal(c, req.ReceivedDate)
		if !ok {
			return
		}
		status := "di_toko"
		var kirim interface{}
		if req.Posisi == "bo" {
			status = "dikirim_bo"
			kirim = tgl
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(
			`INSERT INTO service_jobs (customer_name, customer_phone, item_name, karat, weight_gram, complaint, note,
			        origin_location_id, location_id, status, received_date, sent_bo_date, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			cust, utils.StrNull(strings.TrimSpace(req.CustomerPhone)), item, karat, req.WeightGram, keluhan,
			utils.StrNull(strings.TrimSpace(req.Note)), locID, locID, status, tgl, kirim, cl.UserID,
		)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		no := fmt.Sprintf("SRV-%05d", id)
		tx.Exec("UPDATE service_jobs SET job_no = ? WHERE id = ?", no, id)
		services.CatatAudit(tx, cl.UserID, "service_terima", "service_jobs", id, req)

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		pesan := fmt.Sprintf("service %s tercatat", no)
		if status == "dikirim_bo" {
			pesan += ", diantar ke back office (menunggu konfirmasi)"
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "job_no": no, "message": pesan})
	}
}

// ---------- DAFTAR ----------

func ServiceList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)

		q := `SELECT s.id, IFNULL(s.job_no,''), s.customer_name, IFNULL(s.customer_phone,''), s.item_name, IFNULL(s.karat,''),
		             s.weight_gram, s.complaint, IFNULL(s.note,''),
		             s.origin_location_id, o.name, s.location_id, l.name, IFNULL(s.pickup_location_id,0), IFNULL(p.name,''),
		             s.status, s.received_date, s.sent_bo_date, s.arrived_bo_date, s.belum_sampai, IFNULL(s.belum_sampai_note,''),
		             IFNULL(s.service_name,''), s.service_start_date, s.service_end_date, s.weight_after, IFNULL(s.service_note,''),
		             s.sent_toko_date, s.pickup_date, s.returned_bo_date, s.service_fee
		      FROM service_jobs s
		      JOIN locations o ON o.id = s.origin_location_id
		      JOIN locations l ON l.id = s.location_id
		      LEFT JOIN locations p ON p.id = s.pickup_location_id
		      WHERE s.is_void = FALSE`
		args := []interface{}{}

		if cl.Role == "cabang" {
			q += " AND (s.origin_location_id = ? OR s.pickup_location_id = ? OR s.location_id = ?)"
			args = append(args, cl.LocationID, cl.LocationID, cl.LocationID)
		}
		switch st := c.DefaultQuery("status", "aktif"); st {
		case "aktif":
			q += " AND s.status <> 'diambil'"
		case "semua":
		default:
			q += " AND s.status = ?"
			args = append(args, st)
		}
		if s := strings.TrimSpace(c.Query("q")); s != "" {
			like := "%" + s + "%"
			q += " AND (s.customer_name LIKE ? OR s.item_name LIKE ? OR IFNULL(s.job_no,'') LIKE ? OR IFNULL(s.service_name,'') LIKE ?)"
			args = append(args, like, like, like, like)
		}
		q += " ORDER BY s.id DESC LIMIT 300"

		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()

		out := []models.ServiceListRow{}
		for rows.Next() {
			var r models.ServiceListRow
			var d1, d2, d3, d4, d5, d6, d7 sql.NullTime
			var wa, fee sql.NullFloat64
			if err := rows.Scan(&r.ID, &r.JobNo, &r.CustomerName, &r.CustomerPhone, &r.ItemName, &r.Karat,
				&r.WeightGram, &r.Complaint, &r.Note,
				&r.OriginID, &r.Origin, &r.LocationID, &r.Location, &r.PickupID, &r.Pickup,
				&r.Status, &r.ReceivedDate, &d1, &d2, &r.BelumSampai, &r.BelumSampaiNote,
				&r.ServiceName, &d3, &d4, &wa, &r.ServiceNote, &d5, &d6, &d7, &fee); err != nil {
				utils.Gagal(c, err)
				return
			}
			r.SentBODate, r.ArrivedBODate = utils.PtrTime(d1), utils.PtrTime(d2)
			r.ServiceStart, r.ServiceEnd = utils.PtrTime(d3), utils.PtrTime(d4)
			r.SentTokoDate, r.PickupDate = utils.PtrTime(d5), utils.PtrTime(d6)
			r.ReturnedBODate = utils.PtrTime(d7)
			r.WeightAfter = utils.PtrFloat(wa)
			r.ServiceFee = utils.PtrFloat(fee)
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- PERPINDAHAN STATUS ----------

var aturanService = map[string]struct {
	Dari  string
	Peran []string
}{
	"kirim_bo":       {"di_toko", []string{"cabang", "owner", "super_admin"}},
	"terima_bo":      {"dikirim_bo", []string{"owner", "super_admin"}},
	"belum_sampai":   {"dikirim_bo", []string{"owner", "super_admin"}},
	"proses":         {"di_bo", []string{"pj", "super_admin"}},
	"selesai":        {"diproses", []string{"pj", "super_admin"}},
	"terima_kembali": {"kembali_bo", []string{"owner", "super_admin"}},
	"belum_kembali":  {"kembali_bo", []string{"owner", "super_admin"}},
	"oper":           {"selesai", []string{"owner", "super_admin"}},
	"terima_toko":    {"dikirim_toko", []string{"cabang"}},
	"diambil":        {"siap_diambil", []string{"cabang", "owner", "super_admin"}},
}

var pesanService = map[string]string{
	"kirim_bo":       "diantar ke back office, menunggu konfirmasi",
	"terima_bo":      "diterima di back office",
	"belum_sampai":   "ditandai belum sampai",
	"proses":         "dikirim ke penyervice",
	"selesai":        "selesai diservis, dikembalikan ke back office dan menunggu konfirmasi owner",
	"terima_kembali": "diterima di back office, siap dioper ke cabang",
	"belum_kembali":  "ditandai belum diterima di back office",
	"oper":           "dioper ke cabang, menunggu diterima",
	"terima_toko":    "diterima di toko, siap diambil customer",
	"diambil":        "sudah diambil customer",
}

func ServiceAksi(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		var req models.ServiceAksiRequest
		if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang service"})
			return
		}
		aturan, ada := aturanService[req.Action]
		if !ada {
			c.JSON(http.StatusBadRequest, gin.H{"error": "aksi tidak dikenal"})
			return
		}
		boleh := false
		for _, p := range aturan.Peran {
			if p == cl.Role {
				boleh = true
			}
		}
		if !boleh {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses untuk langkah ini"})
			return
		}
		if len(req.IDs) > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "maksimal 200 barang sekali proses"})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		namaSvc := strings.TrimSpace(req.ServiceName)
		note := strings.TrimSpace(req.Note)
		if req.Action == "proses" && namaSvc == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama penyervice wajib diisi"})
			return
		}
		if req.Action == "oper" {
			if _, ok := services.CekCabang(db, req.TargetLoc); !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "pilih cabang tempat customer mengambil"})
				return
			}
		}
		boID, err := services.IDBackOffice(db)
		if err != nil {
			utils.Gagal(c, err)
			return
		}

		uniq := map[int]bool{}
		args := []interface{}{}
		for _, id := range req.IDs {
			if !uniq[id] {
				uniq[id] = true
				args = append(args, id)
			}
		}

		// biaya service diisi manual per barang saat diambil customer (0 kalau tanpa biaya)
		if req.Action == "diambil" {
			for _, id := range req.IDs {
				fee, ada := req.Fees[id]
				if !ada || fee < 0 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "biaya service wajib diisi untuk setiap barang (isi 0 kalau tanpa biaya)"})
					return
				}
			}
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		rows, err := tx.Query(
			`SELECT id, status, location_id, origin_location_id, IFNULL(pickup_location_id,0), IFNULL(job_no,'')
			 FROM service_jobs WHERE is_void = FALSE AND id IN (`+utils.Placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		type jb struct {
			ID     int
			Status string
			Loc    int
			Origin int
			Pickup int
			JobNo  string
		}
		daftarJob := []jb{}
		for rows.Next() {
			var j jb
			if err := rows.Scan(&j.ID, &j.Status, &j.Loc, &j.Origin, &j.Pickup, &j.JobNo); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			daftarJob = append(daftarJob, j)
		}
		rows.Close()
		if len(daftarJob) != len(args) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang service yang tidak ditemukan"})
			return
		}

		for _, j := range daftarJob {
			if j.Status != aturan.Dari {
				c.JSON(http.StatusBadRequest, gin.H{"error": "status barang service tidak sesuai untuk langkah ini"})
				return
			}
			if cl.Role == "cabang" {
				sesuai := true
				switch req.Action {
				case "kirim_bo":
					sesuai = j.Loc == cl.LocationID && j.Origin == cl.LocationID
				case "terima_toko":
					sesuai = j.Pickup == cl.LocationID
				case "diambil":
					sesuai = j.Loc == cl.LocationID
				}
				if !sesuai {
					c.JSON(http.StatusForbidden, gin.H{"error": "bukan barang service di toko kamu"})
					return
				}
			}
		}

		// biaya service masuk ke kas lokasi tempat barang diambil (satu catatan kas per lokasi)
		type kasServis struct {
			Total float64
			IDs   []int
			Nos   []string
		}
		kasPerLokasi := map[int]*kasServis{}
		urutKas := []int{}
		var totalBiaya float64

		for _, j := range daftarJob {
			var err error
			switch req.Action {
			case "kirim_bo":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'dikirim_bo', sent_bo_date = ?, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?", tgl, j.ID)
			case "terima_bo":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'di_bo', location_id = ?, arrived_bo_date = ?, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?", boID, tgl, j.ID)
			case "belum_sampai":
				_, err = tx.Exec("UPDATE service_jobs SET belum_sampai = TRUE, belum_sampai_note = ? WHERE id = ?", utils.StrNull(note), j.ID)
			case "proses":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'diproses', service_name = ?, service_start_date = ? WHERE id = ?", namaSvc, tgl, j.ID)
			case "selesai":
				var wa interface{}
				if len(daftarJob) == 1 && req.WeightAfter > 0 {
					wa = req.WeightAfter
				}
				_, err = tx.Exec("UPDATE service_jobs SET status = 'kembali_bo', service_end_date = ?, weight_after = ?, service_note = ?, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?", tgl, wa, utils.StrNull(note), j.ID)
			case "terima_kembali":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'selesai', location_id = ?, returned_bo_date = ?, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?", boID, tgl, j.ID)
			case "belum_kembali":
				_, err = tx.Exec("UPDATE service_jobs SET belum_sampai = TRUE, belum_sampai_note = ? WHERE id = ?", utils.StrNull(note), j.ID)
			case "oper":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'dikirim_toko', pickup_location_id = ?, sent_toko_date = ? WHERE id = ?", req.TargetLoc, tgl, j.ID)
			case "terima_toko":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'siap_diambil', location_id = pickup_location_id WHERE id = ?", j.ID)
			case "diambil":
				fee := req.Fees[j.ID]
				_, err = tx.Exec("UPDATE service_jobs SET status = 'diambil', pickup_date = ?, service_fee = ? WHERE id = ?", tgl, fee, j.ID)
				if err == nil && fee > 0 {
					kj := kasPerLokasi[j.Loc]
					if kj == nil {
						kj = &kasServis{}
						kasPerLokasi[j.Loc] = kj
						urutKas = append(urutKas, j.Loc)
					}
					kj.Total += fee
					kj.IDs = append(kj.IDs, j.ID)
					kj.Nos = append(kj.Nos, j.JobNo)
					totalBiaya += fee
				}
			}
			if err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		for _, lokasi := range urutKas {
			kj := kasPerLokasi[lokasi]
			res, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
				 VALUES (?, 'in', 'biaya_service', ?, ?, ?, ?)`,
				lokasi, kj.Total, "Biaya service "+strings.Join(kj.Nos, ", "), tgl, cl.UserID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			kasID, _ := res.LastInsertId()
			for _, jobID := range kj.IDs {
				if _, err := tx.Exec("UPDATE service_jobs SET fee_cash_id = ? WHERE id = ?", kasID, jobID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			services.CatatAudit(tx, cl.UserID, "create_auto", "cash_transactions", kasID, gin.H{"biaya_service": len(kj.IDs), "total": kj.Total})
		}
		services.CatatAudit(tx, cl.UserID, "service_"+req.Action, "service_jobs", int64(daftarJob[0].ID), req)

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang service %s%s", len(daftarJob), pesanService[req.Action], pesanBiaya(totalBiaya))})
	}
}

// ---------- BATALKAN (Super Admin, sebelum diserahkan ke penyervice) ----------

func ServiceVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaSuperAdmin(c)
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

		var status string
		var isVoid bool
		err = tx.QueryRow("SELECT status, is_void FROM service_jobs WHERE id = ? FOR UPDATE", id).Scan(&status, &isVoid)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "barang service tidak ditemukan"})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sudah dibatalkan"})
			return
		}
		if status != "di_toko" && status != "dikirim_bo" && status != "di_bo" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang sudah diserahkan ke penyervice, tidak bisa dibatalkan"})
			return
		}
		tx.Exec("UPDATE service_jobs SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id)
		services.CatatAudit(tx, cl.UserID, "void", "service_jobs", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang service dibatalkan"})
	}
}

// ---------- ANGKA MERAH ----------

func ServiceNotif(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		n := 0
		switch cl.Role {
		case "owner", "super_admin":
			db.QueryRow("SELECT COUNT(*) FROM service_jobs WHERE is_void = FALSE AND status IN ('dikirim_bo','kembali_bo')").Scan(&n)
		case "pj":
			db.QueryRow("SELECT COUNT(*) FROM service_jobs WHERE is_void = FALSE AND status = 'di_bo'").Scan(&n)
		case "cabang":
			db.QueryRow(
				`SELECT COUNT(*) FROM service_jobs WHERE is_void = FALSE AND
				 ((status = 'dikirim_toko' AND pickup_location_id = ?) OR (status = 'siap_diambil' AND location_id = ?))`,
				cl.LocationID, cl.LocationID).Scan(&n)
		}
		c.JSON(http.StatusOK, gin.H{"badge": n})
	}
}

func ServiceRingkas(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)

		locRows, err := db.Query("SELECT id, name, type FROM locations ORDER BY id")
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer locRows.Close()

		m := map[int]*models.ServiceRingkasRow{}
		urut := []int{}
		boID := 0
		for locRows.Next() {
			var r models.ServiceRingkasRow
			if err := locRows.Scan(&r.LocationID, &r.Name, &r.Type); err != nil {
				utils.Gagal(c, err)
				return
			}
			if r.Type == "back_office" {
				boID = r.LocationID
			}
			if cl.Role == "cabang" && r.LocationID != cl.LocationID {
				continue
			}
			rr := r
			m[r.LocationID] = &rr
			urut = append(urut, r.LocationID)
		}

		// posisi barang service saat ini
		stRows, err := db.Query(`SELECT location_id, status, COUNT(*) FROM service_jobs
		                         WHERE is_void = FALSE AND status <> 'diambil' GROUP BY location_id, status`)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		for stRows.Next() {
			var loc, n int
			var st string
			if err := stRows.Scan(&loc, &st, &n); err != nil {
				stRows.Close()
				utils.Gagal(c, err)
				return
			}
			r := m[loc]
			if r == nil {
				continue
			}
			switch st {
			case "di_toko":
				r.DiToko = n
			case "di_bo":
				r.MenungguServis = n
			case "diproses":
				r.Diproses = n
			case "kembali_bo":
				r.KembaliBO = n
			case "selesai":
				r.Selesai = n
			case "siap_diambil":
				r.SiapDiambil = n
			case "dikirim_bo", "dikirim_toko":
				r.Keluar += n
			}
		}
		stRows.Close()

		// barang yang sedang menuju lokasi
		if r := m[boID]; r != nil {
			var n int
			db.QueryRow("SELECT COUNT(*) FROM service_jobs WHERE is_void = FALSE AND status = 'dikirim_bo'").Scan(&n)
			r.Masuk += n
		}
		inRows, err := db.Query(`SELECT pickup_location_id, COUNT(*) FROM service_jobs
		                         WHERE is_void = FALSE AND status = 'dikirim_toko' AND pickup_location_id IS NOT NULL
		                         GROUP BY pickup_location_id`)
		if err == nil {
			for inRows.Next() {
				var loc, n int
				if inRows.Scan(&loc, &n) == nil {
					if r := m[loc]; r != nil {
						r.Masuk += n
					}
				}
			}
			inRows.Close()
		}

		out := []*models.ServiceRingkasRow{}
		for _, id := range urut {
			r := m[id]
			r.DiSini = r.DiToko + r.MenungguServis + r.Diproses + r.KembaliBO + r.Selesai + r.SiapDiambil
			out = append(out, r)
		}
		c.JSON(http.StatusOK, out)
	}
}

// pesanBiaya melengkapi pesan sukses bila ada biaya service yang masuk kas.
func pesanBiaya(total float64) string {
	if total <= 0 {
		return ""
	}
	return ", biaya service " + utils.RupiahTeks(total) + " masuk ke kas toko"
}
