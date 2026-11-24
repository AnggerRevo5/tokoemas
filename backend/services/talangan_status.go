package services

import "database/sql"

const (
	ArahMasuk  = "MASUK"
	ArahKeluar = "KELUAR"

	StTalangan    = "TALANGAN"
	StKirimBO     = "KIRIM_BO"
	StDiBO        = "DI_BO"
	StDikirimKeBP = "DIKIRIM_KE_BP"
	StTercatat    = "TERCATAT"
	StDihitung    = "DIHITUNG"
	StLunas       = "LUNAS"
)

// langkah[arah][dari] = status berikutnya yang sah; tiap status hanya punya satu langkah maju (tidak boleh loncat).
var langkah = map[string]map[string]string{
	ArahMasuk: {
		StTalangan:    StKirimBO,
		StKirimBO:     StDiBO,
		StDiBO:        StDikirimKeBP,
		StDikirimKeBP: StDihitung,
		StDihitung:    StLunas,
	},
	ArahKeluar: {
		StTercatat: StDihitung,
		StDihitung: StLunas,
	},
}

// LangkahValid: apakah perpindahan dari -> ke diizinkan untuk arah itu.
func LangkahValid(arah, dari, ke string) bool {
	s, ok := langkah[arah][dari]
	return ok && s == ke
}

// PerananUntuk: role yang boleh memicu langkah ke status itu secara manual. Kosong = hanya sistem
// (DIHITUNG dan LUNAS terjadi otomatis dari penukaran dan pembayaran). super_admin boleh semua langkah manual.
func PerananUntuk(arah, ke string) []string {
	switch ke {
	case StKirimBO, StDikirimKeBP:
		return []string{"pj", "super_admin"}
	case StDiBO:
		return []string{"owner", "super_admin"}
	case StTercatat:
		return []string{"pj", "super_admin"}
	}
	return nil
}

// CatatStatus menambah satu baris log perpindahan status. Log hanya di-INSERT, tidak pernah diubah atau dihapus.
func CatatStatus(tx *sql.Tx, id int, dari, ke string, oleh int, catatan string) error {
	_, err := tx.Exec(
		"INSERT INTO talangan_status_log (talangan_id, dari, ke, oleh, catatan) VALUES (?,?,?,?,?)",
		id, dari, ke, oleh, catatan)
	return err
}
