package models

import "time"

// ServiceJob mencerminkan tabel service_jobs (barang service milik customer).
type ServiceJob struct {
	ID               int        `json:"id"`
	JobNo            *string    `json:"job_no"`
	CustomerName     string     `json:"customer_name"`
	CustomerPhone    *string    `json:"customer_phone"`
	ItemName         string     `json:"item_name"`
	Karat            *string    `json:"karat"`
	WeightGram       float64    `json:"weight_gram"`
	Complaint        string     `json:"complaint"`
	Note             *string    `json:"note"`
	OriginLocationID int        `json:"origin_location_id"`
	LocationID       int        `json:"location_id"`
	PickupLocationID *int       `json:"pickup_location_id"`
	Status           string     `json:"status"` // di_toko | dikirim_bo | di_bo | diproses | kembali_bo | selesai | dikirim_toko | siap_diambil | diambil
	ReceivedDate     time.Time  `json:"received_date"`
	SentBODate       *time.Time `json:"sent_bo_date"`
	ArrivedBODate    *time.Time `json:"arrived_bo_date"`
	ReturnedBODate   *time.Time `json:"returned_bo_date"`
	BelumSampai      bool       `json:"belum_sampai"`
	BelumSampaiNote  *string    `json:"belum_sampai_note"`
	ServiceName      *string    `json:"service_name"`
	ServiceStartDate *time.Time `json:"service_start_date"`
	ServiceEndDate   *time.Time `json:"service_end_date"`
	WeightAfter      *float64   `json:"weight_after"`
	ServiceNote      *string    `json:"service_note"`
	ServiceFee       *float64   `json:"service_fee"`
	FeeCashID        *int       `json:"fee_cash_id"`
	SentTokoDate     *time.Time `json:"sent_toko_date"`
	PickupDate       *time.Time `json:"pickup_date"`
	IsVoid           bool       `json:"is_void"`
	VoidReason       *string    `json:"void_reason"`
	CreatedBy        int        `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
}
