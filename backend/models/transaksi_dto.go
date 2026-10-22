package models

import "time"

// VoidTransaksiRequest adalah payload JSON untuk VoidTransaksi.
type VoidTransaksiRequest struct {
	Reason string `json:"reason"`
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
	LocationID       int     `json:"location_id"`
	TargetLocationID int     `json:"target_location_id"`
	Direction        string  `json:"direction"`
	Category         string  `json:"category"`
	ItemName         string  `json:"item_name"`
	WeightGram       float64 `json:"weight_gram"`
	Karat            string  `json:"karat"`
	Qty              int     `json:"qty"`
	Note             string  `json:"note"`
	Date             string  `json:"transaction_date"`
}

type KasReq struct {
	LocationID int     `json:"location_id"`
	Direction  string  `json:"direction"`
	Category   string  `json:"category"`
	Amount     float64 `json:"amount"`
	Note       string  `json:"note"`
	Date       string  `json:"transaction_date"`
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
