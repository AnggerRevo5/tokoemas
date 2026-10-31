package models

import "time"

// JualItem adalah satu barang yang dijual beserta harga jualnya (diisi manual per barang).
type JualItem struct {
	ItemID int     `json:"item_id" binding:"gte=0"`
	Harga  float64 `json:"harga" binding:"lte=1000000000000"`
}

type JualReq struct {
	Items []JualItem `json:"items" binding:"max=1000,dive"`
	Date  string     `json:"date" binding:"max=10"`
	Note  string     `json:"note" binding:"max=200"`
}

type BarangSatu struct {
	Jenis      string  `json:"jenis" binding:"max=20"`
	ItemName   string  `json:"item_name" binding:"max=100"`
	Brand      string  `json:"brand" binding:"max=50"`
	Karat      string  `json:"karat" binding:"max=10"`
	WeightGram float64 `json:"weight_gram" binding:"lte=100000"`
	HargaBeli  float64 `json:"harga_beli" binding:"lte=1000000000000"`
	Note       string  `json:"note" binding:"max=200"`
	TrayID     int     `json:"tray_id" binding:"gte=0"`
}

// BarangEditReq adalah payload JSON untuk BarangEdit. TrayID 0 = tanpa baki.
type BarangEditReq struct {
	ItemName   string  `json:"item_name" binding:"max=100"`
	Brand      string  `json:"brand" binding:"max=50"`
	Karat      string  `json:"karat" binding:"max=10"`
	WeightGram float64 `json:"weight_gram" binding:"lte=100000"`
	HargaBeli  float64 `json:"harga_beli" binding:"gte=0,lte=1000000000000"`
	Note       string  `json:"note" binding:"max=200"`
	TrayID     int     `json:"tray_id" binding:"gte=0"`
}

// BakiBaruReq adalah payload JSON untuk BakiCreate.
type BakiBaruReq struct {
	LocationID int `json:"location_id" binding:"gte=0"`
}

// PindahBakiReq adalah payload JSON untuk BakiPindah.
type PindahBakiReq struct {
	ItemIDs []int `json:"item_ids" binding:"max=1000,dive,gt=0"`
	TrayID  int   `json:"tray_id" binding:"gte=0"`
}

type BarangMasukReq struct {
	LocationID int          `json:"location_id" binding:"gte=0"`
	Category   string       `json:"category" binding:"max=30"`
	EntryDate  string       `json:"entry_date" binding:"max=10"`
	Note       string       `json:"note" binding:"max=200"`
	Posisi     string       `json:"posisi" binding:"max=10"`
	Items      []BarangSatu `json:"items" binding:"max=1000,dive"`
}

// BarangListRow adalah satu baris respons JSON dari BarangList.
type BarangListRow struct {
	ID           int        `json:"id"`
	ItemNo       string     `json:"item_no"`
	ItemName     string     `json:"item_name"`
	Karat        string     `json:"karat"`
	WeightGram   float64    `json:"weight_gram"`
	Jenis        string     `json:"jenis"`
	Brand        string     `json:"brand"`
	Kondisi      string     `json:"kondisi"`
	HargaBeli    float64    `json:"harga_beli"`
	LocationID   int        `json:"location_id"`
	Lokasi       string     `json:"lokasi"`
	OriginID     int        `json:"origin_location_id"`
	Asal         string     `json:"asal"`
	Status       string     `json:"status"`
	EntryDate    time.Time  `json:"entry_date"`
	Note         string     `json:"note"`
	KirimTgl     *time.Time `json:"kirim_tgl"`
	BelumSampai  bool       `json:"belum_sampai"`
	BelumNote    string     `json:"belum_sampai_note"`
	Pencuci      string     `json:"pencuci"`
	CuciTgl      *time.Time `json:"cuci_tgl"`
	TujuanID     int        `json:"tujuan_id"`
	Tujuan       string     `json:"tujuan"`
	HargaJual    *float64   `json:"harga_jual"`
	TanggalJual  *time.Time `json:"tanggal_jual"`
	BakiID       int        `json:"baki_id"`
	BakiNo       int        `json:"baki_no"`
	DariCustomer bool       `json:"dari_customer"`
}

type KirimBOReq struct {
	ItemIDs []int  `json:"item_ids" binding:"max=1000,dive,gt=0"`
	Date    string `json:"date" binding:"max=10"`
}

type TerimaReq struct {
	ItemIDs []int  `json:"item_ids" binding:"max=1000,dive,gt=0"`
	Result  string `json:"result" binding:"max=10"` // sampai atau belum
	Date    string `json:"date" binding:"max=10"`
	Note    string `json:"note" binding:"max=200"`
}

// BarangOperRequest adalah payload JSON untuk BarangOper.
type BarangOperRequest struct {
	ItemIDs          []int  `json:"item_ids" binding:"max=1000,dive,gt=0"`
	TargetLocationID int    `json:"target_location_id" binding:"gte=0"`
	Date             string `json:"date" binding:"max=10"`
}

// BarangTerimaTokoRequest adalah payload JSON untuk BarangTerimaToko.
type BarangTerimaTokoRequest struct {
	ItemIDs []int  `json:"item_ids" binding:"max=1000,dive,gt=0"`
	Result  string `json:"result" binding:"max=10"`
	Date    string `json:"date" binding:"max=10"`
	Note    string `json:"note" binding:"max=200"`
}

// BarangTerimaKembaliRequest adalah payload JSON untuk BarangTerimaKembali.
type BarangTerimaKembaliRequest struct {
	ItemIDs []int  `json:"item_ids" binding:"max=1000,dive,gt=0"`
	Result  string `json:"result" binding:"max=10"`
	Date    string `json:"date" binding:"max=10"`
	Note    string `json:"note" binding:"max=200"`
}
