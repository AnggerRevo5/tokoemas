package services

import (
	"database/sql"
	"fmt"

	"toko-emas/models"
	"toko-emas/utils"
)

// ambil dan kunci barang yang dipilih; semua id harus ada
func AmbilBarang(tx *sql.Tx, ids []int) (map[int]models.ItemRow, error) {
	uniq := map[int]bool{}
	args := []interface{}{}
	for _, id := range ids {
		if !uniq[id] {
			uniq[id] = true
			args = append(args, id)
		}
	}
	rows, err := tx.Query(
		`SELECT id, item_name, IFNULL(karat,''), weight_gram, location_id, status
		 FROM items WHERE is_void = FALSE AND id IN (`+utils.Placeholders(len(args))+`) FOR UPDATE`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]models.ItemRow{}
	for rows.Next() {
		var r models.ItemRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Karat, &r.Weight, &r.Location, &r.Status); err != nil {
			return nil, err
		}
		out[r.ID] = r
	}
	if len(out) != len(args) {
		return nil, fmt.Errorf("ada barang yang tidak ditemukan")
	}
	return out, nil
}

func Ringkas(items []models.ItemRow) (string, float64, int, string) {
	nama, karat := "", ""
	var berat float64
	samaNama, samaKarat := true, true
	for i, it := range items {
		berat += it.Weight
		if i == 0 {
			nama, karat = it.Name, it.Karat
			continue
		}
		if it.Name != nama {
			samaNama = false
		}
		if it.Karat != karat {
			samaKarat = false
		}
	}
	if !samaNama {
		nama = "Aneka barang"
	}
	if !samaKarat {
		karat = ""
	}
	return nama, berat, len(items), karat
}

func Kunci(m map[int]models.ItemRow) []int {
	out := make([]int, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	return out
}

func AmbilIDs(tx *sql.Tx, q string, args ...interface{}) ([]int, error) {
	rows, err := tx.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func AmbilAsal(tx *sql.Tx, ids []int) (map[int]int, error) {
	args := make([]interface{}, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := tx.Query("SELECT id, origin_location_id FROM items WHERE id IN ("+utils.Placeholders(len(args))+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]int{}
	for rows.Next() {
		var id, o int
		if err := rows.Scan(&id, &o); err != nil {
			return nil, err
		}
		out[id] = o
	}
	return out, nil
}
