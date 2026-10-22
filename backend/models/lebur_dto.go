package models

import "time"

// LeburKembaliRequest adalah payload JSON untuk LeburKembali.
type LeburKembaliRequest struct {
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

// LeburMulaiRequest adalah payload JSON untuk LeburMulai.
type LeburMulaiRequest struct {
	ItemIDs   []int  `json:"item_ids"`
	StartDate string `json:"start_date"`
	Note      string `json:"note"`
}

// LeburCatatLeburanRequest adalah payload JSON untuk LeburCatatLeburan.
type LeburCatatLeburanRequest struct {
	CutWeight     float64 `json:"cut_weight"`
	LeburanWeight float64 `json:"leburan_weight"`
	Date          string  `json:"date"`
	Note          string  `json:"note"`
}

// LeburCatatCukimRequest adalah payload JSON untuk LeburCatatCukim.
type LeburCatatCukimRequest struct {
	CukimWeight float64 `json:"cukim_weight"`
	Karat       string  `json:"karat"`
	Date        string  `json:"date"`
	Note        string  `json:"note"`
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
