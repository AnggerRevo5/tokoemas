// Status talangan (sama dengan backend services/talangan_status.go).
export const STATUS_MASUK = ['TALANGAN', 'KIRIM_BO', 'DI_BO', 'DIKIRIM_KE_BP', 'DIHITUNG', 'LUNAS']
export const STATUS_KELUAR = ['TERCATAT', 'DI_BO', 'DIHITUNG', 'LUNAS']

export const LABEL_STATUS = {
  TALANGAN: 'Talangan di toko',
  KIRIM_BO: 'Dibawa ke BO',
  DI_BO: 'Di back office',
  DIKIRIM_KE_BP: 'Dikirim ke BP',
  TERCATAT: 'Tercatat, menunggu owner',
  DIHITUNG: 'Sudah dihitung',
  LUNAS: 'Lunas',
  BELUM_SAMPAI: 'Ditandai belum sampai', // hanya muncul di riwayat
}

export const KELAS_STATUS = {
  TALANGAN: 'tag warn',
  KIRIM_BO: 'tag info',
  DI_BO: 'tag info',
  DIKIRIM_KE_BP: 'tag info',
  TERCATAT: 'tag warn',
  DIHITUNG: 'tag warn',
  LUNAS: 'tag in',
}

export const LABEL_UANG = {
  BELUM_BAYAR: 'Belum dibayar',
  DI_TANGAN_PJ: 'Uang di tangan PJ',
  MASUK_KAS_BO: 'Sudah masuk kas BO',
}

export const KELAS_UANG = {
  BELUM_BAYAR: 'tag warn',
  DI_TANGAN_PJ: 'tag info',
  MASUK_KAS_BO: 'tag in',
}

export const JENIS_TALANGAN = [
  ['perhiasan', 'Perhiasan'],
  ['logam_mulia', 'Logam mulia'],
]

export const BAHAN_TALANGAN = [
  ['emas', 'Emas'],
  ['perak', 'Perak'],
]

// arah uang dari saldo penukaran (saldo = total masuk - total keluar): positif BP bayar ke kita
export function teksSaldo(saldo, rupiah) {
  if (saldo > 0) return `BP membayar ke kita ${rupiah(saldo)}`
  if (saldo < 0) return `Kita membayar ke BP ${rupiah(-saldo)}`
  return 'Impas, tidak ada uang pindah'
}

// aksi yang boleh dilakukan role pada satu baris talangan (backend tetap memeriksa ulang)
export function aksiTalangan(row, role) {
  const pj = role === 'pj' || role === 'super_admin'
  const owner = role === 'owner' || role === 'super_admin'
  const aksi = []
  if (row.arah === 'MASUK' && row.status === 'TALANGAN' && pj) aksi.push('kirim-bo')
  // owner memeriksa barang di BO: MASUK yang dibawa PJ (KIRIM_BO) dan barang kembali KELUAR (TERCATAT)
  const menungguOwner = (row.arah === 'MASUK' && row.status === 'KIRIM_BO') || (row.arah === 'KELUAR' && row.status === 'TERCATAT')
  if (menungguOwner && owner) aksi.push('terima-bo', 'belum-sampai')
  if (owner && !row.tukar_id && row.status !== 'LUNAS') aksi.push('koreksi-harga')
  return aksi
}
