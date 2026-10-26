package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// maksBody adalah ukuran maksimal body request (input 300 barang sekaligus masih jauh di bawah batas ini).
const maksBody = 1 << 20 // 1 MB

// HeaderKeamanan memasang header keamanan pada semua respons API.
func HeaderKeamanan() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		// data keuangan tidak boleh tersimpan di cache browser atau proxy
		h.Set("Cache-Control", "no-store")
		c.Next()
	}
}

// BatasBody menolak body request yang lebih besar dari maksBody.
func BatasBody() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maksBody {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "data terlalu besar"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maksBody)
		c.Next()
	}
}

// Pemulihan menangkap panic tanpa membocorkan detail ke pengguna; detailnya hanya di log server.
func Pemulihan() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Printf("panic pada %s %s: %v", c.Request.Method, c.FullPath(), err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "terjadi kesalahan di server"})
	})
}
