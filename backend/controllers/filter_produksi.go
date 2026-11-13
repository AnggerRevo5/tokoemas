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
	if k, ok := utils.NormalKarat(c.Query("karat")); ok {
		kond += " AND " + a + ".karat = ?"
		args = append(args, k)
	}
	if v, err := strconv.ParseFloat(c.Query("berat_min"), 64); err == nil && v > 0 {
		kond += " AND " + beratExpr + " >= ?"
		args = append(args, v)
	}
	if v, err := strconv.ParseFloat(c.Query("berat_max"), 64); err == nil && v > 0 {
		kond += " AND " + beratExpr + " <= ?"
		args = append(args, v)
	}
	if n, err := strconv.Atoi(c.Query("asal")); err == nil && n > 0 {
		kond += " AND " + kolomAsal + " = ?"
		args = append(args, n)
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
