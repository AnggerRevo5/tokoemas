package services

import (
	"testing"

	"toko-emas/models"
)

func TestTolakTalangan(t *testing.T) {
	biasa := map[int]models.ItemRow{1: {ID: 1, Name: "Cincin"}}
	if err := TolakTalangan(biasa); err != nil {
		t.Errorf("barang biasa ditolak: %v", err)
	}
	campur := map[int]models.ItemRow{1: {ID: 1, Name: "Cincin"}, 2: {ID: 2, Name: "Gelang BP", TalanganID: 7}}
	err := TolakTalangan(campur)
	if err == nil || !AdalahKesalahanInput(err) {
		t.Fatalf("barang talangan harus ditolak sebagai kesalahan input, dapat %v", err)
	}
}
