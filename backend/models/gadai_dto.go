package models

// GadaiBaruReq adalah payload JSON untuk GadaiCreate. Nominal dalam rupiah bulat.
// JatuhTempoPertama kosong berarti satu bulan setelah tanggal gadai.
type GadaiBaruReq struct {
	ItemID            int    `json:"item_id" binding:"gte=0"`
	Pegadaian         string `json:"pegadaian" binding:"max=100"`
	Dana              int64  `json:"dana" binding:"gte=0,lte=100000000000"`
	Tenor             int    `json:"tenor" binding:"gte=0,lte=120"`
	Cicilan           int64  `json:"cicilan" binding:"gte=0,lte=100000000000"`
	Date              string `json:"date" binding:"max=10"`
	JatuhTempoPertama string `json:"jatuh_tempo_pertama" binding:"max=10"`
	Note              string `json:"note" binding:"max=200"`
}

// GadaiBayarReq dipakai untuk membayar cicilan dan untuk menebus lebih cepat (Diskon hanya untuk tebus).
// Denda adalah bunga keterlambatan dan tidak mengurangi sisa kewajiban.
type GadaiBayarReq struct {
	Date   string `json:"date" binding:"max=10"`
	Amount int64  `json:"amount" binding:"gte=0,lte=100000000000"`
	Denda  int64  `json:"denda" binding:"gte=0,lte=100000000000"`
	Diskon int64  `json:"diskon" binding:"gte=0,lte=100000000000"`
	Note   string `json:"note" binding:"max=200"`
}

// GadaiAkhirReq dipakai untuk "barang sudah kembali ke toko" dan "tidak ditebus" (alasan wajib).
type GadaiAkhirReq struct {
	Date   string `json:"date" binding:"max=10"`
	Reason string `json:"reason" binding:"max=250"`
}

// GadaiRow adalah satu gadai beserta angka ringkasnya.
type GadaiRow struct {
	ID           int     `json:"id"`
	ItemID       int     `json:"item_id"`
	ItemNo       string  `json:"item_no"`
	ItemName     string  `json:"item_name"`
	Karat        string  `json:"karat"`
	Brand        string  `json:"brand"`
	WeightGram   float64 `json:"weight_gram"`
	LocationID   int     `json:"location_id"`
	Lokasi       string  `json:"lokasi"`
	Pegadaian    string  `json:"pegadaian"`
	Dana         int64   `json:"dana"`
	Tenor        int     `json:"tenor"`
	Cicilan      int64   `json:"cicilan"`
	Total        int64   `json:"total"`
	StartDate    string  `json:"start_date"`
	FirstDue     string  `json:"first_due"`
	Status       string  `json:"status"`
	Note         string  `json:"note"`
	EndDate      string  `json:"end_date"`
	EndNote      string  `json:"end_note"`
	Dibayar      int64   `json:"dibayar"`
	Denda        int64   `json:"denda"`
	Diskon       int64   `json:"diskon"`
	PokokDibayar int64   `json:"pokok_dibayar"`
	BulanDibayar int     `json:"bulan_dibayar"`
	Sisa         int64   `json:"sisa"`
	SisaBulan    int     `json:"sisa_bulan"`
	JatuhTempo   string  `json:"jatuh_tempo_berikut"`
	Terlambat    bool    `json:"terlambat"`
	BisaUbah     bool    `json:"bisa_ubah"` // akun ini boleh membatalkan gadai ini
}

// GadaiBayarRow adalah satu catatan pembayaran.
type GadaiBayarRow struct {
	ID         int    `json:"id"`
	Jenis      string `json:"jenis"`
	TxDate     string `json:"tx_date"`
	Amount     int64  `json:"amount"`
	Denda      int64  `json:"denda"`
	Diskon     int64  `json:"diskon"`
	Pokok      int64  `json:"pokok"`
	Biaya      int64  `json:"biaya"`
	Note       string `json:"note"`
	IsVoid     bool   `json:"is_void"`
	VoidReason string `json:"void_reason"`
	Pencatat   string `json:"pencatat"`
	BisaUbah   bool   `json:"bisa_ubah"`
}

// GadaiJadwal adalah satu baris jadwal cicilan.
type GadaiJadwal struct {
	Ke         int    `json:"ke"`
	JatuhTempo string `json:"jatuh_tempo"`
	Dibayar    bool   `json:"dibayar"`
	TglBayar   string `json:"tgl_bayar"`
	Via        string `json:"via"` // cicilan | pelunasan
}

// GadaiDetail adalah gadai lengkap dengan jadwal dan riwayat pembayarannya.
type GadaiDetail struct {
	GadaiRow
	Jadwal     []GadaiJadwal   `json:"jadwal"`
	Pembayaran []GadaiBayarRow `json:"pembayaran"`
}
