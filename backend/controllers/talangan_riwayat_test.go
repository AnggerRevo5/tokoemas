package controllers

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

type barisRiwayat struct {
	Jenis   string    `json:"jenis"`
	Dari    string    `json:"dari"`
	Ke      string    `json:"ke"`
	Oleh    string    `json:"oleh"`
	Waktu   time.Time `json:"waktu"`
	Catatan string    `json:"catatan"`
}

func ambilRiwayat(t *testing.T, role string, user, toko, id int) []barisRiwayat {
	t.Helper()
	db := dbUji(t)
	w := panggil(mesinTalangan(db, role, user, toko), "GET", fmt.Sprintf("/talangan/%d/riwayat", id), "")
	if w.Code != http.StatusOK {
		t.Fatalf("riwayat: %d %s", w.Code, w.Body.String())
	}
	var r []barisRiwayat
	jsonKe(t, w, &r)
	return r
}

func adaKe(r []barisRiwayat, jenis, ke string) bool {
	for _, x := range r {
		if x.Jenis == jenis && x.Ke == ke {
			return true
		}
	}
	return false
}

// riwayat talangan juga memuat kejadian cuci barangnya (dari antrean dan gerak stok cuci), diurutkan menurut waktu
func TestRiwayatTalanganMemuatCuci(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	id, stokID := masukDiBO(t, a, 350000)

	r := ambilRiwayat(t, "owner", a.User, 0, id)
	for _, x := range r {
		if x.Jenis != "status" {
			t.Errorf("sebelum dicuci hanya ada riwayat status, dapat jenis %q", x.Jenis)
		}
	}

	// masuk antrean cuci lewat alur lama
	owner := mesinBarang(db, "owner", a.User)
	if w := panggil(owner, "POST", "/cuci", fmt.Sprintf(`{"item_ids":[%d]}`, stokID)); w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("cuci: %d %s", w.Code, w.Body.String())
	}
	r = ambilRiwayat(t, "owner", a.User, 0, id)
	if !adaKe(r, "cuci", "Masuk antrean cuci") {
		t.Errorf("riwayat harus memuat masuk antrean cuci: %+v", r)
	}

	// dikirim ke pencuci lalu selesai: dicatat alur lama sebagai gerak stok
	db.Exec("DELETE FROM washing_queue_items WHERE item_id = ?", stokID)
	ins := func(arah, kat string, menit int) {
		db.Exec(`INSERT INTO stock_movements (location_id, direction, category, item_name, weight_gram, qty, note, transaction_date, created_by, item_id, created_at)
		         VALUES (?,?,?,?,1,1,'uji',CURDATE(),?,?, NOW() + INTERVAL ? MINUTE)`, a.BO, arah, kat, "Gelang uji", a.User, stokID, menit)
	}
	ins("out", "cuci", 1)
	ins("in", "cucian", 2)
	r = ambilRiwayat(t, "owner", a.User, 0, id)
	if !adaKe(r, "cuci", "Dikirim ke pencuci") || !adaKe(r, "cuci", "Selesai cuci, diterima back office") {
		t.Errorf("riwayat harus memuat kirim ke pencuci dan selesai cuci: %+v", r)
	}
	for i := 1; i < len(r); i++ {
		if r[i].Waktu.Before(r[i-1].Waktu) {
			t.Errorf("riwayat tidak urut waktu pada baris %d", i)
		}
	}
	// gerak stok yang di-void tidak ikut
	db.Exec("UPDATE stock_movements SET is_void = TRUE WHERE item_id = ? AND category = 'cucian'", stokID)
	r = ambilRiwayat(t, "owner", a.User, 0, id)
	if adaKe(r, "cuci", "Selesai cuci, diterima back office") {
		t.Error("gerak stok yang dibatalkan tidak boleh tampil")
	}
}
