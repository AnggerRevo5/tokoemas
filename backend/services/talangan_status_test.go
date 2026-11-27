package services

import "testing"

func TestLangkahValid(t *testing.T) {
	ok := [][3]string{
		{"MASUK", "TALANGAN", "KIRIM_BO"},
		{"MASUK", "KIRIM_BO", "DI_BO"},
		{"MASUK", "DI_BO", "DIKIRIM_KE_BP"},
		{"MASUK", "DIKIRIM_KE_BP", "DIHITUNG"},
		{"MASUK", "DIHITUNG", "LUNAS"},
		{"KELUAR", "TERCATAT", "DI_BO"},
		{"KELUAR", "DI_BO", "DIHITUNG"},
		{"KELUAR", "DIHITUNG", "LUNAS"},
	}
	for _, c := range ok {
		if !LangkahValid(c[0], c[1], c[2]) {
			t.Errorf("%s %s→%s seharusnya valid", c[0], c[1], c[2])
		}
	}
	salah := [][3]string{
		{"MASUK", "TALANGAN", "DI_BO"},     // loncat
		{"MASUK", "TALANGAN", "LUNAS"},     // loncat
		{"MASUK", "DI_BO", "DIHITUNG"},     // loncat
		{"MASUK", "LUNAS", "DIHITUNG"},     // mundur
		{"MASUK", "LUNAS", "LUNAS"},        // akhir
		{"MASUK", "TERCATAT", "DIHITUNG"},  // status milik KELUAR
		{"KELUAR", "TALANGAN", "KIRIM_BO"}, // status milik MASUK
		{"KELUAR", "TERCATAT", "LUNAS"},    // loncat
		{"KELUAR", "TERCATAT", "DIHITUNG"}, // harus lewat konfirmasi owner dulu
		{"LAIN", "TALANGAN", "KIRIM_BO"},   // arah tidak dikenal
		{"MASUK", "KIRIM_BO", "KIRIM_BO"},  // diam di tempat
	}
	for _, c := range salah {
		if LangkahValid(c[0], c[1], c[2]) {
			t.Errorf("%s %s→%s seharusnya ditolak", c[0], c[1], c[2])
		}
	}
}

func TestPerananUntuk(t *testing.T) {
	cocok := func(role string, daftar []string) bool {
		for _, r := range daftar {
			if r == role {
				return true
			}
		}
		return false
	}
	if !cocok("pj", PerananUntuk("MASUK", "KIRIM_BO")) || cocok("cabang", PerananUntuk("MASUK", "KIRIM_BO")) {
		t.Error("KIRIM_BO hanya PJ")
	}
	if !cocok("owner", PerananUntuk("MASUK", "DI_BO")) || cocok("pj", PerananUntuk("MASUK", "DI_BO")) {
		t.Error("DI_BO hanya owner")
	}
	if !cocok("owner", PerananUntuk("KELUAR", "DI_BO")) || cocok("pj", PerananUntuk("KELUAR", "DI_BO")) {
		t.Error("barang kembali (KELUAR) juga dikonfirmasi owner")
	}
	if !cocok("super_admin", PerananUntuk("MASUK", "DI_BO")) {
		t.Error("super_admin boleh semua langkah manual")
	}
	// langkah otomatis (dipicu penukaran/pembayaran) tidak boleh dipicu manual
	if len(PerananUntuk("MASUK", "DIHITUNG")) != 0 || len(PerananUntuk("MASUK", "LUNAS")) != 0 {
		t.Error("DIHITUNG dan LUNAS hanya oleh sistem")
	}
}
