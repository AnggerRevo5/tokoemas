package utils

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func NormalKarat(s string) (string, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimSpace(strings.TrimSuffix(s, "K"))
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 24 {
		return "", false
	}
	return fmt.Sprintf("%dK", n), true
}

func BatasTeks(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// ---------- PJ: CUCI TIDAK JADI, BARANG KEMBALI KE BACK OFFICE ----------

func PtrFloat(n sql.NullFloat64) *float64 {
	if n.Valid {
		v := n.Float64
		return &v
	}
	return nil
}

func PtrTime(n sql.NullTime) *time.Time {
	if n.Valid {
		v := n.Time
		return &v
	}
	return nil
}

// ---------- 1. OWNER MEMULAI LEBUR ----------

func StrNull(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// ---------- TERIMA BARANG SERVICE DI TOKO ----------
