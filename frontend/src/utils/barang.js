// barang yang boleh dicentang di daftar stok
export const bisaPilih = (r) =>
  r.status === 'tersedia' || r.status === 'dikirim' || r.status === 'cuci' || r.status === 'kembali'

// satu baris isian pada form barang masuk (meniru jenis, nama, merek, dan kadar baris sebelumnya)
export const barisBaru = (dari) => ({
  jenis: dari ? dari.jenis : 'perhiasan',
  item_name: dari ? dari.item_name : '',
  brand: dari ? dari.brand : '',
  karat: dari ? dari.karat : '',
  weight_gram: '',
  harga_beli: '',
  note: '',
})

// Menentukan "mode" dari barang yang dicentang:
// campur, transit (menuju back office), transit_toko (menuju toko), kembali (dari PJ), cuci,
// back_office / cabang (barang tersedia di lokasi itu). tipeLok(id) = tipe lokasi.
export function tentukanMode(terpilih, tipeLok) {
  if (terpilih.length === 0) return null

  const statusSet = new Set(terpilih.map((r) => r.status))
  if (statusSet.size > 1) return 'campur'

  if (statusSet.has('dikirim')) {
    const keToko = terpilih.filter((r) => r.tujuan_id > 0).length
    if (keToko === terpilih.length) return 'transit_toko'
    if (keToko === 0) return 'transit'
    return 'campur'
  }
  if (statusSet.has('kembali')) return 'kembali'
  if (statusSet.has('cuci')) return 'cuci'

  const tipeTerpilih = new Set(terpilih.map((r) => tipeLok(r.location_id)))
  return tipeTerpilih.size > 1 ? 'campur' : [...tipeTerpilih][0]
}

// Tombol aksi yang tersedia untuk mode tertentu: [[kode aksi, label]].
// transit_toko hanya untuk akun cabang tujuan (owner hanya memantau progres).
export function aksiUntukMode(mode, { isCabang, adaLM }) {
  if (mode === 'transit' && !isCabang) return [['sampai', 'Sudah sampai'], ['belum', 'Belum sampai']]
  if (mode === 'transit_toko' && isCabang) {
    return [['terima_toko', 'Sudah diterima'], ['belum_toko', 'Belum diterima']]
  }
  if (mode === 'kembali' && !isCabang) {
    return [['terima_kembali', 'Sudah diterima'], ['belum_kembali', 'Belum diterima']]
  }
  if (mode === 'back_office' && !isCabang) {
    return adaLM
      ? [['kirim', 'Jual (oper ke toko)']]
      : [['kirim', 'Jual (oper ke toko)'], ['cuci', 'Cuci'], ['lebur', 'Lebur']]
  }
  if (mode === 'cabang') return [['antar', 'Antar ke back office'], ['jual', 'Jual']]
  return []
}
