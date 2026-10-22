package utils

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func Gagal(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func BacaAlasan(c *gin.Context) (string, bool) {
	var req struct {
		Reason string `json:"reason"`
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
