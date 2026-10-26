package utils

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Gagal membalas 500 dengan pesan umum. Detail kesalahan (query, nama tabel, dsb.) hanya ditulis
// ke log server agar tidak bocor ke pengguna.
func Gagal(c *gin.Context, err error) {
	log.Printf("kesalahan server pada %s %s: %v", c.Request.Method, c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan di server, coba lagi atau hubungi admin"})
}

// TanggalOpsional memeriksa filter tanggal dari query (boleh kosong).
func TanggalOpsional(c *gin.Context, nama ...string) bool {
	for _, n := range nama {
		if v := c.Query(n); v != "" {
			if _, err := time.Parse("2006-01-02", v); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "format tanggal harus YYYY-MM-DD"})
				return false
			}
		}
	}
	return true
}

func BacaAlasan(c *gin.Context) (string, bool) {
	var req struct {
		Reason string `json:"reason" binding:"max=250"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alasan pembatalan wajib diisi"})
		return "", false
	}
	return req.Reason, true
}

func IDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id tidak valid"})
		return 0, false
	}
	return id, true
}

func ParseTanggal(c *gin.Context, s string) (string, bool) {
	if s == "" {
		return time.Now().Format("2006-01-02"), true
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "format tanggal harus YYYY-MM-DD"})
		return "", false
	}
	return s, true
}
