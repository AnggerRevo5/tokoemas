// status barang service: [label, kelas tag]
export const STATUS_SERVICE = {
  di_toko: ['Di toko', 'tag in'],
  dikirim_bo: ['Dalam perjalanan ke back office', 'tag warn'],
  di_bo: ['Di back office', 'tag in'],
  diproses: ['Sedang diservis', 'tag warn'],
  kembali_bo: ['Dikembalikan ke back office', 'tag warn'],
  selesai: ['Selesai servis', 'tag in'],
  dikirim_toko: ['Dalam perjalanan ke toko', 'tag warn'],
  siap_diambil: ['Siap diambil', 'tag in'],
  diambil: ['Sudah diambil', 'tag gray'],
}

// langkah yang boleh dilakukan role tertentu pada status tertentu: [[kode aksi, label]]
export const aksiUntuk = (status, role) => {
  const admin = role === 'owner' || role === 'super_admin'
  const pj = role === 'pj' || role === 'super_admin'
  const toko = role === 'cabang' || admin
  switch (status) {
    case 'di_toko':
      return toko ? [['kirim_bo', 'Antar ke back office']] : []
    case 'dikirim_bo':
      return admin ? [['terima_bo', 'Sudah sampai'], ['belum_sampai', 'Belum sampai']] : []
    case 'di_bo':
      return pj ? [['proses', 'Kirim ke penyervice']] : []
    case 'diproses':
      return pj ? [['selesai', 'Selesai servis']] : []
    case 'kembali_bo':
      return admin ? [['terima_kembali', 'Sudah diterima'], ['belum_kembali', 'Belum diterima']] : []
    case 'selesai':
      return admin ? [['oper', 'Oper ke cabang']] : []
    case 'dikirim_toko':
      // konfirmasi hanya oleh cabang tujuan; owner hanya memantau
      return role === 'cabang' ? [['terima_toko', 'Sudah diterima di toko']] : []
    case 'siap_diambil':
      return toko ? [['diambil', 'Diambil customer']] : []
    default:
      return []
  }
}
