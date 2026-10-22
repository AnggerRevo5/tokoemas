package models

import "time"

// WashingJob mencerminkan tabel washing_jobs.
type WashingJob struct {
	ID               int        `json:"id"`
	OriginLocationID int        `json:"origin_location_id"`
	ItemName         string     `json:"item_name"`
	WeightGram       float64    `json:"weight_gram"`
	Karat            *string    `json:"karat"`
	Qty              int        `json:"qty"`
	WasherName       string     `json:"washer_name"`
	Status           string     `json:"status"` // menunggu | proses | selesai | batal
	StartDate        time.Time  `json:"start_date"`
	WasherSentDate   *time.Time `json:"washer_sent_date"`
	SentBy           *int       `json:"sent_by"`
	FinishDate       *time.Time `json:"finish_date"`
	Note             *string    `json:"note"`
	CancelReason     *string    `json:"cancel_reason"`
	CancelledBy      *int       `json:"cancelled_by"`
	CancelledDate    *time.Time `json:"cancelled_date"`
	CancelReturned   *int       `json:"cancel_returned"`
	OutStockID       *int       `json:"out_stock_id"`
	InStockID        *int       `json:"in_stock_id"`
	IsVoid           bool       `json:"is_void"`
	VoidReason       *string    `json:"void_reason"`
	CreatedBy        int        `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
}

// WashingJobItem mencerminkan tabel washing_job_items (berat awal dan sesudah per barang).
type WashingJobItem struct {
	ID           int      `json:"id"`
	JobID        int      `json:"job_id"`
	ItemID       int      `json:"item_id"`
	WeightSystem float64  `json:"weight_system"`
	WeightBefore *float64 `json:"weight_before"`
	WeightAfter  *float64 `json:"weight_after"`
}
