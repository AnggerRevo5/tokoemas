package models

import "time"

// ItemRow dipakai services/controller untuk membaca ringkas barang (kunci, status, lokasi).
type ItemRow struct {
	ID       int
	Name     string
	Karat    string
	Weight   float64
	Location int
	Status   string
}

// Item mencerminkan tabel items (barang per pcs).
type Item struct {
	ID               int        `json:"id"`
	ItemNo           *string    `json:"item_no"`
	ItemName         string     `json:"item_name"`
	Karat            *string    `json:"karat"`
	WeightGram       float64    `json:"weight_gram"`
	Jenis            string     `json:"jenis"` // logam_mulia | perhiasan | hasil_lebur
	Brand            *string    `json:"brand"`
	Kondisi          string     `json:"kondisi"` // kotor | bersih
	HargaBeli        float64    `json:"harga_beli"`
	PurchaseCashID   *int       `json:"purchase_cash_id"`
	OriginLocationID int        `json:"origin_location_id"`
	LocationID       int        `json:"location_id"`
	Status           string     `json:"status"` // tersedia | dikirim | cuci | lebur | terjual | hasil | kembali
	SourceCategory   string     `json:"source_category"`
	EntryDate        time.Time  `json:"entry_date"`
	Note             *string    `json:"note"`
	WashingJobID     *int       `json:"washing_job_id"`
	MeltID           *int       `json:"melt_id"`
	SaleID           *int       `json:"sale_id"`
	HargaJual        *float64   `json:"harga_jual"`
	JualCashID       *int       `json:"jual_cash_id"`
	TanggalJual      *time.Time `json:"tanggal_jual"`
	KirimTgl         *time.Time `json:"kirim_tgl"`
	KirimKeLocation  *int       `json:"kirim_ke_location_id"`
	BelumSampai      bool       `json:"belum_sampai"`
	BelumSampaiNote  *string    `json:"belum_sampai_note"`
	KembaliJenis     *string    `json:"kembali_jenis"` // cuci_selesai | cuci_batal | lebur_batal
	IsVoid           bool       `json:"is_void"`
	VoidReason       *string    `json:"void_reason"`
	CreatedBy        int        `json:"created_by"`
	CreatedAt        time.Time  `json:"created_at"`
}
