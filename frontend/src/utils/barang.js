// barang yang boleh dicentang di daftar stok
export const bisaPilih = (r) =>
  r.status === 'tersedia' || r.status === 'dikirim' || r.status === 'cuci' || r.status === 'kembali' || r.status === 'di_bp'

// label jenis barang
export const JENIS_LABEL = {
  perhiasan: 'Perhiasan',
  logam_mulia: 'Logam mulia',
  hasil_lebur: 'Hasil lebur',
  susuk: 'Susuk emas',
  tempat_perhiasan: 'Tempat perhiasan',
  kotak_perhiasan: 'Kotak perhiasan',
}

// aturan jenis, sama dengan backend/services/jenis_barang.go:
// susuk/tempat/kotak tanpa gram (berat-kadar opsional, tidak cuci/lebur/gadai); tempat/kotak boleh gratis
export const tanpaGram = (jenis) => jenis === 'susuk' || jenis === 'tempat_perhiasan' || jenis === 'kotak_perhiasan'
export const bolehGratis = (jenis) => jenis === 'tempat_perhiasan' || jenis === 'kotak_perhiasan'
export const bolehBaki = (jenis) => jenis === 'perhiasan' || jenis === 'logam_mulia' || jenis === 'susuk'

// satu baris isian pada form barang masuk (meniru jenis, nama, merek, dan kadar baris sebelumnya)
export const barisBaru = (dari) => ({
  jenis: dari ? dari.jenis : 'perhiasan',
  item_name: dari ? dari.item_name : '',
  brand: dari ? dari.brand : '',
  karat: dari ? dari.karat : '',
  weight_gram: '',
  harga_beli: '',
  note: '',
  jumlah: '1',
})

// Menentukan "mode" dari barang yang dicentang:
// campur, transit (menuju back office), transit_toko (menuju toko), siap/siap_toko (sama, tapi belum diserahkan ke kurir),
// kembali (dari PJ), cuci,
// back_office / cabang (barang tersedia di lokasi itu). tipeLok(id) = tipe lokasi.
export function tentukanMode(terpilih, tipeLok) {
  if (terpilih.length === 0) return null

  const statusSet = new Set(terpilih.map((r) => r.status))
  if (statusSet.size > 1) return 'campur'

  if (statusSet.has('dikirim')) {
    const keToko = terpilih.filter((r) => r.tujuan_id > 0 || r.bp_nama).length
    // disiapkan (belum diserahkan ke kurir): pengirim masih bisa membatalkan
    const siap = terpilih.filter((r) => !r.diserahkan).length
    if (siap > 0 && siap < terpilih.length) return 'campur'
    const akhiran = siap ? 'siap' : 'transit'
    if (keToko === terpilih.length) return `${akhiran}_toko`
    if (keToko === 0) return akhiran
    return 'campur'
  }
  if (statusSet.has('di_bp')) return 'di_bp'
  if (statusSet.has('kembali')) return 'kembali'
  if (statusSet.has('cuci')) return 'cuci'

  const tipeTerpilih = new Set(terpilih.map((r) => tipeLok(r.location_id)))
  return tipeTerpilih.size > 1 ? 'campur' : [...tipeTerpilih][0]
}

// Tombol aksi yang tersedia untuk mode tertentu: [[kode aksi, label]].
// transit_toko hanya untuk akun cabang tujuan (owner hanya memantau progres).
// bisaGadai: satu atau beberapa barang tersedia dari lokasi yang sama dipilih oleh akun yang boleh menggadaikan.
// adaTanpaGram: ada susuk/tempat/kotak yang dipilih (tidak bisa dicuci atau dilebur).
export function aksiUntukMode(mode, { isCabang, adaLM, adaTanpaGram, bisaGadai }) {
  // disiapkan: pengirim (toko untuk kiriman ke back office, back office untuk oper ke toko) serah ke kurir atau batal
  if (mode === 'siap' || (mode === 'siap_toko' && !isCabang)) {
    return [['serah', 'Sudah diserahkan ke kurir'], ['batal_kirim', 'Batal kirim']]
  }
  if (mode === 'di_bp' && !isCabang) return [['tarik_bp', 'Tarik kembali dari BP']]
  if (mode === 'transit' && !isCabang) return [['sampai', 'Sudah sampai'], ['belum', 'Belum sampai']]
  if (mode === 'transit_toko' && isCabang) {
    return [['terima_toko', 'Sudah diterima'], ['belum_toko', 'Belum diterima']]
  }
  if (mode === 'kembali' && !isCabang) {
    return [['terima_kembali', 'Sudah diterima'], ['belum_kembali', 'Belum diterima']]
  }
  const gadai = bisaGadai ? [['gadai', 'Digadaikan']] : []
  if (mode === 'back_office' && !isCabang) {
    if (adaTanpaGram) return [['kirim', 'Oper toko']]
    return adaLM
      ? [['kirim', 'Oper toko'], ...gadai]
      : [['kirim', 'Oper toko'], ['cuci', 'Cuci'], ['lebur', 'Lebur']]
  }
  if (mode === 'cabang') return [['antar', 'Antar ke back office'], ['jual', 'Jual'], ...gadai]
  return []
}
