package utils

import (
	"strconv"
	"strings"
)

// RupiahTeks memformat angka menjadi teks rupiah, contoh 1500000 menjadi "Rp 1.500.000".
func RupiahTeks(n float64) string {
	s := strconv.FormatInt(int64(n+0.5), 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return "Rp " + b.String()
}
