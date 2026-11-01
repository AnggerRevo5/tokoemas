package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

const tglFormat = "2006-01-02"

// Gadai untuk semua barang terdaftar (perhiasan, logam mulia emas atau perak). Dana yang diterima masuk ke kas lokasi barang, cicilan keluar dari kas yang sama.
// Setiap cicilan dipecah menjadi pokok (kategori gadai_pokok, bukan laba) dan biaya (gadai_biaya, masuk Laba Rugi).

const sqlGadai = `SELECT g.id, g.item_id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), IFNULL(i.brand,''), i.weight_gram,
	       g.location_id, l.name, g.pegadaian, g.dana, g.tenor, g.cicilan, g.total, g.start_date, g.first_due, g.status,
	       IFNULL(g.note,''), g.end_date, IFNULL(g.end_note,''), g.created_at >= NOW() - INTERVAL 7 DAY,
	       IFNULL(p.dibayar,0), IFNULL(p.denda,0), IFNULL(p.diskon,0), IFNULL(p.pokok,0), IFNULL(p.bulan,0)
	FROM gadai g
	JOIN items i ON i.id = g.item_id
	JOIN locations l ON l.id = g.location_id
	LEFT JOIN (SELECT gadai_id, SUM(amount) AS dibayar, SUM(denda) AS denda, SUM(diskon) AS diskon, SUM(pokok) AS pokok,
	                  SUM(jenis = 'cicilan') AS bulan
	           FROM gadai_pembayaran WHERE is_void = FALSE GROUP BY gadai_id) p ON p.gadai_id = g.id
	WHERE g.is_void = FALSE`

type pemindai interface{ Scan(dest ...any) error }

// bolehUbahTransaksi: akun cabang hanya di lokasinya sendiri dan dalam batas hari (gadai dan penjualan toko);
// Super Admin dan Owner selalu boleh.
func bolehUbahTransaksi(cl *models.Claims, lokasi int, barukah bool) bool {
	switch cl.Role {
	case "super_admin", "owner":
		return true
	case "cabang":
		return cl.LocationID == lokasi && barukah
	}
	return false
}

// bolehKelolaGadai: mencatat gadai dan pembayarannya hanya untuk akun cabang (lokasinya sendiri) dan Super Admin.
func bolehKelolaGadai(c *gin.Context, cl *models.Claims, lokasi int) bool {
	if cl.Role == "super_admin" || (cl.Role == "cabang" && cl.LocationID == lokasi) {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses untuk gadai di lokasi ini"})
	return false
}

func pindaiGadai(rs pemindai, cl *models.Claims) (models.GadaiRow, error) {
	var r models.GadaiRow
	var mulai, tempo time.Time
	var akhir sql.NullTime
	var baru bool
	if err := rs.Scan(&r.ID, &r.ItemID, &r.ItemNo, &r.ItemName, &r.Karat, &r.Brand, &r.WeightGram,
		&r.LocationID, &r.Lokasi, &r.Pegadaian, &r.Dana, &r.Tenor, &r.Cicilan, &r.Total, &mulai, &tempo, &r.Status,
		&r.Note, &akhir, &r.EndNote, &baru,
		&r.Dibayar, &r.Denda, &r.Diskon, &r.PokokDibayar, &r.BulanDibayar); err != nil {
		return r, err
	}
	r.StartDate = mulai.Format(tglFormat)
	r.FirstDue = tempo.Format(tglFormat)
	if akhir.Valid {
		r.EndDate = akhir.Time.Format(tglFormat)
	}
	r.Sisa = services.SisaGadai(r.Total, r.Dibayar, r.Diskon)
	if r.Status == services.GadaiAktif && r.Sisa > 0 {
		r.SisaBulan = r.Tenor - r.BulanDibayar
		if r.SisaBulan < 0 {
			r.SisaBulan = 0
		}
		jt := services.JatuhTempoKe(services.TanggalSaja(tempo), r.BulanDibayar)
		r.JatuhTempo = jt.Format(tglFormat)
		r.Terlambat = jt.Before(services.HariIniWIB())
	}
	r.BisaUbah = bolehUbahTransaksi(cl, r.LocationID, baru)
	return r, nil
}

func muatGadai(db *sql.DB, cl *models.Claims, id int) (models.GadaiRow, bool, error) {
	r, err := pindaiGadai(db.QueryRow(sqlGadai+" AND g.id = ?", id), cl)
	if err == sql.ErrNoRows {
		return r, false, nil
	}
	if err != nil {
		return r, false, err
	}
	satu := []models.GadaiRow{r}
	if err := isiBarangGadai(db, satu); err != nil {
		return r, false, err
	}
	return satu[0], true, nil
}

type penanya interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

// isiBarangGadai mengisi daftar barang tiap gadai dan menjumlahkan beratnya.
func isiBarangGadai(q penanya, daftar []models.GadaiRow) error {
	if len(daftar) == 0 {
		return nil
	}
	idx := map[int]int{}
	args := make([]interface{}, 0, len(daftar))
	for i := range daftar {
		idx[daftar[i].ID] = i
		daftar[i].Barang = []models.GadaiBarangRow{}
		daftar[i].WeightGram = 0
		args = append(args, daftar[i].ID)
	}
	rows, err := q.Query(
		`SELECT b.gadai_id, i.id, IFNULL(i.item_no,''), i.item_name, IFNULL(i.karat,''), IFNULL(i.brand,''), i.jenis, i.bahan, i.weight_gram, i.status
		 FROM gadai_barang b JOIN items i ON i.id = b.item_id
		 WHERE b.gadai_id IN (`+utils.Placeholders(len(args))+`) ORDER BY b.id`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var gid int
		var b models.GadaiBarangRow
		if err := rows.Scan(&gid, &b.ItemID, &b.ItemNo, &b.ItemName, &b.Karat, &b.Brand, &b.Jenis, &b.Bahan, &b.WeightGram, &b.Status); err != nil {
			return err
		}
		i := idx[gid]
		daftar[i].Barang = append(daftar[i].Barang, b)
		daftar[i].WeightGram += b.WeightGram
	}
	return rows.Err()
}

// ---------- DAFTAR GADAI ----------

func GadaiList(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		q := sqlGadai
		args := []interface{}{}
		if cl.Role == "cabang" {
			q += " AND g.location_id = ?"
			args = append(args, cl.LocationID)
		} else if loc := c.Query("location_id"); loc != "" && loc != "0" {
			q += " AND g.location_id = ?"
			args = append(args, loc)
		}
		if it := c.Query("item_id"); it != "" {
			q += " AND g.id IN (SELECT gadai_id FROM gadai_barang WHERE item_id = ?)"
			args = append(args, it)
		}
		switch s := c.DefaultQuery("status", "berjalan"); s {
		case "berjalan":
			q += " AND g.status IN ('aktif','lunas')"
		case "semua":
		case services.GadaiAktif, services.GadaiLunas, services.GadaiKembali, services.GadaiTidakDitebus:
			q += " AND g.status = ?"
			args = append(args, s)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "status tidak dikenal"})
			return
		}
		q += " ORDER BY g.id DESC LIMIT 500"
		rows, err := db.Query(q, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		out := []models.GadaiRow{}
		for rows.Next() {
			r, err := pindaiGadai(rows, cl)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			utils.Gagal(c, err)
			return
		}
		if err := isiBarangGadai(db, out); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

// ---------- DETAIL GADAI ----------

func GadaiDetail(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		g, ada, err := muatGadai(db, cl, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if !ada || (cl.Role == "cabang" && g.LocationID != cl.LocationID) {
			c.JSON(http.StatusNotFound, gin.H{"error": "gadai tidak ditemukan"})
			return
		}

		rows, err := db.Query(
			`SELECT p.id, p.jenis, p.tx_date, p.amount, p.denda, p.diskon, p.pokok, p.biaya, IFNULL(p.note,''),
			        p.is_void, IFNULL(p.void_reason,''), IFNULL(u.username,''), p.created_at >= NOW() - INTERVAL 7 DAY
			 FROM gadai_pembayaran p LEFT JOIN users u ON u.id = p.created_by
			 WHERE p.gadai_id = ? ORDER BY p.tx_date, p.id`, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer rows.Close()
		d := models.GadaiDetail{GadaiRow: g, Jadwal: []models.GadaiJadwal{}, Pembayaran: []models.GadaiBayarRow{}}
		var cicilan []string // tanggal bayar tiap cicilan yang berlaku, berurutan
		pelunasan := ""
		for rows.Next() {
			var p models.GadaiBayarRow
			var tgl time.Time
			var baru bool
			if err := rows.Scan(&p.ID, &p.Jenis, &tgl, &p.Amount, &p.Denda, &p.Diskon, &p.Pokok, &p.Biaya, &p.Note,
				&p.IsVoid, &p.VoidReason, &p.Pencatat, &baru); err != nil {
				utils.Gagal(c, err)
				return
			}
			p.TxDate = tgl.Format(tglFormat)
			p.BisaUbah = !p.IsVoid && (g.Status == services.GadaiAktif || g.Status == services.GadaiLunas) &&
				bolehUbahTransaksi(cl, g.LocationID, baru)
			if !p.IsVoid {
				if p.Jenis == "cicilan" {
					cicilan = append(cicilan, p.TxDate)
				} else {
					pelunasan = p.TxDate
				}
			}
			d.Pembayaran = append(d.Pembayaran, p)
		}
		first, _ := time.Parse(tglFormat, g.FirstDue)
		for k := 0; k < g.Tenor; k++ {
			j := models.GadaiJadwal{Ke: k + 1, JatuhTempo: services.JatuhTempoKe(first, k).Format(tglFormat)}
			switch {
			case k < len(cicilan):
				j.Dibayar, j.TglBayar, j.Via = true, cicilan[k], "cicilan"
			case pelunasan != "":
				j.Dibayar, j.TglBayar, j.Via = true, pelunasan, "pelunasan"
			}
			d.Jadwal = append(d.Jadwal, j)
		}
		c.JSON(http.StatusOK, d)
	}
}

// ---------- GADAIKAN BARANG ----------

func GadaiCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if cl.Role != "cabang" && cl.Role != "super_admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses menggadaikan barang"})
			return
		}
		var req models.GadaiBaruReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data gadai tidak valid"})
			return
		}
		req.Pegadaian = strings.TrimSpace(req.Pegadaian)
		req.Note = strings.TrimSpace(req.Note)
		// beberapa barang dalam satu gadai; item_id tunggal tetap diterima, tanpa duplikat dan dengan urutan dipertahankan
		semua := req.ItemIDs
		if len(semua) == 0 && req.ItemID > 0 {
			semua = []int{req.ItemID}
		}
		ids := []int{}
		terlihat := map[int]bool{}
		for _, id := range semua {
			if id > 0 && !terlihat[id] {
				terlihat[id] = true
				ids = append(ids, id)
			}
		}
		switch {
		case len(ids) == 0:
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih barang yang digadaikan"})
			return
		case req.Pegadaian == "":
			c.JSON(http.StatusBadRequest, gin.H{"error": "nama pegadaian wajib diisi"})
			return
		case req.Dana <= 0:
			c.JSON(http.StatusBadRequest, gin.H{"error": "dana yang diterima wajib diisi"})
			return
		case req.Tenor < 1:
			c.JSON(http.StatusBadRequest, gin.H{"error": "masa cicilan minimal 1 bulan"})
			return
		case req.Cicilan <= 0:
			c.JSON(http.StatusBadRequest, gin.H{"error": "cicilan per bulan wajib diisi"})
			return
		}
		total := int64(req.Tenor) * req.Cicilan
		if total < req.Dana {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("total cicilan (%s) lebih kecil dari dana yang diterima (%s)",
				services.FmtRp(float64(total)), services.FmtRp(float64(req.Dana)))})
			return
		}
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		mulai, _ := time.Parse(tglFormat, tgl)
		tempo := services.JatuhTempoKe(mulai, 1).Format(tglFormat)
		if req.JatuhTempoPertama != "" {
			if tempo, ok = utils.ParseTanggal(c, req.JatuhTempoPertama); !ok {
				return
			}
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		items, err := services.AmbilBarang(tx, ids)
		if err != nil {
			balasKesalahan(c, err)
			return
		}
		pertama := items[ids[0]]
		var totalBerat float64
		namaSemua := make([]string, 0, len(ids))
		for _, id := range ids {
			it := items[id]
			if it.Status != "tersedia" {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s sedang tidak tersedia (%s)", it.Name, it.Status)})
				return
			}
			if it.Location != pertama.Location {
				c.JSON(http.StatusBadRequest, gin.H{"error": "semua barang dalam satu gadai harus berasal dari lokasi yang sama"})
				return
			}
			totalBerat += it.Weight
			namaSemua = append(namaSemua, it.Name)
		}
		if !bolehKelolaGadai(c, cl, pertama.Location) {
			return
		}

		catatan := fmt.Sprintf("Dana gadai %s: %s %s %s g", req.Pegadaian, pertama.Name, pertama.Karat, strconv.FormatFloat(pertama.Weight, 'f', -1, 64))
		if len(ids) > 1 {
			catatan = fmt.Sprintf("Dana gadai %s: %d barang (%s) %s g", req.Pegadaian, len(ids), strings.Join(namaSemua, ", "),
				strconv.FormatFloat(totalBerat, 'f', -1, 64))
		}
		if r := []rune(catatan); len(r) > 250 {
			catatan = string(r[:247]) + "..."
		}
		rk, err := tx.Exec(
			`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
			 VALUES (?, 'in', ?, ?, ?, ?, ?)`,
			pertama.Location, services.KategoriGadaiDana, req.Dana, catatan, tgl, cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		kasID, _ := rk.LastInsertId()

		stokIDs := make([]int64, 0, len(ids))
		for _, id := range ids {
			it := items[id]
			rs, err := tx.Exec(services.SQLStokItem, it.Location, "out", "pegadaian", it.Name, it.Weight, it.Karat,
				"Digadaikan di "+req.Pegadaian, tgl, cl.UserID, it.ID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			stokID, _ := rs.LastInsertId()
			stokIDs = append(stokIDs, stokID)
			if _, err := tx.Exec("UPDATE items SET status = 'digadai' WHERE id = ?", it.ID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		rg, err := tx.Exec(
			`INSERT INTO gadai (item_id, location_id, pegadaian, dana, tenor, cicilan, total, start_date, first_due, note,
			                    cash_id, out_stock_id, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			pertama.ID, pertama.Location, req.Pegadaian, req.Dana, req.Tenor, req.Cicilan, total, tgl, tempo,
			sql.NullString{String: req.Note, Valid: req.Note != ""}, kasID, stokIDs[0], cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		gid, _ := rg.LastInsertId()
		for k, id := range ids {
			if _, err := tx.Exec("INSERT INTO gadai_barang (gadai_id, item_id, out_stock_id) VALUES (?,?,?)", gid, id, stokIDs[k]); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatAudit(tx, middleware.Meta(c), "create", "gadai", gid, gin.H{"request": req, "item_ids": ids, "total": total})
		services.CatatAudit(tx, middleware.Meta(c), "create_auto", "cash_transactions", kasID, gin.H{"gadai": gid, "dana": req.Dana})

		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		apa := pertama.Name
		if len(ids) > 1 {
			apa = fmt.Sprintf("%d barang", len(ids))
		}
		c.JSON(http.StatusCreated, gin.H{
			"id": gid,
			"message": fmt.Sprintf("%s digadaikan di %s, kas bertambah %s. Cicilan %d x %s",
				apa, req.Pegadaian, utils.RupiahTeks(float64(req.Dana)), req.Tenor, utils.RupiahTeks(float64(req.Cicilan))),
		})
	}
}

// ---------- BAYAR CICILAN DAN TEBUS LEBIH CEPAT ----------

type gadaiKunci struct {
	Lokasi                               int
	Status, Pegadaian                    string
	Dana, Tenor, Cicilan, Total          int64
	Dibayar, Diskon, PokokDibayar, Bulan int64
}

// kunciGadai mengunci satu gadai (belum dibatalkan) beserta jumlah pembayarannya.
func kunciGadai(tx *sql.Tx, id int) (gadaiKunci, error) {
	var g gadaiKunci
	err := tx.QueryRow(
		"SELECT location_id, status, pegadaian, dana, tenor, cicilan, total FROM gadai WHERE id = ? AND is_void = FALSE FOR UPDATE", id,
	).Scan(&g.Lokasi, &g.Status, &g.Pegadaian, &g.Dana, &g.Tenor, &g.Cicilan, &g.Total)
	if err != nil {
		return g, err
	}
	err = tx.QueryRow(
		`SELECT IFNULL(SUM(amount),0), IFNULL(SUM(diskon),0), IFNULL(SUM(pokok),0), IFNULL(SUM(jenis = 'cicilan'),0)
		 FROM gadai_pembayaran WHERE gadai_id = ? AND is_void = FALSE`, id,
	).Scan(&g.Dibayar, &g.Diskon, &g.PokokDibayar, &g.Bulan)
	return g, err
}

func GadaiBayar(db *sql.DB, pelunasan bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.GadaiBayarReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data pembayaran tidak valid"})
			return
		}
		req.Note = strings.TrimSpace(req.Note)
		tgl, ok := utils.ParseTanggal(c, req.Date)
		if !ok {
			return
		}
		if !pelunasan && req.Diskon > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "diskon hanya untuk tebus lebih cepat"})
			return
		}
		if !pelunasan && req.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "jumlah cicilan wajib diisi"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()

		g, err := kunciGadai(tx, id)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "gadai tidak ditemukan"})
			return
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if !bolehKelolaGadai(c, cl, g.Lokasi) {
			return
		}
		if g.Status != services.GadaiAktif {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gadai ini sudah lunas atau sudah selesai"})
			return
		}
		sisa := services.SisaGadai(g.Total, g.Dibayar, g.Diskon)

		var pokok, biaya int64
		jenis := "cicilan"
		if pelunasan {
			jenis = "pelunasan"
			if req.Amount+req.Diskon != sisa {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("jumlah bayar ditambah diskon harus sama dengan sisa %s", services.FmtRp(float64(sisa)))})
				return
			}
			var ok bool
			if pokok, biaya, ok = services.AlokasiPelunasan(g.Dana, g.PokokDibayar, req.Amount); !ok {
				c.JSON(http.StatusBadRequest, gin.H{"error": "jumlah bayar lebih kecil dari sisa pokok pinjaman, kurangi diskonnya"})
				return
			}
		} else {
			if req.Amount > sisa {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("cicilan melebihi sisa %s", services.FmtRp(float64(sisa)))})
				return
			}
			pokok, biaya = services.AlokasiCicilan(g.Dana, g.Total, g.PokokDibayar, req.Amount, sisa-req.Amount)
		}

		label := fmt.Sprintf("gadai %s", g.Pegadaian)
		if g.Tenor > 0 && !pelunasan {
			label = fmt.Sprintf("cicilan ke-%d gadai %s", g.Bulan+1, g.Pegadaian)
		} else if pelunasan {
			label = fmt.Sprintf("pelunasan gadai %s", g.Pegadaian)
		}
		var pokokID, biayaID sql.NullInt64
		if pokok > 0 {
			r, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
				 VALUES (?, 'out', ?, ?, ?, ?, ?)`,
				g.Lokasi, services.KategoriGadaiPokok, pokok, "Pokok "+label, tgl, cl.UserID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			n, _ := r.LastInsertId()
			pokokID = sql.NullInt64{Int64: n, Valid: true}
		}
		if biaya+req.Denda > 0 {
			catatan := "Biaya " + label
			if req.Denda > 0 {
				catatan += fmt.Sprintf(" (termasuk denda %s)", services.FmtRp(float64(req.Denda)))
			}
			r, err := tx.Exec(
				`INSERT INTO cash_transactions (location_id, direction, category, amount, note, transaction_date, created_by)
				 VALUES (?, 'out', ?, ?, ?, ?, ?)`,
				g.Lokasi, services.KategoriGadaiBiaya, biaya+req.Denda, catatan, tgl, cl.UserID)
			if err != nil {
				utils.Gagal(c, err)
				return
			}
			n, _ := r.LastInsertId()
			biayaID = sql.NullInt64{Int64: n, Valid: true}
		}
		rp, err := tx.Exec(
			`INSERT INTO gadai_pembayaran (gadai_id, jenis, tx_date, amount, denda, diskon, pokok, biaya, cash_pokok_id, cash_biaya_id, note, created_by)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, jenis, tgl, req.Amount, req.Denda, req.Diskon, pokok, biaya, pokokID, biayaID,
			sql.NullString{String: req.Note, Valid: req.Note != ""}, cl.UserID)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		pid, _ := rp.LastInsertId()

		sisaSetelah := sisa - req.Amount - req.Diskon
		if sisaSetelah == 0 {
			if _, err := tx.Exec("UPDATE gadai SET status = 'lunas' WHERE id = ?", id); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatAudit(tx, middleware.Meta(c), "create", "gadai_pembayaran", pid,
			gin.H{"gadai_id": id, "jenis": jenis, "amount": req.Amount, "denda": req.Denda, "diskon": req.Diskon, "pokok": pokok, "biaya": biaya})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}

		pesan := fmt.Sprintf("Pembayaran %s tercatat", services.FmtRp(float64(req.Amount+req.Denda)))
		if sisaSetelah == 0 {
			pesan += ". Gadai lunas, ambil barang di pegadaian lalu tekan Sudah kembali ke toko"
		} else {
			pesan += fmt.Sprintf(". Sisa %s", services.FmtRp(float64(sisaSetelah)))
		}
		c.JSON(http.StatusCreated, gin.H{"id": pid, "message": pesan, "lunas": sisaSetelah == 0})
	}
}

// ---------- BARANG KEMBALI KE TOKO / TIDAK DITEBUS ----------

// barangGadai adalah satu barang yang dijaminkan, dibaca sebelum diubah.
type barangGadai struct {
	ID    int
	Nama  string
	Karat string
	Berat float64
}

// kunciBarangGadai mengunci dan membaca semua barang dalam satu gadai.
func kunciBarangGadai(tx *sql.Tx, id int) ([]barangGadai, error) {
	rows, err := tx.Query(
		`SELECT i.id, i.item_name, IFNULL(i.karat,''), i.weight_gram
		 FROM gadai_barang b JOIN items i ON i.id = b.item_id WHERE b.gadai_id = ? ORDER BY b.id FOR UPDATE`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []barangGadai{}
	for rows.Next() {
		var b barangGadai
		if err := rows.Scan(&b.ID, &b.Nama, &b.Karat, &b.Berat); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func GadaiAkhir(db *sql.DB, kembali bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.GadaiAkhirReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		req.Reason = strings.TrimSpace(req.Reason)
		if !kembali && req.Reason == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "alasan tidak ditebus wajib diisi"})
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

		g, err := kunciGadai(tx, id)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "gadai tidak ditemukan"})
			return
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if !bolehKelolaGadai(c, cl, g.Lokasi) {
			return
		}
		barang, err := kunciBarangGadai(tx, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if len(barang) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "daftar barang gadai tidak ditemukan"})
			return
		}
		itemIDs := make([]int, 0, len(barang))
		for _, b := range barang {
			itemIDs = append(itemIDs, b.ID)
		}

		if kembali {
			if g.Status != services.GadaiLunas {
				c.JSON(http.StatusBadRequest, gin.H{"error": "barang baru bisa dikembalikan setelah gadai lunas"})
				return
			}
			var stokPertama int64
			for k, b := range barang {
				rs, err := tx.Exec(services.SQLStokItem, g.Lokasi, "in", "gadai_kembali", b.Nama, b.Berat, b.Karat,
					"Kembali dari "+g.Pegadaian, tgl, cl.UserID, b.ID)
				if err != nil {
					utils.Gagal(c, err)
					return
				}
				stokID, _ := rs.LastInsertId()
				if k == 0 {
					stokPertama = stokID
				}
				if _, err := tx.Exec("UPDATE items SET status = 'tersedia' WHERE id = ?", b.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
				if _, err := tx.Exec("UPDATE gadai_barang SET in_stock_id = ? WHERE gadai_id = ? AND item_id = ?", stokID, id, b.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			if _, err := tx.Exec("UPDATE gadai SET status = 'kembali', end_date = ?, in_stock_id = ? WHERE id = ?", tgl, stokPertama, id); err != nil {
				utils.Gagal(c, err)
				return
			}
			services.CatatAudit(tx, middleware.Meta(c), "gadai_kembali", "gadai", int64(id), gin.H{"item_ids": itemIDs, "tanggal": tgl})
		} else {
			if g.Status != services.GadaiAktif {
				c.JSON(http.StatusBadRequest, gin.H{"error": "hanya gadai yang masih berjalan yang bisa ditandai tidak ditebus"})
				return
			}
			for _, b := range barang {
				if _, err := tx.Exec("UPDATE items SET status = 'tidak_ditebus' WHERE id = ?", b.ID); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			if _, err := tx.Exec("UPDATE gadai SET status = 'tidak_ditebus', end_date = ?, end_note = ? WHERE id = ?", tgl, req.Reason, id); err != nil {
				utils.Gagal(c, err)
				return
			}
			services.CatatAudit(tx, middleware.Meta(c), "gadai_tidak_ditebus", "gadai", int64(id),
				gin.H{"item_ids": itemIDs, "tanggal": tgl, "alasan": req.Reason})
		}
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		apa := barang[0].Nama
		if len(barang) > 1 {
			apa = fmt.Sprintf("%d barang", len(barang))
		}
		if kembali {
			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s kembali ke stok toko", apa)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s ditandai tidak ditebus, barang tidak ada lagi di stok", apa)})
	}
}

// ---------- BATALKAN PEMBAYARAN / BATALKAN GADAI ----------

// voidKas membatalkan catatan kas yang valid (id boleh kosong).
func voidKas(tx *sql.Tx, meta services.MetaAudit, alasan string, ids ...sql.NullInt64) error {
	for _, id := range ids {
		if !id.Valid {
			continue
		}
		if _, err := tx.Exec("UPDATE cash_transactions SET is_void = TRUE, void_reason = ? WHERE id = ?", alasan, id.Int64); err != nil {
			return err
		}
		services.CatatAudit(tx, meta, "void", "cash_transactions", id.Int64, gin.H{"reason": alasan})
	}
	return nil
}

func GadaiBayarVoid(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
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

		var gid, lokasi int
		var status string
		var isVoid, baru bool
		var pokokID, biayaID sql.NullInt64
		err = tx.QueryRow(
			`SELECT p.gadai_id, g.location_id, g.status, p.is_void, p.created_at >= NOW() - INTERVAL 7 DAY, p.cash_pokok_id, p.cash_biaya_id
			 FROM gadai_pembayaran p JOIN gadai g ON g.id = p.gadai_id WHERE p.id = ? FOR UPDATE`, id,
		).Scan(&gid, &lokasi, &status, &isVoid, &baru, &pokokID, &biayaID)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "pembayaran tidak ditemukan"})
			return
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if !bolehUbahTransaksi(cl, lokasi, baru) {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("lebih dari %d hari atau bukan lokasi kamu, minta Super Admin atau Owner yang membatalkan", services.BatasHariUbahGadai)})
			return
		}
		if isVoid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pembayaran sudah dibatalkan"})
			return
		}
		if status != services.GadaiAktif && status != services.GadaiLunas {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gadai ini sudah selesai, pembayarannya tidak bisa dibatalkan"})
			return
		}
		if _, err := tx.Exec("UPDATE gadai_pembayaran SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if err := voidKas(tx, middleware.Meta(c), reason, pokokID, biayaID); err != nil {
			utils.Gagal(c, err)
			return
		}
		if status == services.GadaiLunas {
			tx.Exec("UPDATE gadai SET status = 'aktif' WHERE id = ?", gid)
		}
		services.CatatAudit(tx, middleware.Meta(c), "void", "gadai_pembayaran", int64(id), gin.H{"reason": reason, "gadai_id": gid})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "pembayaran dibatalkan, kas dikembalikan"})
	}
}

func GadaiBatal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
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

		var lokasi int
		var status string
		var baru bool
		var kasID sql.NullInt64
		err = tx.QueryRow(
			`SELECT location_id, status, created_at >= NOW() - INTERVAL 7 DAY, cash_id
			 FROM gadai WHERE id = ? AND is_void = FALSE FOR UPDATE`, id,
		).Scan(&lokasi, &status, &baru, &kasID)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "gadai tidak ditemukan"})
			return
		}
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if !bolehUbahTransaksi(cl, lokasi, baru) {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("lebih dari %d hari atau bukan lokasi kamu, minta Super Admin atau Owner yang membatalkan", services.BatasHariUbahGadai)})
			return
		}
		if status != services.GadaiAktif {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hanya gadai yang masih berjalan yang bisa dibatalkan"})
			return
		}
		var n int
		tx.QueryRow("SELECT COUNT(*) FROM gadai_pembayaran WHERE gadai_id = ? AND is_void = FALSE", id).Scan(&n)
		if n > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "gadai sudah punya pembayaran, batalkan pembayarannya dulu"})
			return
		}
		if _, err := tx.Exec("UPDATE gadai SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if err := voidKas(tx, middleware.Meta(c), reason, kasID); err != nil {
			utils.Gagal(c, err)
			return
		}
		type jaminan struct {
			item int
			stok sql.NullInt64
		}
		daftar := []jaminan{}
		rb, err := tx.Query("SELECT item_id, out_stock_id FROM gadai_barang WHERE gadai_id = ?", id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		for rb.Next() {
			var j jaminan
			if err := rb.Scan(&j.item, &j.stok); err != nil {
				rb.Close()
				utils.Gagal(c, err)
				return
			}
			daftar = append(daftar, j)
		}
		rb.Close()
		itemIDs := make([]int, 0, len(daftar))
		for _, j := range daftar {
			itemIDs = append(itemIDs, j.item)
			if j.stok.Valid {
				if _, err := tx.Exec("UPDATE stock_movements SET is_void = TRUE, void_reason = ? WHERE id = ?", reason, j.stok.Int64); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
			if _, err := tx.Exec("UPDATE items SET status = 'tersedia' WHERE id = ?", j.item); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatAudit(tx, middleware.Meta(c), "void", "gadai", int64(id), gin.H{"reason": reason, "item_ids": itemIDs})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "gadai dibatalkan, barang kembali tersedia dan kas dikembalikan"})
	}
}
