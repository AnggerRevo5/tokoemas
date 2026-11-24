package controllers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"toko-emas/models"
)

// mesinBarang: router uji untuk rute stok lama yang harus menolak atau mengecualikan barang talangan.
func mesinBarang(db *sql.DB, role string, userID int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("claims", &models.Claims{UserID: userID, Username: "uji", Role: role})
		c.Next()
	})
	r.GET("/saldo", Saldo(db))
	r.GET("/barang", BarangList(db))
	r.POST("/barang/oper", BarangOper(db))
	r.POST("/barang/tarik-bp", BarangTarikBP(db))
	r.POST("/barang/jual", BarangJual(db))
	r.POST("/barang/void/:id", BarangVoid(db))
	r.PUT("/barang/:id", BarangEdit(db))
	r.POST("/lebur", LeburMulai(db))
	r.POST("/cuci", CuciMulai(db))
	r.GET("/barang/notif", BarangNotif(db))
	r.POST("/kas/:id/void", VoidTransaksi(db, "cash_transactions"))
	return r
}

// masukDiBO membuat talangan MASUK sampai DI_BO dan mengembalikan id talangan dan id barang di stok BO.
func masukDiBO(t *testing.T, a akunUji, harga int64) (id, stokID int) {
	t.Helper()
	db := dbUji(t)
	id = masukUji(t, a, harga)
	pj := mesinTalangan(db, "pj", a.User, 0)
	owner := mesinTalangan(db, "owner", a.User, 0)
	if w := panggil(pj, "POST", fmt.Sprintf("/talangan/%d/kirim-bo", id), `{}`); w.Code != http.StatusOK {
		t.Fatalf("kirim-bo: %s", w.Body.String())
	}
	if w := panggil(owner, "POST", fmt.Sprintf("/talangan/%d/terima-bo", id), `{}`); w.Code != http.StatusOK {
		t.Fatalf("terima-bo: %s", w.Body.String())
	}
	db.QueryRow("SELECT stok_id FROM talangan_barang WHERE id = ?", id).Scan(&stokID)
	return id, stokID
}

func TestKoreksiHargaValidasiDanRole(t *testing.T) {
	body := `{"harga_baru":2000000,"alasan":"salah ketik"}`
	for _, role := range []string{"cabang", "pj"} {
		r := mesinTalangan(nil, role, 1, 1)
		if w := panggil(r, "POST", "/talangan/1/koreksi-harga", body); w.Code != http.StatusForbidden {
			t.Errorf("%s: kode %d, seharusnya 403", role, w.Code)
		}
	}
	r := mesinTalangan(nil, "owner", 1, 0)
	for nama, b := range map[string]string{
		"tanpa alasan": `{"harga_baru":2000000}`,
		"alasan spasi": `{"harga_baru":2000000,"alasan":"   "}`,
		"harga nol":    `{"harga_baru":0,"alasan":"x"}`,
	} {
		if w := panggil(r, "POST", "/talangan/1/koreksi-harga", b); w.Code != http.StatusBadRequest {
			t.Errorf("%s: kode %d, seharusnya 400", nama, w.Code)
		}
	}
}

func TestKoreksiHargaOwnerTercatat(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	id, stokID := masukDiBO(t, a, 1000000)
	owner := mesinTalangan(db, "owner", a.User, 0)
	path := fmt.Sprintf("/talangan/%d/koreksi-harga", id)
	if w := panggil(owner, "POST", path, `{"harga_baru":1250000,"alasan":"salah ketik nominal"}`); w.Code != http.StatusOK {
		t.Fatalf("koreksi: %d %s", w.Code, w.Body.String())
	}
	var harga, hargaItem int64
	var ubah bool
	db.QueryRow("SELECT CAST(harga_beli AS SIGNED), ubah_harga FROM talangan_barang WHERE id = ?", id).Scan(&harga, &ubah)
	db.QueryRow("SELECT CAST(harga_beli AS SIGNED) FROM items WHERE id = ?", stokID).Scan(&hargaItem)
	if harga != 1250000 || !ubah || hargaItem != 1250000 {
		t.Errorf("harga %d ubah %v harga barang BO %d", harga, ubah, hargaItem)
	}
	var lama, baru int64
	var alasan string
	var oleh int
	db.QueryRow("SELECT CAST(harga_lama AS SIGNED), CAST(harga_baru AS SIGNED), alasan, oleh FROM talangan_koreksi_harga WHERE talangan_id = ?", id).Scan(&lama, &baru, &alasan, &oleh)
	if lama != 1000000 || baru != 1250000 || alasan != "salah ketik nominal" || oleh != a.User {
		t.Errorf("tabel koreksi: %d→%d %q oleh %d", lama, baru, alasan, oleh)
	}
	// sesudah masuk penukaran nilainya sudah disalin, koreksi biasa ditolak
	db.Exec("UPDATE talangan_barang SET tukar_id = 999999 WHERE id = ?", id)
	if w := panggil(owner, "POST", path, `{"harga_baru":1300000,"alasan":"coba lagi"}`); w.Code != http.StatusConflict {
		t.Errorf("koreksi setelah penukaran: kode %d, seharusnya 409", w.Code)
	}
}

func TestTidakAdaEndpointUbahHargaBiasa(t *testing.T) {
	r := mesinTalangan(nil, "owner", 1, 0)
	if w := panggil(r, "PUT", "/talangan/1", `{"harga_beli":1}`); w.Code != http.StatusNotFound {
		t.Errorf("PUT /talangan/1 seharusnya tidak ada (404), dapat %d", w.Code)
	}
}

func TestSaldoStokTidakMenghitungBarangTalangan(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	ambil := func() (gram float64, pcs int) {
		w := panggil(mesinBarang(db, "owner", a.User), "GET", "/saldo", "")
		var rows []struct {
			LocationID int     `json:"location_id"`
			StokGram   float64 `json:"stok_gram"`
			StokPcs    int     `json:"stok_pcs"`
		}
		jsonKe(t, w, &rows)
		for _, x := range rows {
			if x.LocationID == a.BO {
				return x.StokGram, x.StokPcs
			}
		}
		t.Fatal("lokasi BO tidak ada di saldo")
		return
	}
	g1, p1 := ambil()
	masukDiBO(t, a, 500000) // 4,5 gram masuk BO bertanda talangan
	g2, p2 := ambil()
	if g1 != g2 || p1 != p2 {
		t.Errorf("stok BO berubah karena barang talangan: %.3f g %d pcs -> %.3f g %d pcs", g1, p1, g2, p2)
	}
}

func TestDaftarBarangLabelDanFilterTalangan(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	_, stokID := masukDiBO(t, a, 700000)
	r := mesinBarang(db, "owner", a.User)
	ambil := func(q string) []struct {
		ID       int  `json:"id"`
		Talangan bool `json:"talangan"`
	} {
		w := panggil(r, "GET", "/barang?limit=2000"+q, "")
		var res struct {
			Rows []struct {
				ID       int  `json:"id"`
				Talangan bool `json:"talangan"`
			} `json:"rows"`
		}
		jsonKe(t, w, &res)
		return res.Rows
	}
	ada := func(rows []struct {
		ID       int  `json:"id"`
		Talangan bool `json:"talangan"`
	}) (ketemu, bertanda bool) {
		for _, x := range rows {
			if x.ID == stokID {
				return true, x.Talangan
			}
		}
		return false, false
	}
	if k, b := ada(ambil("")); !k || !b {
		t.Errorf("tanpa filter: barang talangan harus tampil bertanda (ketemu %v bertanda %v)", k, b)
	}
	if k, _ := ada(ambil("&talangan=0")); k {
		t.Error("talangan=0 tidak boleh memuat barang talangan")
	}
	rows1 := ambil("&talangan=1")
	if k, _ := ada(rows1); !k {
		t.Error("talangan=1 harus memuat barang talangan")
	}
	for _, x := range rows1 {
		if !x.Talangan {
			t.Errorf("talangan=1 memuat barang biasa id %d", x.ID)
		}
	}
}

func TestBarangTalanganDitolakOlehOperBPLama(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	_, stokID := masukDiBO(t, a, 400000)
	r := mesinBarang(db, "owner", a.User)
	cek := func(nama, metode, path, body string) {
		w := panggil(r, metode, path, body)
		if w.Code != http.StatusBadRequest || !strings.Contains(strings.ToLower(w.Body.String()), "talangan") {
			t.Errorf("%s: kode %d %s, seharusnya 400 menyebut talangan", nama, w.Code, w.Body.String())
		}
	}
	cek("oper ke BP", "POST", "/barang/oper", fmt.Sprintf(`{"item_ids":[%d],"bp_nama":"BP Iss"}`, stokID))
	cek("tarik dari BP", "POST", "/barang/tarik-bp", fmt.Sprintf(`{"item_ids":[%d]}`, stokID))
	cek("jual biasa", "POST", "/barang/jual", fmt.Sprintf(`{"items":[{"item_id":%d,"harga":900000}]}`, stokID))
	// status barang tidak berubah
	var st string
	db.QueryRow("SELECT status FROM items WHERE id = ?", stokID).Scan(&st)
	if st != "tersedia" {
		t.Errorf("status barang berubah jadi %s", st)
	}
}

func TestVoidKasDanBarangTalanganDitolak(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	id, stokID := masukDiBO(t, a, 600000)
	var kasID int
	db.QueryRow("SELECT kas_id FROM talangan_barang WHERE id = ?", id).Scan(&kasID)
	r := mesinBarang(db, "super_admin", a.User)
	if w := panggil(r, "POST", fmt.Sprintf("/kas/%d/void", kasID), `{"reason":"coba batal"}`); w.Code != http.StatusBadRequest {
		t.Errorf("void kas pembelian talangan: kode %d %s, seharusnya 400", w.Code, w.Body.String())
	}
	if w := panggil(r, "POST", fmt.Sprintf("/barang/void/%d", stokID), `{"reason":"coba batal"}`); w.Code != http.StatusBadRequest {
		t.Errorf("void barang talangan: kode %d %s, seharusnya 400", w.Code, w.Body.String())
	}
	var voidKas, voidItem bool
	db.QueryRow("SELECT is_void FROM cash_transactions WHERE id = ?", kasID).Scan(&voidKas)
	db.QueryRow("SELECT is_void FROM items WHERE id = ?", stokID).Scan(&voidItem)
	if voidKas || voidItem {
		t.Errorf("data ikut ter-void: kas %v barang %v", voidKas, voidItem)
	}
}

// harga beli terkunci: edit barang biasa (super admin) tidak boleh dipakai untuk mengubah barang talangan
func TestEditBarangTalanganDitolakAgarHargaTerkunci(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	id, stokID := masukDiBO(t, a, 800000)
	r := mesinBarang(db, "super_admin", a.User)
	body := `{"item_name":"Gelang uji","karat":"18K","weight_gram":4.5,"harga_beli":1}`
	if w := panggil(r, "PUT", fmt.Sprintf("/barang/%d", stokID), body); w.Code != http.StatusBadRequest {
		t.Errorf("edit barang talangan: kode %d %s, seharusnya 400", w.Code, w.Body.String())
	}
	var harga, hargaItem int64
	db.QueryRow("SELECT CAST(harga_beli AS SIGNED) FROM talangan_barang WHERE id = ?", id).Scan(&harga)
	db.QueryRow("SELECT CAST(harga_beli AS SIGNED) FROM items WHERE id = ?", stokID).Scan(&hargaItem)
	if harga != 800000 || hargaItem != 800000 {
		t.Errorf("harga berubah: talangan %d barang %d", harga, hargaItem)
	}
}

// barang BP (MASUK) hanya boleh dicuci, tidak boleh dilebur; barang kita dari BP (KELUAR) stok biasa
func TestBarangTalanganHanyaBolehDicuciBukanDilebur(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	_, stokID := masukDiBO(t, a, 450000)
	r := mesinBarang(db, "owner", a.User)
	body := fmt.Sprintf(`{"item_ids":[%d]}`, stokID)
	w := panggil(r, "POST", "/lebur", body)
	if w.Code != http.StatusBadRequest || !strings.Contains(strings.ToLower(w.Body.String()), "dicuci") {
		t.Errorf("lebur barang talangan: kode %d %s, seharusnya 400 dan menyebut hanya boleh dicuci", w.Code, w.Body.String())
	}
	var st string
	db.QueryRow("SELECT status FROM items WHERE id = ?", stokID).Scan(&st)
	if st != "tersedia" {
		t.Errorf("barang berubah jadi %s", st)
	}
	if w := panggil(r, "POST", "/cuci", body); w.Code != http.StatusOK && w.Code != http.StatusCreated {
		t.Errorf("cuci barang talangan harus tetap boleh: kode %d %s", w.Code, w.Body.String())
	}
	// barang KELUAR adalah stok kita: tidak ditolak karena alasan talangan
	k := keluarUji(t, a, 700000)
	var stokK int
	db.QueryRow("SELECT stok_id FROM talangan_barang WHERE id = ?", k).Scan(&stokK)
	w = panggil(r, "POST", "/lebur", fmt.Sprintf(`{"item_ids":[%d]}`, stokK))
	if strings.Contains(strings.ToLower(w.Body.String()), "talangan") {
		t.Errorf("barang KELUAR tidak boleh ditolak sebagai talangan: %d %s", w.Code, w.Body.String())
	}
}

// owner melihat badge bila ada barang talangan berstatus KIRIM_BO yang menunggu konfirmasinya
func TestNotifOwnerTalanganKirimBO(t *testing.T) {
	db := dbUji(t)
	a := akunDariDB(t, db)
	hitung := func(role string) int {
		w := panggil(mesinBarang(db, role, a.User), "GET", "/barang/notif", "")
		var n struct {
			TalanganMenunggu int `json:"talangan_menunggu"`
		}
		jsonKe(t, w, &n)
		return n.TalanganMenunggu
	}
	awal := hitung("owner")
	id := masukUji(t, a, 150000)
	if hitung("owner") != awal {
		t.Error("status TALANGAN belum menunggu owner, badge tidak boleh naik")
	}
	panggil(mesinTalangan(db, "pj", a.User, 0), "POST", fmt.Sprintf("/talangan/%d/kirim-bo", id), `{}`)
	if hitung("owner") != awal+1 {
		t.Errorf("badge owner = %d, seharusnya %d", hitung("owner"), awal+1)
	}
	if hitung("cabang") != 0 {
		t.Error("akun cabang tidak boleh mendapat badge talangan")
	}
	panggil(mesinTalangan(db, "owner", a.User, 0), "POST", fmt.Sprintf("/talangan/%d/terima-bo", id), `{}`)
	if hitung("owner") != awal {
		t.Errorf("setelah diterima badge harus turun ke %d, dapat %d", awal, hitung("owner"))
	}
}
