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

// kunciTalangan mengunci satu catatan dan mengembalikan arah, status, serta kolom yang dibutuhkan langkah berikutnya.
type barisTalangan struct {
	ID                                      int
	Arah, Status, Nama, Jenis, Bahan, Karat string
	Berat                                   float64
	Harga                                   int64
	Tanggal                                 string
	TokoInput, BPID, StokID                 int
	BP                                      string
}

func kunciTalangan(c *gin.Context, tx *sql.Tx, id int) (b barisTalangan, ok bool) {
	var toko, stok sql.NullInt64
	var karat sql.NullString
	err := tx.QueryRow(
		`SELECT t.id, t.arah, t.status, t.nama_barang, t.jenis, t.bahan, IFNULL(t.kadar,''), t.berat, CAST(t.harga_beli AS SIGNED),
		        DATE_FORMAT(t.tanggal_beli,'%Y-%m-%d'), t.toko_input_id, t.cabang_pemilik_id, t.stok_id, bp.nama
		 FROM talangan_barang t JOIN bp_cabang bp ON bp.id = t.cabang_pemilik_id WHERE t.id = ? FOR UPDATE`, id).
		Scan(&b.ID, &b.Arah, &b.Status, &b.Nama, &b.Jenis, &b.Bahan, &karat, &b.Berat, &b.Harga, &b.Tanggal, &toko, &b.BPID, &stok, &b.BP)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "data tidak ditemukan"})
		return b, false
	}
	if err != nil {
		utils.Gagal(c, err)
		return b, false
	}
	b.Karat = karat.String
	b.TokoInput, b.StokID = int(toko.Int64), int(stok.Int64)
	return b, true
}

// pindahStatus memeriksa langkah, memperbarui status, dan menulis log. Menulis 409 bila langkah tidak sah.
func pindahStatus(c *gin.Context, tx *sql.Tx, b barisTalangan, ke string, oleh int, catatan string) bool {
	if !services.LangkahValid(b.Arah, b.Status, ke) {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("status %s tidak bisa dipindah ke %s", b.Status, ke)})
		return false
	}
	if _, err := tx.Exec("UPDATE talangan_barang SET status = ? WHERE id = ?", ke, b.ID); err != nil {
		utils.Gagal(c, err)
		return false
	}
	if err := services.CatatStatus(tx, b.ID, b.Status, ke, oleh, catatan); err != nil {
		utils.Gagal(c, err)
		return false
	}
	return true
}

// buatItemBO membuat satu barang di stok back office lengkap dengan gerak stok masuk.
func buatItemBO(tx *sql.Tx, boID, asalID int, b barisTalangan, kategori, note string, userID int, talanganID interface{}) (int64, error) {
	tgl := time.Now().Format("2006-01-02")
	res, err := tx.Exec(
		`INSERT INTO items (item_name, karat, weight_gram, origin_location_id, location_id, source_category, entry_date, note, created_by,
		   jenis, kondisi, harga_beli, bahan, talangan_id)
		 VALUES (?,?,?,?,?,?,?,?,?,?,'kotor',?,?,?)`,
		b.Nama, b.Karat, b.Berat, asalID, boID, kategori, tgl, note, userID, b.Jenis, b.Harga, b.Bahan, talanganID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.Exec("UPDATE items SET item_no = ? WHERE id = ?", fmt.Sprintf("%s-%05d", services.AwalanKode(b.Jenis), id), id); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(services.SQLStokItem, boID, "in", kategori, b.Nama, b.Berat, b.Karat, note, tgl, userID, id); err != nil {
		return 0, err
	}
	return id, nil
}

func idBOTx(c *gin.Context, tx *sql.Tx) (int, bool) {
	var id int
	if err := tx.QueryRow("SELECT id FROM locations WHERE type = 'back_office' LIMIT 1").Scan(&id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lokasi back office tidak ditemukan"})
		return 0, false
	}
	return id, true
}

// TalanganKirimBO: PJ mengonfirmasi catatan dan membawa barang ke BO (TALANGAN -> KIRIM_BO).
func TalanganKirimBO(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, services.PerananUntuk(services.ArahMasuk, services.StKirimBO)...)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		b, ok := kunciTalangan(c, tx, id)
		if !ok || !pindahStatus(c, tx, b, services.StKirimBO, cl.UserID, "dibawa PJ ke back office") {
			return
		}
		if _, err := tx.Exec("UPDATE talangan_barang SET dikonfirmasi_pj_oleh = ? WHERE id = ?", cl.UserID, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "talangan_kirim_bo", "talangan_barang", int64(id), gin.H{"dari": b.Status})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang dalam perjalanan ke back office, menunggu konfirmasi owner"})
	}
}

// TalanganTerimaBO: owner mengonfirmasi barang sudah masuk BO (KIRIM_BO -> DI_BO). Barang masuk stok BO bertanda talangan.
func TalanganTerimaBO(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, services.PerananUntuk(services.ArahMasuk, services.StDiBO)...)
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		b, ok := kunciTalangan(c, tx, id)
		if !ok {
			return
		}
		if !pindahStatus(c, tx, b, services.StDiBO, cl.UserID, "diterima owner di back office") {
			return
		}
		boID, ok := idBOTx(c, tx)
		if !ok {
			return
		}
		stokID, err := buatItemBO(tx, boID, b.TokoInput, b, "talangan", "Talangan "+b.BP, cl.UserID, id)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec("UPDATE talangan_barang SET stok_id = ?, dikonfirmasi_owner_oleh = ? WHERE id = ?", stokID, cl.UserID, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "talangan_terima_bo", "talangan_barang", int64(id), gin.H{"stok_id": stokID})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "barang masuk stok back office (bertanda talangan)", "stok_id": stokID})
	}
}

// TalanganKeluarCreate: PJ mencatat barang kita yang dibeli BP dari customer (talangan KELUAR, status TERCATAT).
// Barang masuk stok BO sebagai stok kita biasa (tanpa tanda talangan).
func TalanganKeluarCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, services.PerananUntuk(services.ArahKeluar, services.StTercatat)...)
		if !ok {
			return
		}
		var req models.TalanganKeluarReq
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
		boID, ok := idBOTx(c, tx)
		if !ok {
			return
		}
		res, err := tx.Exec(
			`INSERT INTO talangan_barang (arah, cabang_pemilik_id, nama_barang, jenis, bahan, berat, kadar, harga_beli, tanggal_beli,
			   no_referensi, status, diinput_oleh, catatan)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			services.ArahKeluar, req.BPID, req.NamaBarang, jenis, bahan, req.Berat, karat, req.HargaBeli, tgl,
			strings.TrimSpace(req.NoReferensi), services.StTercatat, cl.UserID, utils.StrNull(strings.TrimSpace(req.Catatan)))
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		id, _ := res.LastInsertId()
		b := barisTalangan{Nama: req.NamaBarang, Jenis: jenis, Bahan: bahan, Karat: karat, Berat: req.Berat, Harga: req.HargaBeli}
		stokID, err := buatItemBO(tx, boID, boID, b, "talangan_keluar", "Barang kita dari "+bp, cl.UserID, nil)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec("UPDATE talangan_barang SET stok_id = ? WHERE id = ?", stokID, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if err := services.CatatStatus(tx, int(id), "", services.StTercatat, cl.UserID, "barang diambil dari "+bp); err != nil {
			utils.Gagal(c, err)
			return
		}
		services.CatatAudit(tx, middleware.Meta(c), "talangan_keluar_baru", "talangan_barang", id, req)
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "stok_id": stokID, "message": "barang kita dari " + bp + " tercatat dan masuk stok back office"})
	}
}

// TalanganKoreksiHarga: satu-satunya jalan mengubah harga_beli (terkunci sejak input). Hanya owner, alasan wajib,
// dan hanya sebelum barang masuk penukaran (nilainya disalin ke penukaran). Kas toko TIDAK ikut berubah.
func TalanganKoreksiHarga(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "owner", "super_admin")
		if !ok {
			return
		}
		id, ok := utils.IDParam(c)
		if !ok {
			return
		}
		var req models.KoreksiHargaReq
		if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Alasan) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "harga baru dan alasan wajib diisi"})
			return
		}
		req.Alasan = strings.TrimSpace(req.Alasan)
		tx, err := db.Begin()
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		defer tx.Rollback()
		b, ok := kunciTalangan(c, tx, id)
		if !ok {
			return
		}
		var tukar sql.NullInt64
		tx.QueryRow("SELECT tukar_id FROM talangan_barang WHERE id = ?", id).Scan(&tukar)
		if tukar.Valid {
			c.JSON(http.StatusConflict, gin.H{"error": "barang sudah masuk penukaran; harga tidak bisa dikoreksi di sini (koreksi nominal penukaran oleh owner)"})
			return
		}
		if req.HargaBaru == b.Harga {
			c.JSON(http.StatusBadRequest, gin.H{"error": "harga baru sama dengan harga lama"})
			return
		}
		if _, err := tx.Exec(
			"INSERT INTO talangan_koreksi_harga (talangan_id, harga_lama, harga_baru, alasan, oleh) VALUES (?,?,?,?,?)",
			id, b.Harga, req.HargaBaru, req.Alasan, cl.UserID); err != nil {
			utils.Gagal(c, err)
			return
		}
		if _, err := tx.Exec("UPDATE talangan_barang SET harga_beli = ?, ubah_harga = TRUE WHERE id = ?", req.HargaBaru, id); err != nil {
			utils.Gagal(c, err)
			return
		}
		if b.StokID > 0 {
			if _, err := tx.Exec("UPDATE items SET harga_beli = ? WHERE id = ?", req.HargaBaru, b.StokID); err != nil {
				utils.Gagal(c, err)
				return
			}
		}
		services.CatatPerubahan(tx, middleware.Meta(c), "talangan_koreksi_harga", "talangan_barang", int64(id),
			gin.H{"harga_beli": b.Harga}, gin.H{"harga_beli": req.HargaBaru, "alasan": req.Alasan})
		if err := tx.Commit(); err != nil {
			utils.Gagal(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "harga dikoreksi; kas toko tidak ikut berubah"})
	}
}
