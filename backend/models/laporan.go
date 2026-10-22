package models

type TabelLaporan struct {
	Kode  string          `json:"kode"`
	Judul string          `json:"judul"`
	Kolom []string        `json:"kolom"`
	Tipe  []string        `json:"tipe"`
	Baris [][]interface{} `json:"baris"`
	Tebal []int           `json:"tebal"`
}
