package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func strNull(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// ---------- TERIMA BARANG SERVICE DI TOKO ----------

type serviceReq struct {
	LocationID    int     `json:"location_id"`
	CustomerName  string  `json:"customer_name"`
	CustomerPhone string  `json:"customer_phone"`
	ItemName      string  `json:"item_name"`
	Karat         string  `json:"karat"`
	WeightGram    float64 `json:"weight_gram"`
	Complaint     string  `json:"complaint"`
	Note          string  `json:"note"`
	ReceivedDate  string  `json:"received_date"`
	Posisi        string  `json:"posisi"`
}

func serviceCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		if cl.Role == "pj" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
			return
		}
		var req serviceReq
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
			k, ok := normalKarat(req.Karat)
			if !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "kadar harus 1K sampai 24K"})
				return
			}
			karat = sql.NullString{String: k, Valid: true}
		}
		tgl, ok := parseTanggal(c, req.ReceivedDate)
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
			gagal(c, err)
			return
		}
		defer tx.Rollback()

		res, err := tx.Exec(
			`INSERT INTO service_jobs (customer_name, customer_phone, item_name, karat, weight_gram, complaint, note,
			        origin_location_id, location_id, status, received_date, sent_bo_date, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			cust, strNull(strings.TrimSpace(req.CustomerPhone)), item, karat, req.WeightGram, keluhan,
			strNull(strings.TrimSpace(req.Note)), locID, locID, status, tgl, kirim, cl.UserID,
		)
		if err != nil {
			gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		no := fmt.Sprintf("SRV-%05d", id)
		tx.Exec("UPDATE service_jobs SET job_no = ? WHERE id = ?", no, id)
		catatAudit(tx, cl.UserID, "service_terima", "service_jobs", id, req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
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

func serviceList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)

		q := `SELECT s.id, IFNULL(s.job_no,''), s.customer_name, IFNULL(s.customer_phone,''), s.item_name, IFNULL(s.karat,''),
		             s.weight_gram, s.complaint, IFNULL(s.note,''),
		             s.origin_location_id, o.name, s.location_id, l.name, IFNULL(s.pickup_location_id,0), IFNULL(p.name,''),
		             s.status, s.received_date, s.sent_bo_date, s.arrived_bo_date, s.belum_sampai, IFNULL(s.belum_sampai_note,''),
		             IFNULL(s.service_name,''), s.service_start_date, s.service_end_date, s.weight_after, IFNULL(s.service_note,''),
		             s.sent_toko_date, s.pickup_date
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
			gagal(c, err)
			return
		}
		defer rows.Close()

		type row struct {
			ID              int        `json:"id"`
			JobNo           string     `json:"job_no"`
			CustomerName    string     `json:"customer_name"`
			CustomerPhone   string     `json:"customer_phone"`
			ItemName        string     `json:"item_name"`
			Karat           string     `json:"karat"`
			WeightGram      float64    `json:"weight_gram"`
			Complaint       string     `json:"complaint"`
			Note            string     `json:"note"`
			OriginID        int        `json:"origin_id"`
			Origin          string     `json:"origin"`
			LocationID      int        `json:"location_id"`
			Location        string     `json:"location"`
			PickupID        int        `json:"pickup_id"`
			Pickup          string     `json:"pickup"`
			Status          string     `json:"status"`
			ReceivedDate    time.Time  `json:"received_date"`
			SentBODate      *time.Time `json:"sent_bo_date"`
			ArrivedBODate   *time.Time `json:"arrived_bo_date"`
			BelumSampai     bool       `json:"belum_sampai"`
			BelumSampaiNote string     `json:"belum_sampai_note"`
			ServiceName     string     `json:"service_name"`
			ServiceStart    *time.Time `json:"service_start_date"`
			ServiceEnd      *time.Time `json:"service_end_date"`
			WeightAfter     *float64   `json:"weight_after"`
			ServiceNote     string     `json:"service_note"`
			SentTokoDate    *time.Time `json:"sent_toko_date"`
			PickupDate      *time.Time `json:"pickup_date"`
		}
		out := []row{}
		for rows.Next() {
			var r row
			var d1, d2, d3, d4, d5, d6 sql.NullTime
			var wa sql.NullFloat64
			if err := rows.Scan(&r.ID, &r.JobNo, &r.CustomerName, &r.CustomerPhone, &r.ItemName, &r.Karat,
				&r.WeightGram, &r.Complaint, &r.Note,
				&r.OriginID, &r.Origin, &r.LocationID, &r.Location, &r.PickupID, &r.Pickup,
				&r.Status, &r.ReceivedDate, &d1, &d2, &r.BelumSampai, &r.BelumSampaiNote,
				&r.ServiceName, &d3, &d4, &wa, &r.ServiceNote, &d5, &d6); err != nil {
				gagal(c, err)
				return
			}
			r.SentBODate, r.ArrivedBODate = ptrTime(d1), ptrTime(d2)
			r.ServiceStart, r.ServiceEnd = ptrTime(d3), ptrTime(d4)
			r.SentTokoDate, r.PickupDate = ptrTime(d5), ptrTime(d6)
			r.WeightAfter = ptrFloat(wa)
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
	"kirim_bo":     {"di_toko", []string{"cabang", "owner", "super_admin"}},
	"terima_bo":    {"dikirim_bo", []string{"owner", "super_admin"}},
	"belum_sampai": {"dikirim_bo", []string{"owner", "super_admin"}},
	"proses":       {"di_bo", []string{"pj", "super_admin"}},
	"selesai":      {"diproses", []string{"pj", "super_admin"}},
	"oper":         {"selesai", []string{"owner", "super_admin"}},
	"terima_toko":  {"dikirim_toko", []string{"cabang", "owner", "super_admin"}},
	"diambil":      {"siap_diambil", []string{"cabang", "owner", "super_admin"}},
}

var pesanService = map[string]string{
	"kirim_bo":     "diantar ke back office, menunggu konfirmasi",
	"terima_bo":    "diterima di back office",
	"belum_sampai": "ditandai belum sampai",
	"proses":       "dikirim ke penyervice",
	"selesai":      "selesai diservis, kembali di back office",
	"oper":         "dioper ke cabang, menunggu diterima",
	"terima_toko":  "diterima di toko, siap diambil customer",
	"diambil":      "sudah diambil customer",
}

func serviceAksi(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		var req struct {
			Action      string  `json:"action"`
			IDs         []int   `json:"ids"`
			Date        string  `json:"date"`
			ServiceName string  `json:"service_name"`
			TargetLoc   int     `json:"target_location_id"`
			WeightAfter float64 `json:"weight_after"`
			Note        string  `json:"note"`
		}
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
		tgl, ok := parseTanggal(c, req.Date)
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
			if _, ok := cekCabang(db, req.TargetLoc); !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "pilih cabang tempat customer mengambil (Kepanjen atau Mall Dinoyo)"})
				return
			}
		}
		boID, err := idBackOffice(db)
		if err != nil {
			gagal(c, err)
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

		tx, err := db.Begin()
		if err != nil {
			gagal(c, err)
			return
		}
		defer tx.Rollback()

		rows, err := tx.Query(
			`SELECT id, status, location_id, origin_location_id, IFNULL(pickup_location_id,0)
			 FROM service_jobs WHERE is_void = FALSE AND id IN (`+placeholders(len(args))+`) FOR UPDATE`, args...)
		if err != nil {
			gagal(c, err)
			return
		}
		type jb struct {
			ID     int
			Status string
			Loc    int
			Origin int
			Pickup int
		}
		daftarJob := []jb{}
		for rows.Next() {
			var j jb
			if err := rows.Scan(&j.ID, &j.Status, &j.Loc, &j.Origin, &j.Pickup); err != nil {
				rows.Close()
				gagal(c, err)
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

		for _, j := range daftarJob {
			var err error
			switch req.Action {
			case "kirim_bo":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'dikirim_bo', sent_bo_date = ?, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?", tgl, j.ID)
			case "terima_bo":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'di_bo', location_id = ?, arrived_bo_date = ?, belum_sampai = FALSE, belum_sampai_note = NULL WHERE id = ?", boID, tgl, j.ID)
			case "belum_sampai":
				_, err = tx.Exec("UPDATE service_jobs SET belum_sampai = TRUE, belum_sampai_note = ? WHERE id = ?", strNull(note), j.ID)
			case "proses":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'diproses', service_name = ?, service_start_date = ? WHERE id = ?", namaSvc, tgl, j.ID)
			case "selesai":
				var wa interface{}
				if len(daftarJob) == 1 && req.WeightAfter > 0 {
					wa = req.WeightAfter
				}
				_, err = tx.Exec("UPDATE service_jobs SET status = 'selesai', service_end_date = ?, weight_after = ?, service_note = ? WHERE id = ?", tgl, wa, strNull(note), j.ID)
			case "oper":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'dikirim_toko', pickup_location_id = ?, sent_toko_date = ? WHERE id = ?", req.TargetLoc, tgl, j.ID)
			case "terima_toko":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'siap_diambil', location_id = pickup_location_id WHERE id = ?", j.ID)
			case "diambil":
				_, err = tx.Exec("UPDATE service_jobs SET status = 'diambil', pickup_date = ? WHERE id = ?", tgl, j.ID)
			}
			if err != nil {
				gagal(c, err)
				return
			}
		}
		catatAudit(tx, cl.UserID, "service_"+req.Action, "service_jobs", int64(daftarJob[0].ID), req)

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d barang service %s", len(daftarJob), pesanService[req.Action])})
	}
}

// ---------- BATALKAN (Super Admin, sebelum diserahkan ke penyervice) ----------

func serviceVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := hanyaSuperAdmin(c)
		if !ok {
			return
		}
		id, ok := idParam(c)
		if !ok {
			return
		}
		reason, ok := bacaAlasan(c)
		if !ok {
			return
		}

		tx, err := db.Begin()
		if err != nil {
			gagal(c, err)
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
		catatAudit(tx, cl.UserID, "void", "service_jobs", int64(id), gin.H{"reason": reason})

		if err := tx.Commit(); err != nil {
			gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang service dibatalkan"})
	}
}

// ---------- ANGKA MERAH ----------

func serviceNotif(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*Claims)
		n := 0
		switch cl.Role {
		case "owner", "super_admin":
			db.QueryRow("SELECT COUNT(*) FROM service_jobs WHERE is_void = FALSE AND status IN ('dikirim_bo','selesai')").Scan(&n)
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