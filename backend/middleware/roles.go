package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"toko-emas/models"
)

func HanyaSuperAdmin(c *gin.Context) (*models.Claims, bool) {
	cl := c.MustGet("claims").(*models.Claims)
	if cl.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "hanya Super Admin yang bisa melakukan ini"})
		return nil, false
	}
	return cl, true
}

func HanyaAdminOwner(c *gin.Context) (*models.Claims, bool) {
	cl := c.MustGet("claims").(*models.Claims)
	if cl.Role != "super_admin" && cl.Role != "owner" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
		return nil, false
	}
	return cl, true
}

func HanyaPJ(c *gin.Context) (*models.Claims, bool) {
	cl := c.MustGet("claims").(*models.Claims)
	if cl.Role != "pj" && cl.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "hanya PJ yang bisa mencatat langkah ini"})
		return nil, false
	}
	return cl, true
}

func BolehLihatLebur(c *gin.Context) (*models.Claims, bool) {
	cl := c.MustGet("claims").(*models.Claims)
	if cl.Role != "pj" && cl.Role != "owner" && cl.Role != "super_admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
		return nil, false
	}
	return cl, true
}

func BatasiPJ() gin.HandlerFunc {
	return func(c *gin.Context) {
		cl := c.MustGet("claims").(*models.Claims)
		if cl.Role == "pj" {
			p := c.FullPath()
			boleh := p == "/me" || p == "/barang/notif" ||
				strings.HasPrefix(p, "/lebur") ||
				strings.HasPrefix(p, "/penjualan") ||
				strings.HasPrefix(p, "/service")
			if !boleh {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "tidak punya akses"})
				return
			}
		}
		c.Next()
	}
}

// ---------- BANTUAN ----------
