package models

import "time"

type ServiceReq struct {
	LocationID    int     `json:"location_id"`
	CustomerName  string  `json:"customer_name"`
	CustomerPhone string  `json:"customer_phone"`
	ItemName      string  `json:"item_name"`
	Karat         string  `json:"karat"`
	WeightGram    float64 `json:"weight_gram"`
	Complaint     string  `json:"complaint"`
	Note          string  `json:"note"`
	ReceivedDate  string  `json:"received_date"`
	Posisi        string  `json:"posisi"`
}

// ServiceListRow adalah satu baris respons JSON dari ServiceList.
type ServiceListRow struct {
	ID              int        `json:"id"`
	JobNo           string     `json:"job_no"`
	CustomerName    string     `json:"customer_name"`
	CustomerPhone   string     `json:"customer_phone"`
	ItemName        string     `json:"item_name"`
	Karat           string     `json:"karat"`
	WeightGram      float64    `json:"weight_gram"`
	Complaint       string     `json:"complaint"`
	Note            string     `json:"note"`
	OriginID        int        `json:"origin_id"`
	Origin          string     `json:"origin"`
	LocationID      int        `json:"location_id"`
	Location        string     `json:"location"`
	PickupID        int        `json:"pickup_id"`
	Pickup          string     `json:"pickup"`
	Status          string     `json:"status"`
	ReceivedDate    time.Time  `json:"received_date"`
	SentBODate      *time.Time `json:"sent_bo_date"`
	ArrivedBODate   *time.Time `json:"arrived_bo_date"`
	BelumSampai     bool       `json:"belum_sampai"`
	BelumSampaiNote string     `json:"belum_sampai_note"`
	ServiceName     string     `json:"service_name"`
	ServiceStart    *time.Time `json:"service_start_date"`
	ServiceEnd      *time.Time `json:"service_end_date"`
	WeightAfter     *float64   `json:"weight_after"`
	ServiceNote     string     `json:"service_note"`
	SentTokoDate    *time.Time `json:"sent_toko_date"`
	PickupDate      *time.Time `json:"pickup_date"`
	ReturnedBODate  *time.Time `json:"returned_bo_date"`
	ServiceFee      *float64   `json:"service_fee"`
}

// ServiceAksiRequest adalah payload JSON untuk ServiceAksi.
type ServiceAksiRequest struct {
	Action      string  `json:"action"`
	IDs         []int   `json:"ids"`
	Date        string  `json:"date"`
	ServiceName string  `json:"service_name"`
	TargetLoc   int     `json:"target_location_id"`
	WeightAfter float64 `json:"weight_after"`
	Note        string  `json:"note"`
	// biaya service per barang (id barang service -> rupiah), wajib untuk aksi diambil; 0 = tanpa biaya
	Fees map[int]float64 `json:"fees"`
}

// ServiceRingkasRow adalah satu baris respons JSON dari ServiceRingkas.
type ServiceRingkasRow struct {
	LocationID     int    `json:"location_id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	DiSini         int    `json:"di_sini"`
	DiToko         int    `json:"di_toko"`
	MenungguServis int    `json:"menunggu_servis"`
	Diproses       int    `json:"diproses"`
	KembaliBO      int    `json:"kembali_bo"`
	Selesai        int    `json:"selesai"`
	SiapDiambil    int    `json:"siap_diambil"`
	Keluar         int    `json:"keluar"`
	Masuk          int    `json:"masuk"`
}
