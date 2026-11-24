package models

import "time"

// TalanganCreateReq: petugas toko mencatat barang milik BP lain yang dibeli dari customer (talangan MASUK).
// HargaBeli dalam rupiah bulat (int64), tidak pernah float.
type TalanganCreateReq struct {
	LocationID int     `json:"location_id" binding:"gte=0"`
	BPID       int     `json:"bp_id" binding:"gt=0"`
	NamaBarang string  `json:"nama_barang" binding:"required,max=100"`
	Jenis      string  `json:"jenis" binding:"max=20"` // perhiasan | logam_mulia
	Bahan      string  `json:"bahan" binding:"max=5"`  // emas | perak
	Karat      string  `json:"karat" binding:"max=10"`
	Berat      float64 `json:"berat" binding:"gt=0,lte=100000"`
	HargaBeli  int64   `json:"harga_beli" binding:"gt=0,lte=100000000000"`
	Tanggal    string  `json:"tanggal" binding:"max=10"`
	NoSurat    string  `json:"no_surat" binding:"max=60"`
	Catatan    string  `json:"catatan" binding:"max=200"`
}

// TalanganKeluarReq: PJ mengambil barang milik kita yang dibeli BP dari customer, lalu mencatatnya (talangan KELUAR).
type TalanganKeluarReq struct {
	BPID        int     `json:"bp_id" binding:"gt=0"`
	NamaBarang  string  `json:"nama_barang" binding:"required,max=100"`
	Jenis       string  `json:"jenis" binding:"max=20"`
	Bahan       string  `json:"bahan" binding:"max=5"`
	Karat       string  `json:"karat" binding:"max=10"`
	Berat       float64 `json:"berat" binding:"gt=0,lte=100000"`
	HargaBeli   int64   `json:"harga_beli" binding:"gt=0,lte=100000000000"` // harga yang BP bayar ke customer
	Tanggal     string  `json:"tanggal" binding:"max=10"`
	NoReferensi string  `json:"no_referensi" binding:"max=60"` // nomor nota BP, opsional
	Catatan     string  `json:"catatan" binding:"max=200"`
}

// TalanganRow: satu baris daftar talangan.
type TalanganRow struct {
	ID          int       `json:"id"`
	Arah        string    `json:"arah"`
	BPID        int       `json:"bp_id"`
	BP          string    `json:"bp"`
	TokoInputID int       `json:"toko_input_id"`
	TokoInput   string    `json:"toko_input"`
	StokID      int       `json:"stok_id"`
	ProsesID    int       `json:"proses_id"`
	ProsesJenis string    `json:"proses_jenis"`
	NamaBarang  string    `json:"nama_barang"`
	Jenis       string    `json:"jenis"`
	Bahan       string    `json:"bahan"`
	Berat       float64   `json:"berat"`
	Kadar       string    `json:"kadar"`
	HargaBeli   int64     `json:"harga_beli"`
	UbahHarga   bool      `json:"ubah_harga"`
	TanggalBeli string    `json:"tanggal_beli"`
	NoSurat     string    `json:"no_surat"`
	NoReferensi string    `json:"no_referensi"`
	Status      string    `json:"status"`
	TukarID     int       `json:"tukar_id"`
	Catatan     string    `json:"catatan"`
	DiinputOleh string    `json:"diinput_oleh"`
	DibuatPada  time.Time `json:"created_at"`
}

// TukarCreateReq: satu penukaran = satu cabang BP, barang MASUK (di BO) dan barang KELUAR yang diambil dari BP.
type TukarCreateReq struct {
	BPID      int    `json:"bp_id" binding:"gt=0"`
	MasukIDs  []int  `json:"masuk_ids" binding:"max=300,dive,gt=0"`
	KeluarIDs []int  `json:"keluar_ids" binding:"max=300,dive,gt=0"`
	Catatan   string `json:"catatan" binding:"max=200"`
}

// TukarBayarReq: nominal pembayaran (rupiah bulat) dan nomor referensi transfer (opsional).
type TukarBayarReq struct {
	Nominal     int64  `json:"nominal" binding:"gt=0,lte=100000000000"`
	NoReferensi string `json:"no_referensi" binding:"max=60"`
}

// KoreksiNominalReq: owner menyetujui nominal berbeda dari saldo, alasan wajib.
type KoreksiNominalReq struct {
	Nominal int64  `json:"nominal" binding:"gt=0,lte=100000000000"`
	Alasan  string `json:"alasan" binding:"required,max=250"`
}

// TukarItemRow: barang dalam satu penukaran.
type TukarItemRow struct {
	BarangID   int    `json:"barang_id"`
	Arah       string `json:"arah"`
	NamaBarang string `json:"nama_barang"`
	Nilai      int64  `json:"nilai"`
	Status     string `json:"status"`
}

// TukarRow: satu penukaran (daftar dan detail). TanpaBukti = transfer tanpa nomor referensi.
type TukarRow struct {
	ID            int            `json:"id"`
	BPID          int            `json:"bp_id"`
	BP            string         `json:"bp"`
	TotalMasuk    int64          `json:"total_masuk"`
	TotalKeluar   int64          `json:"total_keluar"`
	Saldo         int64          `json:"saldo"`
	ArahUang      string         `json:"arah_uang"`
	MetodeBayar   string         `json:"metode_bayar"`
	NominalBayar  int64          `json:"nominal_bayar"`
	NoReferensi   string         `json:"no_referensi"`
	StatusUang    string         `json:"status_uang"`
	TanpaBukti    bool           `json:"tanpa_bukti"`
	KoreksiAlasan string         `json:"koreksi_alasan"`
	PJ            string         `json:"pj"`
	Tanggal       string         `json:"tanggal"`
	JumlahBarang  int            `json:"jumlah_barang"`
	Catatan       string         `json:"catatan"`
	Items         []TukarItemRow `json:"items,omitempty"`
}

// KoreksiHargaReq: hanya owner, alasan wajib.
type KoreksiHargaReq struct {
	HargaBaru int64  `json:"harga_baru" binding:"gt=0,lte=100000000000"`
	Alasan    string `json:"alasan" binding:"required,max=250"`
}
