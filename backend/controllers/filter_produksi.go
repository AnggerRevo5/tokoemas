package controllers

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/utils"
)

// syaratBarang: syarat per barang dari query (q = nama atau nomor, karat, berat_min, berat_max, asal = id lokasi)
// untuk filter Cuci, Lebur, dan Service. a = alias tabel barang (i, atau s untuk Service).
// beratExpr dan kolomAsal = ekspresi SQL yang dibandingkan. denganNama=false bila q sudah ditangani pemanggil.
func syaratBarang(c *gin.Context, a, beratExpr, kolomAsal string, denganNama bool) (string, []interface{}) {
	kond, args := "", []interface{}{}
	if v := strings.TrimSpace(c.Query("q")); v != "" && denganNama {
		p := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(utils.BatasTeks(v, 60)) + "%"
		kond += " AND (" + a + ".item_name LIKE ? OR IFNULL(" + a + ".item_no,'') LIKE ?)"
		args = append(args, p, p)
	}
	k2, a2 := syaratKadarBerat(c, a, beratExpr)
	kond += k2
	args = append(args, a2...)
	if n, err := strconv.Atoi(c.Query("asal")); err == nil && n > 0 {
		kond += " AND " + kolomAsal + " = ?"
		args = append(args, n)
	}
	return kond, args
}

// syaratCariStok: pencarian di Stok. Tiap kata yang diketik harus cocok di salah satu: nama, catatan, nomor barang,
// atau nama pencuci. Kata berakhiran K (8K, 17k) juga dicocokkan ke kadar, karena kadar tersimpan di kolom terpisah
// dari nama. shortcut: maksimal 8 kata, selebihnya diabaikan.
func syaratCariStok(q string) (string, []interface{}) {
	kond, args := "", []interface{}{}
	kata := strings.Fields(q)
	if len(kata) > 8 {
		kata = kata[:8]
	}
	for _, k := range kata {
		p := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(utils.BatasTeks(k, 60)) + "%"
		kond += ` AND (i.item_name LIKE ? OR IFNULL(i.note,'') LIKE ? OR IFNULL(i.item_no,'') LIKE ? OR
			(i.status = 'cuci' AND (IFNULL(w.washer_name,'') LIKE ? OR IFNULL(x.washer_name,'') LIKE ?))`
		args = append(args, p, p, p, p, p)
		if kd, ok := utils.NormalKarat(k); ok && strings.HasSuffix(strings.ToUpper(k), "K") {
			kond += " OR i.karat = ?"
			args = append(args, kd)
		}
		kond += ")"
	}
	return kond, args
}

// syaratKadarBerat: filter kadar (query karat: 1K-24K, atau 999 untuk perak) dan rentang berat (berat_min, berat_max).
// Dipakai Cuci, Lebur, Service lewat syaratBarang, dan Stok langsung.
func syaratKadarBerat(c *gin.Context, a, beratExpr string) (string, []interface{}) {
	kond, args := "", []interface{}{}
	karats := c.QueryArray("karat")
	valid := make([]string, 0, len(karats))
	seen := map[string]bool{}
	for _, value := range karats {
		k, ok := utils.NormalKarat(value)
		if !ok && strings.TrimSpace(value) == "999" {
			k, ok = "999", true
		}
		if ok && !seen[k] {
			valid = append(valid, k)
			seen[k] = true
		}
	}
	if len(valid) == 1 {
		kond += " AND " + a + ".karat = ?"
		args = append(args, valid[0])
	} else if len(valid) > 1 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(valid)), ",")
		kond += " AND " + a + ".karat IN (" + placeholders + ")"
		for _, k := range valid {
			args = append(args, k)
		}
	}
	if v, err := strconv.ParseFloat(c.Query("berat_min"), 64); err == nil && v > 0 {
		kond += " AND " + beratExpr + " >= ?"
		args = append(args, v)
	}
	if v, err := strconv.ParseFloat(c.Query("berat_max"), 64); err == nil && v > 0 {
		kond += " AND " + beratExpr + " <= ?"
		args = append(args, v)
	}
	return kond, args
}

// syaratTanggal: filter tanggal mulai (query from dan to, sudah divalidasi TanggalOpsional) pada kolom tanggal.
func syaratTanggal(c *gin.Context, kolom string) (string, []interface{}) {
	kond, args := "", []interface{}{}
	if v := c.Query("from"); v != "" {
		kond += " AND DATE(" + kolom + ") >= ?"
		args = append(args, v)
	}
	if v := c.Query("to"); v != "" {
		kond += " AND DATE(" + kolom + ") <= ?"
		args = append(args, v)
	}
	return kond, args
}

// syaratOrang: pencarian nama orang (query orang) pada satu atau beberapa kolom.
func syaratOrang(c *gin.Context, kolom ...string) (string, []interface{}) {
	v := strings.TrimSpace(c.Query("orang"))
	if v == "" {
		return "", nil
	}
	p := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(utils.BatasTeks(v, 40)) + "%"
	parts, args := []string{}, []interface{}{}
	for _, k := range kolom {
		parts = append(parts, "IFNULL("+k+",'') LIKE ?")
		args = append(args, p)
	}
	return " AND (" + strings.Join(parts, " OR ") + ")", args
}
