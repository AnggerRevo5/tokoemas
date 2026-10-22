package models

import "time"

// JualItem adalah satu barang yang dijual beserta harga jualnya (diisi manual per barang).
type JualItem struct {
	ItemID int     `json:"item_id"`
	Harga  float64 `json:"harga"`
}

type JualReq struct {
	Items []JualItem `json:"items"`
	Date  string     `json:"date"`
	Note  string     `json:"note"`
}

type BarangSatu struct {
	Jenis      string  `json:"jenis"`
	ItemName   string  `json:"item_name"`
	Brand      string  `json:"brand"`
	Karat      string  `json:"karat"`
	WeightGram float64 `json:"weight_gram"`
	HargaBeli  float64 `json:"harga_beli"`
	Note       string  `json:"note"`
}

type BarangMasukReq struct {
	LocationID int          `json:"location_id"`
	Category   string       `json:"category"`
	EntryDate  string       `json:"entry_date"`
	Note       string       `json:"note"`
	Posisi     string       `json:"posisi"`
	Items      []BarangSatu `json:"items"`
}

// BarangListRow adalah satu baris respons JSON dari BarangList.
type BarangListRow struct {
	ID          int        `json:"id"`
	ItemNo      string     `json:"item_no"`
	ItemName    string     `json:"item_name"`
	Karat       string     `json:"karat"`
	WeightGram  float64    `json:"weight_gram"`
	Jenis       string     `json:"jenis"`
	Brand       string     `json:"brand"`
	Kondisi     string     `json:"kondisi"`
	HargaBeli   float64    `json:"harga_beli"`
	LocationID  int        `json:"location_id"`
	Lokasi      string     `json:"lokasi"`
	OriginID    int        `json:"origin_location_id"`
	Asal        string     `json:"asal"`
	Status      string     `json:"status"`
	EntryDate   time.Time  `json:"entry_date"`
	Note        string     `json:"note"`
	KirimTgl    *time.Time `json:"kirim_tgl"`
	BelumSampai bool       `json:"belum_sampai"`
	BelumNote   string     `json:"belum_sampai_note"`
	Pencuci     string     `json:"pencuci"`
	CuciTgl     *time.Time `json:"cuci_tgl"`
	TujuanID    int        `json:"tujuan_id"`
	Tujuan      string     `json:"tujuan"`
	HargaJual   *float64   `json:"harga_jual"`
	TanggalJual *time.Time `json:"tanggal_jual"`
}

type KirimBOReq struct {
	ItemIDs []int  `json:"item_ids"`
	Date    string `json:"date"`
}

type TerimaReq struct {
	ItemIDs []int  `json:"item_ids"`
	Result  string `json:"result"` // sampai atau belum
	Date    string `json:"date"`
	Note    string `json:"note"`
}

// BarangOperRequest adalah payload JSON untuk BarangOper.
type BarangOperRequest struct {
	ItemIDs          []int  `json:"item_ids"`
	TargetLocationID int    `json:"target_location_id"`
	Date             string `json:"date"`
}

// BarangTerimaTokoRequest adalah payload JSON untuk BarangTerimaToko.
type BarangTerimaTokoRequest struct {
	ItemIDs []int  `json:"item_ids"`
	Result  string `json:"result"`
	Date    string `json:"date"`
	Note    string `json:"note"`
}

// BarangTerimaKembaliRequest adalah payload JSON untuk BarangTerimaKembali.
type BarangTerimaKembaliRequest struct {
	ItemIDs []int  `json:"item_ids"`
	Result  string `json:"result"`
	Date    string `json:"date"`
	Note    string `json:"note"`
}
