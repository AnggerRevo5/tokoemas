package controllers

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

// RuteTalangan mendaftarkan semua rute talangan; dipakai routes.Setup dan tes.
func RuteTalangan(r gin.IRoutes, db *sql.DB) {
	r.GET("/talangan", TalanganList(db))
	r.POST("/talangan", TalanganCreate(db))
	r.GET("/talangan/bp", TalanganBPList(db))
	r.POST("/talangan/bp", BPCreate(db))
	r.PUT("/talangan/bp/:id", BPUpdate(db))
	r.GET("/talangan/piutang", PiutangList(db))
	r.POST("/talangan/tukar", TukarCreate(db))
	r.GET("/talangan/tukar", TukarList(db))
	r.GET("/talangan/tukar/:id", TukarDetail(db))
	r.POST("/talangan/tukar/:id/terima-pj", TukarTerimaPJ(db))
	r.POST("/talangan/tukar/:id/setor", TukarSetorBO(db))
	r.POST("/talangan/tukar/:id/transfer", TukarBayarTransfer(db))
	r.POST("/talangan/tukar/:id/koreksi-nominal", TukarKoreksiNominal(db))
	r.POST("/talangan/keluar", TalanganKeluarCreate(db))
	r.GET("/talangan/:id/riwayat", TalanganRiwayat(db))
	r.POST("/talangan/:id/kirim-bo", TalanganKirimBO(db))
	r.POST("/talangan/:id/terima-bo", TalanganTerimaBO(db))
	r.POST("/talangan/:id/koreksi-harga", TalanganKoreksiHarga(db))
}
