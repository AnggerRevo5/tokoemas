// Potongan QRIS/Debit bisa diisi dalam rupiah atau persen; yang satunya dihitung otomatis dari total penjualan.
// State: { mode: 'nominal' | 'persen', nilai: string } (mode = kolom yang terakhir diisi pengguna).
export const BIAYA_KOSONG = { mode: 'nominal', nilai: '' }

const bulat2 = (x) => Math.round(x * 100) / 100

export function nominalBiaya(total, b) {
  const v = Number(b.nilai) || 0
  return b.mode === 'persen' ? Math.round(((Number(total) || 0) * v) / 100) : v
}

// teks untuk kolom persen: apa adanya bila sedang diisi, selain itu dihitung dari nominal
export function persenBiayaTeks(total, b) {
  if (b.mode === 'persen') return b.nilai
  const t = Number(total) || 0
  const v = Number(b.nilai) || 0
  return t > 0 && v > 0 ? String(bulat2((v / t) * 100)) : ''
}

// teks untuk kolom nominal: apa adanya bila sedang diisi, selain itu dihitung dari persen
export function nominalBiayaTeks(total, b) {
  if (b.mode === 'nominal') return b.nilai
  const n = nominalBiaya(total, b)
  return n > 0 ? String(n) : ''
}
