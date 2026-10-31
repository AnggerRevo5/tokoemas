package models

// InvestorBaruReq adalah payload JSON untuk InvestorCreate: investor baru beserta setoran pertamanya.
// Nominal dalam rupiah bulat; RatePercent kosong berarti 1% per bulan.
type InvestorBaruReq struct {
	Name        string  `json:"name" binding:"max=100"`
	Note        string  `json:"note" binding:"max=200"`
	StartDate   string  `json:"start_date" binding:"max=10"`
	Amount      int64   `json:"amount" binding:"lte=1000000000000"`
	RatePercent float64 `json:"rate_percent" binding:"gte=0,lte=100"`
}

// InvestorGerakReq adalah payload JSON untuk InvestorGerak.
type InvestorGerakReq struct {
	Jenis  string `json:"jenis" binding:"max=20"` // setor | tarik_pokok | bayar_bunga
	Amount int64  `json:"amount" binding:"lte=1000000000000"`
	TxDate string `json:"tx_date" binding:"max=10"`
	Note   string `json:"note" binding:"max=200"`
}

// InvestorAmbilReq adalah payload JSON untuk InvestorAmbil: pokok dan bunga yang diambil sekaligus.
// Salah satunya boleh 0.
type InvestorAmbilReq struct {
	Pokok  int64  `json:"pokok" binding:"lte=1000000000000"`
	Bunga  int64  `json:"bunga" binding:"lte=1000000000000"`
	TxDate string `json:"tx_date" binding:"max=10"`
	Note   string `json:"note" binding:"max=200"`
}

// InvestorRingkasan adalah satu investor beserta angka ringkasnya.
type InvestorRingkasan struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Note         string  `json:"note"`
	StartDate    string  `json:"start_date"`
	RatePercent  float64 `json:"rate_percent"`
	Pokok        int64   `json:"pokok"`
	BungaTotal   int64   `json:"bunga_total"`
	BungaDibayar int64   `json:"bunga_dibayar"`
	BungaBelum   int64   `json:"bunga_belum"`
	JatuhTempo   string  `json:"jatuh_tempo_berikut"`
	BungaBerikut int64   `json:"perkiraan_bunga_berikut"`
	DasarBerikut int64   `json:"dasar_bunga_berikut"`
}

// InvestorGerakRow adalah satu baris riwayat uang investor.
type InvestorGerakRow struct {
	ID         int    `json:"id"`
	Jenis      string `json:"jenis"`
	Amount     int64  `json:"amount"`
	TxDate     string `json:"tx_date"`
	Note       string `json:"note"`
	IsVoid     bool   `json:"is_void"`
	VoidReason string `json:"void_reason"`
	Pencatat   string `json:"pencatat"`
	CreatedAt  string `json:"created_at"`
}

// InvestorBungaRow adalah bunga satu siklus.
type InvestorBungaRow struct {
	Siklus     int    `json:"siklus"`
	Mulai      string `json:"mulai"`
	JatuhTempo string `json:"jatuh_tempo"`
	Dasar      int64  `json:"dasar"`
	Bunga      int64  `json:"bunga"`
}
