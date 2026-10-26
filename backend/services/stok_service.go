package services

import (
	"database/sql"
	"fmt"

	"github.com/gin-gonic/gin"

	"toko-emas/utils"
)

const SQLStokItem = `INSERT INTO stock_movements
	(location_id, direction, category, item_name, weight_gram, karat, qty, note, transaction_date, created_by, item_id)
	VALUES (?,?,?,?,?,?,1,?,?,?,?)`

// ---------- INPUT BARANG MASUK (banyak sekaligus) ----------

// barang selesai dicuci: kembali ke stok back office, kondisi bersih
func SelesaiCuci(tx *sql.Tx, ids []int, tgl string, meta MetaAudit, boID int) error {
	userID := meta.UserID
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := tx.Query(
		`SELECT i.id, i.item_name, IFNULL(i.karat,''), i.weight_gram, IFNULL(i.washing_job_id,0), IFNULL(w.washer_name,'')
		 FROM items i LEFT JOIN washing_jobs w ON w.id = i.washing_job_id
		 WHERE i.id IN (`+utils.Placeholders(len(args))+`)`, args...)
	if err != nil {
		return err
	}
	type brg struct {
		ID     int
		Name   string
		Karat  string
		Weight float64
		Job    int
		Washer string
	}
	list := []brg{}
	for rows.Next() {
		var b brg
		if err := rows.Scan(&b.ID, &b.Name, &b.Karat, &b.Weight, &b.Job, &b.Washer); err != nil {
			rows.Close()
			return err
		}
		list = append(list, b)
	}
	rows.Close()

	jobs := map[int]int{}
	for _, b := range list {
		if _, err := tx.Exec(SQLStokItem, boID, "in", "cucian", b.Name, b.Weight, b.Karat,
			"Selesai cuci oleh "+b.Washer, tgl, userID, b.ID); err != nil {
			return err
		}
		if _, err := tx.Exec("UPDATE items SET status = 'tersedia', kondisi = 'bersih' WHERE id = ?", b.ID); err != nil {
			return err
		}
		if b.Job > 0 {
			jobs[b.Job]++
		}
	}
	for job, n := range jobs {
		var sisa int
		tx.QueryRow("SELECT COUNT(*) FROM items WHERE washing_job_id = ? AND status = 'cuci'", job).Scan(&sisa)
		if sisa == 0 {
			tx.Exec("UPDATE washing_jobs SET status = 'selesai', finish_date = ? WHERE id = ?", tgl, job)
		}
		CatatAudit(tx, meta, "cuci_selesai", "washing_jobs", int64(job), gin.H{"jumlah": n, "sisa": sisa})
	}
	return nil
}

const SQLItemHasil = `INSERT INTO items
	(item_name, karat, weight_gram, origin_location_id, location_id, source_category, entry_date, note, created_by,
	 jenis, kondisi, harga_beli, status, melt_id)
	VALUES (?,?,?,?,?,'lebur',?,?,?,'hasil_lebur',?,?,'hasil',?)`

func BuatItemHasil(tx *sql.Tx, nama string, karat sql.NullString, berat float64, boID int, tgl, note string,
	userID int, kondisi string, modal float64, batchID int64, kategori string) (int64, error) {
	res, err := tx.Exec(SQLItemHasil, nama, karat, berat, boID, boID, tgl, note, userID, kondisi, modal, batchID)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	tx.Exec("UPDATE items SET item_no = ? WHERE id = ?", fmt.Sprintf("BRG-%05d", id), id)
	karatStr := ""
	if karat.Valid {
		karatStr = karat.String
	}
	if _, err := tx.Exec(SQLStokItem, boID, "in", kategori, nama, berat, karatStr, note, tgl, userID, id); err != nil {
		return 0, err
	}
	return id, nil
}
