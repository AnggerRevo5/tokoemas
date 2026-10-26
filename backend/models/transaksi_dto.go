package models

import "time"

// VoidTransaksiRequest adalah payload JSON untuk VoidTransaksi.
type VoidTransaksiRequest struct {
	Reason string `json:"reason" binding:"max=250"`
}

// SaldoRow adalah satu baris respons JSON dari Saldo.
type SaldoRow struct {
	LocationID int     `json:"location_id"`
	Name       string  `json:"name"`
	SaldoKas   float64 `json:"saldo_kas"`
	StokGram   float64 `json:"stok_gram"`
	StokPcs    int     `json:"stok_pcs"`
}

type StokReq struct {
	LocationID       int     `json:"location_id" binding:"gte=0"`
	TargetLocationID int     `json:"target_location_id" binding:"gte=0"`
	Direction        string  `json:"direction" binding:"max=10"`
	Category         string  `json:"category" binding:"max=30"`
	ItemName         string  `json:"item_name" binding:"max=100"`
	WeightGram       float64 `json:"weight_gram" binding:"lte=100000"`
	Karat            string  `json:"karat" binding:"max=10"`
	Qty              int     `json:"qty" binding:"gte=0,lte=10000"`
	Note             string  `json:"note" binding:"max=200"`
	Date             string  `json:"transaction_date" binding:"max=10"`
}

type KasReq struct {
	LocationID int     `json:"location_id" binding:"gte=0"`
	Direction  string  `json:"direction" binding:"max=10"`
	Category   string  `json:"category" binding:"max=30"`
	Amount     float64 `json:"amount" binding:"lte=1000000000000"`
	Note       string  `json:"note" binding:"max=200"`
	Date       string  `json:"transaction_date" binding:"max=10"`
}

// KasListRow adalah satu baris respons JSON dari KasList.
type KasListRow struct {
	ID         int       `json:"id"`
	LocationID int       `json:"location_id"`
	Direction  string    `json:"direction"`
	Category   string    `json:"category"`
	Amount     float64   `json:"amount"`
	Note       string    `json:"note"`
	Date       time.Time `json:"transaction_date"`
	CreatedAt  time.Time `json:"created_at"`
}

// StokListRow adalah satu baris respons JSON dari StokList.
type StokListRow struct {
	ID         int       `json:"id"`
	LocationID int       `json:"location_id"`
	Direction  string    `json:"direction"`
	Category   string    `json:"category"`
	ItemName   string    `json:"item_name"`
	WeightGram float64   `json:"weight_gram"`
	Karat      string    `json:"karat"`
	Qty        int       `json:"qty"`
	Note       string    `json:"note"`
	Date       time.Time `json:"transaction_date"`
}
