package models

import "time"

type BeratItem struct {
	ItemID int     `json:"item_id"`
	Weight float64 `json:"weight"`
}

// CuciMulaiRequest adalah payload JSON untuk CuciMulai.
type CuciMulaiRequest struct {
	ItemIDs   []int  `json:"item_ids"`
	StartDate string `json:"start_date"`
	Note      string `json:"note"`
}

// CuciSelesaiBarangLangsungRequest adalah payload JSON untuk CuciSelesaiBarangLangsung.
type CuciSelesaiBarangLangsungRequest struct {
	ItemIDs    []int  `json:"item_ids"`
	FinishDate string `json:"finish_date"`
}

// CuciKirimRequest adalah payload JSON untuk CuciKirim.
type CuciKirimRequest struct {
	WasherName string      `json:"washer_name"`
	Date       string      `json:"date"`
	Weights    []BeratItem `json:"weights"`
}

// CuciSelesaiBarangRequest adalah payload JSON untuk CuciSelesaiBarang.
type CuciSelesaiBarangRequest struct {
	Items []struct {
		ItemID int     `json:"item_id"`
		Weight float64 `json:"weight_after"`
	} `json:"items"`
	FinishDate string `json:"finish_date"`
}

// CuciKembaliRequest adalah payload JSON untuk CuciKembali.
type CuciKembaliRequest struct {
	Reason string `json:"reason"`
	Date   string `json:"date"`
}

// CucianBarangRow adalah satu baris respons JSON dari CucianBarang.
type CucianBarangRow struct {
	ID         int        `json:"id"`
	ItemNo     string     `json:"item_no"`
	ItemName   string     `json:"item_name"`
	Karat      string     `json:"karat"`
	WeightGram float64    `json:"weight_gram"`
	Asal       string     `json:"asal"`
	JobID      int        `json:"job_id"`
	JobStatus  string     `json:"job_status"`
	Pencuci    string     `json:"washer_name"`
	StartDate  time.Time  `json:"start_date"`
	SentDate   *time.Time `json:"sent_date"`
	BeratAwal  float64    `json:"berat_awal"`
}

// CuciListRow adalah satu baris respons JSON dari CuciList.
type CuciListRow struct {
	ID               int        `json:"id"`
	LocationID       int        `json:"origin_location_id"`
	Cabang           string     `json:"cabang"`
	ItemName         string     `json:"item_name"`
	Karat            string     `json:"karat"`
	WeightGram       float64    `json:"weight_gram"`
	Qty              int        `json:"qty"`
	Washer           string     `json:"washer_name"`
	Status           string     `json:"status"`
	StartDate        time.Time  `json:"start_date"`
	FinishDate       *time.Time `json:"finish_date"`
	Note             string     `json:"note"`
	Sisa             int        `json:"sisa"`
	CancelReason     string     `json:"cancel_reason"`
	CancelReturned   int        `json:"cancel_returned"`
	BeratAwalSelesai float64    `json:"berat_awal_selesai"`
	BeratSesudah     float64    `json:"berat_sesudah"`
}
