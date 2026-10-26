package models

import "time"

// HutangCreateRequest adalah payload JSON untuk HutangCreate.
type HutangCreateRequest struct {
	CreditorType string  `json:"creditor_type" binding:"max=30"`
	CreditorName string  `json:"creditor_name" binding:"max=100"`
	Principal    float64 `json:"principal" binding:"lte=1000000000000"`
	Note         string  `json:"note" binding:"max=200"`
	StartDate    string  `json:"start_date" binding:"max=10"`
	DueDate      string  `json:"due_date" binding:"max=10"`
}

// HutangTerimaDanaRequest adalah payload JSON untuk HutangTerimaDana.
type HutangTerimaDanaRequest struct {
	CreditorType string  `json:"creditor_type" binding:"max=30"`
	CreditorName string  `json:"creditor_name" binding:"max=100"`
	Principal    float64 `json:"principal" binding:"lte=1000000000000"`
	Note         string  `json:"note" binding:"max=200"`
	StartDate    string  `json:"start_date" binding:"max=10"`
	DueDate      string  `json:"due_date" binding:"max=10"`
}

// HutangListRow adalah satu baris respons JSON dari HutangList.
type HutangListRow struct {
	ID           int        `json:"id"`
	CreditorType string     `json:"creditor_type"`
	CreditorName string     `json:"creditor_name"`
	Principal    float64    `json:"principal"`
	Terbayar     float64    `json:"terbayar"`
	Sisa         float64    `json:"sisa"`
	Note         string     `json:"note"`
	StartDate    time.Time  `json:"start_date"`
	DueDate      *time.Time `json:"due_date"`
	Status       string     `json:"status"`
	DariKas      bool       `json:"dari_kas"`
}

// HutangBayarRequest adalah payload JSON untuk HutangBayar.
type HutangBayarRequest struct {
	Amount   float64 `json:"amount" binding:"lte=1000000000000"`
	Note     string  `json:"note" binding:"max=200"`
	PaidDate string  `json:"paid_date" binding:"max=10"`
}
