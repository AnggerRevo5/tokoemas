package models

import "time"

// MeltBatch mencerminkan tabel melt_batches.
type MeltBatch struct {
	ID            int        `json:"id"`
	StartDate     time.Time  `json:"start_date"`
	Status        string     `json:"status"` // proses | leburan_kotor | cukim | terjual | batal
	ItemCount     int        `json:"item_count"`
	SourceWeight  float64    `json:"source_weight"`
	Modal         float64    `json:"modal"`
	CutWeight     *float64   `json:"cut_weight"`
	CutNote       *string    `json:"cut_note"`
	LeburanWeight *float64   `json:"leburan_weight"`
	LeburanDate   *time.Time `json:"leburan_date"`
	LeburanBy     *int       `json:"leburan_by"`
	LeburanItemID *int       `json:"leburan_item_id"`
	CukimWeight   *float64   `json:"cukim_weight"`
	CukimKarat    *string    `json:"cukim_karat"`
	CukimDate     *time.Time `json:"cukim_date"`
	CukimBy       *int       `json:"cukim_by"`
	CukimItemID   *int       `json:"cukim_item_id"`
	Note          *string    `json:"note"`
	CancelReason  *string    `json:"cancel_reason"`
	CancelledBy   *int       `json:"cancelled_by"`
	CancelledDate *time.Time `json:"cancelled_date"`
	IsVoid        bool       `json:"is_void"`
	VoidReason    *string    `json:"void_reason"`
	CreatedBy     int        `json:"created_by"`
	CreatedAt     time.Time  `json:"created_at"`
}

// PenadahSale mencerminkan tabel penadah_sales (penjualan hasil lebur ke penadah).
type PenadahSale struct {
	ID          int       `json:"id"`
	SaleDate    time.Time `json:"sale_date"`
	PenadahName string    `json:"penadah_name"`
	SaleTotal   float64   `json:"sale_total"`
	Modal       float64   `json:"modal"`
	WeightGram  float64   `json:"weight_gram"`
	ItemCount   int       `json:"item_count"`
	Note        *string   `json:"note"`
	CashID      *int      `json:"cash_id"`
	IsVoid      bool      `json:"is_void"`
	VoidReason  *string   `json:"void_reason"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}
