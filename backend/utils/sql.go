package utils

import (
	"strings"
)

func Placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
