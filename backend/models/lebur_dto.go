package models

import "time"

// LeburKembaliRequest adalah payload JSON untuk LeburKembali.
type LeburKembaliRequest struct {
	Reason string `json:"reason" binding:"max=250"`
	Date   string `json:"date" binding:"max=10"`
}

// LeburMulaiRequest adalah payload JSON untuk LeburMulai.
type LeburMulaiRequest struct {
	ItemIDs   []int  `json:"item_ids" binding:"max=1000,dive,gt=0"`
	StartDate string `json:"start_date" binding:"max=10"`
	Note      string `json:"note" binding:"max=200"`
}

// LeburCatatLeburanRequest adalah payload JSON untuk LeburCatatLeburan.
type LeburCatatLeburanRequest struct {
	CutWeight     float64 `json:"cut_weight" binding:"lte=100000"`
	LeburanWeight float64 `json:"leburan_weight" binding:"lte=100000"`
	Date          string  `json:"date" binding:"max=10"`
	Note          string  `json:"note" binding:"max=200"`
}

// LeburCatatCukimRequest adalah payload JSON untuk LeburCatatCukim.
type LeburCatatCukimRequest struct {
	CukimWeight float64 `json:"cukim_weight" binding:"lte=100000"`
	Karat       string  `json:"karat" binding:"max=10"`
	Date        string  `json:"date" binding:"max=10"`
	Note        string  `json:"note" binding:"max=200"`
}

// LeburListRow adalah satu baris respons JSON dari LeburList.
type LeburListRow struct {
	ID            int        `json:"id"`
	StartDate     time.Time  `json:"start_date"`
	Status        string     `json:"status"`
	ItemCount     int        `json:"item_count"`
	SourceWeight  float64    `json:"source_weight"`
	Modal         float64    `json:"modal"`
	CutWeight     *float64   `json:"cut_weight"`
	LeburanWeight *float64   `json:"leburan_weight"`
	CukimWeight   *float64   `json:"cukim_weight"`
	CukimKarat    string     `json:"cukim_karat"`
	Note          string     `json:"note"`
	CutNote       string     `json:"cut_note"`
	LeburanDate   *time.Time `json:"leburan_date"`
	CukimDate     *time.Time `json:"cukim_date"`
	LeburanBy     string     `json:"leburan_by"`
	CukimBy       string     `json:"cukim_by"`
	Barang        string     `json:"barang"`
	SaleTotal     *float64   `json:"sale_total"`
	Penadah       string     `json:"penadah"`
	SaleDate      *time.Time `json:"sale_date"`
	Untung        *float64   `json:"untung"`
	Susut         *float64   `json:"susut"`
}
