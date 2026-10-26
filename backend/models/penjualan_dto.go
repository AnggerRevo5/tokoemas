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
	ItemIDs     []int   `json:"item_ids" binding:"max=1000,dive,gt=0"`
	PenadahName string  `json:"penadah_name" binding:"max=100"`
	SaleTotal   float64 `json:"sale_total" binding:"lte=1000000000000"`
	SaleDate    string  `json:"sale_date" binding:"max=10"`
	Note        string  `json:"note" binding:"max=200"`
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
