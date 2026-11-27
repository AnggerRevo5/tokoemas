package controllers

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"

	"toko-emas/middleware"
	"toko-emas/models"
	"toko-emas/services"
	"toko-emas/utils"
)

// langkahOtomatis memindah status tanpa pemeriksaan role (langkah yang dipicu sistem) tetapi tetap lewat LangkahValid.
func langkahOtomatis(tx *sql.Tx, id int, arah, dari, ke string, oleh int, catatan string) error {
	if !services.LangkahValid(arah, dari, ke) {
		return fmt.Errorf("langkah %s %s->%s tidak sah", arah, dari, ke)
	}
	if _, err := tx.Exec("UPDATE talangan_barang SET status = ? WHERE id = ?", ke, id); err != nil {
		return err
	}
	return services.CatatStatus(tx, id, dari, ke, oleh, catatan)
}

func adaDuplikat(ids []int) bool {
	lihat := map[int]bool{}
	for _, id := range ids {
		if lihat[id] {
			return true
		}
		lihat[id] = true
	}
	return false
}

func kunciDuplikat(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

type barisTukar struct {
	ID, BP, Stok int
	Arah, Status string
	Tukar        sql.NullInt64
	Harga        int64
	Nama         string
}

// TukarCreate: PJ membuat penukaran dengan satu cabang BP. Saldo dihitung di backend (int64 rupiah).
// Barang MASUK keluar dari stok BO; semua barang jadi DIHITUNG. Saldo 0 langsung LUNAS tanpa uang pindah.
func TukarCreate(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		cl, ok := middleware.HanyaPeran(c, "pj", "super_admin")
		if !ok {
			return
		}
		var req models.TukarCreateReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "data tidak valid"})
			return
		}
		semua := append(append([]int{}, req.MasukIDs...), req.KeluarIDs...)
		if len(semua) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pilih minimal satu barang"})
			return
		}
		if adaDuplikat(semua) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang yang dipilih dua kali"})
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

		args := make([]interface{}, len(semua))
		for i, id := range semua {
			args[i] = id
		}
		rows, err := tx.Query(
			`SELECT id, arah, status, cabang_pemilik_id, tukar_id, IFNULL(stok_id,0), CAST(harga_beli AS SIGNED), nama_barang
			 FROM talangan_barang WHERE id IN (`+utils.Placeholders(len(args))+`) ORDER BY id FOR UPDATE`, args...)
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		data := map[int]barisTukar{}
		for rows.Next() {
			var b barisTukar
			if err := rows.Scan(&b.ID, &b.Arah, &b.Status, &b.BP, &b.Tukar, &b.Stok, &b.Harga, &b.Nama); err != nil {
				rows.Close()
				utils.Gagal(c, err)
				return
			}
			data[b.ID] = b
		}
		rows.Close()
		if len(data) != len(semua) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ada barang talangan yang tidak ditemukan"})
			return
		}

		var nilaiMasuk, nilaiKeluar []int64
		cek := func(ids []int, arah, statusSiap string) bool {
			for _, id := range ids {
				b := data[id]
				switch {
				case b.Arah != arah:
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("%s bukan barang %s", b.Nama, arah)})
					return false
				case b.BP != req.BPID:
					c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("%s milik cabang BP lain", b.Nama)})
					return false
				case b.Tukar.Valid:
					c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("%s sudah masuk penukaran lain", b.Nama)})
					return false
				case b.Status != statusSiap:
					c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("%s berstatus %s, belum bisa ditukar", b.Nama, b.Status)})
					return false
				}
				if arah == services.ArahMasuk {
					var st string
					var loc int
					var void bool
					if err := tx.QueryRow("SELECT status, location_id, is_void FROM items WHERE id = ? FOR UPDATE", b.Stok).Scan(&st, &loc, &void); err != nil ||
						void || st != "tersedia" || loc != boID {
						c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("%s sedang diproses atau tidak ada di stok back office", b.Nama)})
						return false
					}
					nilaiMasuk = append(nilaiMasuk, b.Harga)
				} else {
					nilaiKeluar = append(nilaiKeluar, b.Harga)
				}
			}
			return true
		}
		if !cek(req.MasukIDs, services.ArahMasuk, services.StDiBO) || !cek(req.KeluarIDs, services.ArahKeluar, services.StDiBO) {
			return
		}

		totalMasuk, totalKeluar, saldo, arahUang := services.HitungTukar(nilaiMasuk, nilaiKeluar)
		tgl := time.Now().Format("2006-01-02")
		statusUang := "BELUM_BAYAR"
		if saldo == 0 {
			statusUang = "MASUK_KAS_BO"
		}
		res, err := tx.Exec(
			`INSERT INTO talangan_tukar (cabang_id, total_masuk, total_keluar, saldo, arah_uang, status_uang, pj_id, tanggal, catatan)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			req.BPID, totalMasuk, totalKeluar, saldo, arahUang, statusUang, cl.UserID, tgl, utils.StrNull(strings.TrimSpace(req.Catatan)))
		if err != nil {
			utils.Gagal(c, err)
			return
		}
		tukarID, _ := res.LastInsertId()

		gagal := func(err error, nama string) {
			if kunciDuplikat(err) {
				c.JSON(http.StatusConflict, gin.H{"error": nama + " sudah masuk penukaran lain"})
				return
			}
			utils.Gagal(c, err)
		}
		for _, id := range semua {
			b := data[id]
			if _, err := tx.Exec("INSERT INTO talangan_tukar_item (tukar_id, barang_id, arah, nilai) VALUES (?,?,?,?)", tukarID, id, b.Arah, b.Harga); err != nil {
				gagal(err, b.Nama)
				return
			}
			if _, err := tx.Exec("UPDATE talangan_barang SET tukar_id = ? WHERE id = ?", tukarID, id); err != nil {
				utils.Gagal(c, err)
				return
			}
			if b.Arah == services.ArahMasuk {
				if err := langkahOtomatis(tx, id, b.Arah, services.StDiBO, services.StDikirimKeBP, cl.UserID, "masuk penukaran dengan "+bp); err != nil {
					utils.Gagal(c, err)
					return
				}
				if _, err := tx.Exec("UPDATE items SET status = 'di_bp', bp_nama = ? WHERE id = ?", bp, b.Stok); err != nil {
					utils.Gagal(c, err)
					return
				}
				if _, err := tx.Exec(services.SQLStokItem, boID, "out", "talangan_tukar", b.Nama, 0, "", "Dikirim ke "+bp+" (penukaran)", tgl, cl.UserID, b.Stok); err != nil {
					utils.Gagal(c, err)
					return
				}
				if err := langkahOtomatis(tx, id, b.Arah, services.StDikirimKeBP, services.StDihitung, cl.UserID, "dihitung dalam penukaran"); err != nil {
					utils.Gagal(c, err)
					return
				}
			} else if err := langkahOtomatis(tx, id, b.Arah, services.StDiBO, services.StDihitung, cl.UserID, "dihitung dalam penukaran"); err != nil {
				utils.Gagal(c, err)
				return
			}
			if saldo == 0 {
				if err := langkahOtomatis(tx, id, b.Arah, services.StDihitung, services.StLunas, cl.UserID, "impas, tidak ada uang pindah"); err != nil {
					utils.Gagal(c, err)
					return
				}
			}
		}
		services.CatatAudit(tx, middleware.Meta(c), "talangan_tukar_baru", "talangan_tukar", tukarID,
			gin.H{"bp": bp, "masuk": req.MasukIDs, "keluar": req.KeluarIDs, "saldo": saldo, "arah_uang": arahUang})
		if err := tx.Commit(); err != nil {
			gagal(err, "barang")
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": tukarID, "total_masuk": totalMasuk, "total_keluar": totalKeluar, "saldo": saldo,
			"arah_uang": arahUang, "status_uang": statusUang})
	}
}
