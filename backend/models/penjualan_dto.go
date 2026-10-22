package models

import "time"

// PenjualanSiapRow adalah satu baris respons JSON dari PenjualanSiap.
type PenjualanSiapRow struct {
	ID         int     `json:"id"`
	ItemNo     string  `json:"item_no"`
	ItemName   string  `json:"item_name"`
	Karat      string  `json:"karat"`
	WeightGram float64 `json:"weight_gram"`
	Jenis      string  `json:"jenis"`
	Brand      string  `json:"brand"`
	HargaBeli  float64 `json:"harga_beli"`
	BatchID    *int64  `json:"batch_id"`
}

// PenjualanPenadahRequest adalah payload JSON untuk PenjualanPenadah.
type PenjualanPenadahRequest struct {
	ItemIDs     []int   `json:"item_ids"`
	PenadahName string  `json:"penadah_name"`
	SaleTotal   float64 `json:"sale_total"`
	SaleDate    string  `json:"sale_date"`
	Note        string  `json:"note"`
}

// PenjualanListRow adalah satu baris respons JSON dari PenjualanList.
type PenjualanListRow struct {
	ID          int       `json:"id"`
	SaleDate    time.Time `json:"sale_date"`
	PenadahName string    `json:"penadah_name"`
	ItemCount   int       `json:"item_count"`
	WeightGram  float64   `json:"weight_gram"`
	Modal       float64   `json:"modal"`
	SaleTotal   float64   `json:"sale_total"`
	Untung      float64   `json:"untung"`
	Note        string    `json:"note"`
	PJ          string    `json:"pj"`
	Barang      string    `json:"barang"`
}
