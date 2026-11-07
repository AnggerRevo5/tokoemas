package services

import (
	"database/sql"
	"fmt"
	"strings"
)

// Aturan jenis barang non-emas-gram: susuk emas, tempat perhiasan, kotak perhiasan.
// Ketiganya berat/kadar opsional, tidak masuk total gram, dan tidak bisa dicuci, dilebur, atau digadai.

// SQLJenisTanpaGram dipakai di dalam IN (...) untuk mengecualikan jenis ini dari total gram.
const SQLJenisTanpaGram = "'susuk','tempat_perhiasan','kotak_perhiasan'"

var awalanKode = map[string]string{"susuk": "SSK", "tempat_perhiasan": "TMP", "kotak_perhiasan": "KTK"}

var namaBawaan = map[string]string{"susuk": "Susuk emas", "tempat_perhiasan": "Tempat perhiasan", "kotak_perhiasan": "Kotak perhiasan"}

// TanpaGram: jenis yang tidak dihitung ke total gram emas.
func TanpaGram(jenis string) bool { return awalanKode[jenis] != "" }

// BolehGratis: jenis yang boleh dijual dengan harga 0.
func BolehGratis(jenis string) bool { return jenis == "tempat_perhiasan" || jenis == "kotak_perhiasan" }

// BolehBaki: jenis yang boleh ditaruh di baki.
func BolehBaki(jenis string) bool {
	return jenis == "perhiasan" || jenis == "logam_mulia" || jenis == "susuk"
}

// AwalanKode: awalan kode barang (SSK/TMP/KTK), jenis lain BRG.
func AwalanKode(jenis string) string {
	if a := awalanKode[jenis]; a != "" {
		return a
	}
	return "BRG"
}

// NamaBawaan: nama barang bila dikosongkan saat input; "" untuk jenis lain.
func NamaBawaan(jenis string) string { return namaBawaan[jenis] }

// SiapkanSkemaJenis menambah jenis susuk, tempat_perhiasan, kotak_perhiasan ke items.jenis bila belum ada.
// Isinya sama dengan db/migrations/030_jenis_aksesoris.sql (bisa dijalankan manual oleh admin database).
func SiapkanSkemaJenis(db *sql.DB) error {
	var tipe string
	if err := db.QueryRow(`SELECT COLUMN_TYPE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'items' AND COLUMN_NAME = 'jenis'`).Scan(&tipe); err != nil {
		return fmt.Errorf("gagal membaca kolom items.jenis: %v", err)
	}
	if strings.Contains(tipe, "susuk") {
		return nil
	}
	if _, err := db.Exec(`ALTER TABLE items MODIFY COLUMN jenis
		ENUM('logam_mulia','perhiasan','hasil_lebur','susuk','tempat_perhiasan','kotak_perhiasan') NOT NULL DEFAULT 'perhiasan'`); err != nil {
		return fmt.Errorf("jenis susuk/tempat/kotak belum ada di items.jenis dan gagal ditambah (%v); jalankan db/migrations/030_jenis_aksesoris.sql dengan user database yang punya hak ALTER", err)
	}
	return nil
}
