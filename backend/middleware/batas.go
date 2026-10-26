package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// pembatas menghitung kejadian per kunci dalam satu jendela waktu (disimpan di memori backend).
type pembatas struct {
	mu      sync.Mutex
	jendela time.Duration
	maks    int
	kunci   time.Duration // lama dikunci setelah melewati batas (0 = hanya sampai jendela habis)
	data    map[string]*catatan
}

type catatan struct {
	mulai  time.Time
	jumlah int
	sampai time.Time // terkunci sampai waktu ini
}

func pembatasBaru(jendela time.Duration, maks int, kunci time.Duration) *pembatas {
	p := &pembatas{jendela: jendela, maks: maks, kunci: kunci, data: map[string]*catatan{}}
	go func() {
		for range time.Tick(10 * time.Minute) {
			p.bersihkan()
		}
	}()
	return p
}

// terkunci mengembalikan sisa waktu kunci (0 bila tidak terkunci).
func (p *pembatas) terkunci(k string) time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c := p.data[k]; c != nil && time.Now().Before(c.sampai) {
		return time.Until(c.sampai)
	}
	return 0
}

// tambah mencatat satu kejadian; mengembalikan true bila batas terlewati.
func (p *pembatas) tambah(k string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	c := p.data[k]
	if c == nil || now.Sub(c.mulai) > p.jendela {
		c = &catatan{mulai: now}
		p.data[k] = c
	}
	c.jumlah++
	if c.jumlah > p.maks {
		lama := p.kunci
		if lama == 0 {
			lama = p.jendela - now.Sub(c.mulai)
		}
		c.sampai = now.Add(lama)
		return true
	}
	return false
}

func (p *pembatas) reset(k string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.data, k)
}

func (p *pembatas) bersihkan() {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	for k, c := range p.data {
		if now.Sub(c.mulai) > p.jendela && now.After(c.sampai) {
			delete(p.data, k)
		}
	}
}

var (
	// percobaan login per IP: maksimal 20 dalam 10 menit
	loginPerIP = pembatasBaru(10*time.Minute, 20, 15*time.Minute)
	// login gagal per pasangan username+IP: 5 kali dalam 15 menit lalu dikunci 15 menit.
	// Dipasangkan dengan IP agar orang lain tidak bisa mengunci akun toko dari luar.
	gagalPerAkun = pembatasBaru(15*time.Minute, 5, 15*time.Minute)
	// semua request API per IP (perlindungan banjir request)
	requestPerIP = pembatasBaru(time.Minute, 600, 0)
)

// CekBatasLogin dipanggil sebelum memeriksa password. Mengembalikan false (dan membalas 429) bila terkunci.
func CekBatasLogin(c *gin.Context, username string) bool {
	ip := c.ClientIP()
	sisa := gagalPerAkun.terkunci(username + "|" + ip)
	if s := loginPerIP.terkunci(ip); s > sisa {
		sisa = s
	}
	if sisa == 0 && loginPerIP.tambah(ip) {
		sisa = loginPerIP.terkunci(ip)
	}
	if sisa > 0 {
		c.Header("Retry-After", strconv.Itoa(int(sisa.Seconds())+1))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"error": "terlalu banyak percobaan login, coba lagi dalam " + strconv.Itoa(int(sisa.Minutes())+1) + " menit",
		})
		return false
	}
	return true
}

// CatatLoginGagal menambah hitungan gagal; mengembalikan true bila akun (untuk IP ini) baru saja dikunci.
func CatatLoginGagal(c *gin.Context, username string) bool {
	return gagalPerAkun.tambah(username + "|" + c.ClientIP())
}

// CatatLoginBerhasil menghapus hitungan gagal untuk username+IP ini.
func CatatLoginBerhasil(c *gin.Context, username string) {
	gagalPerAkun.reset(username + "|" + c.ClientIP())
}

// BatasRequest membatasi jumlah request per IP agar API tidak dibanjiri.
func BatasRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		if requestPerIP.tambah(c.ClientIP()) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "terlalu banyak permintaan, tunggu sebentar"})
			return
		}
		c.Next()
	}
}
