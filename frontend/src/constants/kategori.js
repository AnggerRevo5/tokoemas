export const KATEGORI = {
  kas: {
    cabang: {
      in: [
        ['omzet', 'Hasil jual (omzet)'],
        ['tambah_kas', 'Tambahan kas'],
      ],
      out: [
        ['pengeluaran_toko', 'Pengeluaran toko'],
        ['transfer_bo', 'Transfer kas ke back office'],
      ],
    },
    back_office: {
      in: [
        ['jual_barang', 'Pemasukan dari jual barang'],
        ['investor_baru', 'Investor baru'],
        ['setor_investor', 'Setoran / top-up investor'],
        ['terima_pinjaman', 'Terima pinjaman'],
        ['jual_aset', 'Penjualan aset'],
        ['kembali_cabang', 'Barang kembali cabang'],
      ],
      out: [
        ['kas_toko', 'Kas toko'],
        ['bayar_hutang', 'Bayar hutang'],
        ['tarik_pokok_investor', 'Tarik pokok investor'],
        ['bayar_bunga_investor', 'Bayar bunga investor'],
        ['ambil_investor', 'Ambil pokok dan/atau bunga investor'],
        ['operasional', 'Pengeluaran operasional'],
        ['kembali_cabang', 'Barang kembali cabang'],
      ],
      nonkas: [
        ['catat_hutang', 'Catat hutang tanpa uang masuk'],
        ['batalkan_hutang', 'Batalkan catatan hutang'],
      ],
    },
  },
  stok: {
    cabang: {
      in: [
        ['beli_customer', 'Beli dari customer (barang kotor)'],
        ['stok_awal', 'Stok awal toko'],
      ],
      out: [
        ['laku', 'Laku'],
        ['pegadaian', 'Pegadaian'],
        ['cuci', 'Cuci (maintenance)'],
        ['lebur', 'Lebur'],
        ['jual', 'Jual'],
        ['kembali_bo', 'Kembalikan ke Back Office'],
      ],
    },
    back_office: {
      in: [
        ['pabrik', 'Pabrik'],
        ['konsinyasi', 'Konsinyasi'],
        ['stok_awal', 'Stok awal back office'],
      ],
      out: [
        ['ke_toko', 'Ke toko cabang'],
        ['jual', 'Jual'],
      ],
    },
  },
}

const SEMUA = {
  biaya_service: 'Biaya service',
  ke_bp: 'Dikirim ke BP',
  kembali_bp: 'Ditarik kembali dari BP',
  dari_toko: 'Pemasukan dari toko',
  bayar_hutang: 'Pembayaran hutang',
  kembali_cabang: 'Barang kembali cabang',
  pembelian_customer: 'Pembelian dari customer',
  talangan: 'Barang talangan masuk BO',
  talangan_keluar: 'Barang kita dari BP',
  talangan_tukar: 'Barang talangan ditukar ke BP',
  talangan_tukar_masuk: 'Tukar talangan (BP membayar ke kita)',
  talangan_tukar_keluar: 'Tukar talangan (kita membayar ke BP)',
  barang_cabang: 'Barang cabang BP (talangan)',
  tukar_tambah_masuk: 'Tukar tambah (pelanggan membayar selisih)',
  tukar_tambah_keluar: 'Tukar tambah (toko membayar selisih)',
  tukar_tambah: 'Tukar tambah',
  barang_masuk: 'Barang masuk',
  cucian: 'Barang cucian',
  barang_toko: 'Barang toko',
  investor: 'Setoran investor',
  investor_baru: 'Investor baru',
  setor_investor: 'Setoran / top-up investor',
  terima_pinjaman: 'Terima pinjaman',
  catat_hutang: 'Hutang tanpa uang masuk',
  bayar_bunga_investor: 'Pembayaran bunga investor',
  ambil_investor: 'Pengambilan pokok dan/atau bunga investor',
  bulanan_investor: 'Pembayaran bunga investor',
  pinjaman: 'Penerimaan pinjaman',
  jual_qris: 'Penjualan QRIS/Debit',
  biaya_qris: 'Potongan QRIS/Debit',
  tarik_investor: 'Penarikan pokok investor',
  tarik_pokok_investor: 'Penarikan pokok investor',
  gadai_dana: 'Dana gadai diterima',
  gadai_pokok: 'Pembayaran pokok gadai',
  gadai_biaya: 'Biaya gadai (bunga dan denda)',
  gadai_kembali: 'Kembali dari gadai',
}
Object.values(KATEGORI).forEach((j) =>
  Object.values(j).forEach((t) =>
    Object.values(t).forEach((arr) => arr.forEach(([k, l]) => (SEMUA[k] = l)))
  )
)

export const labelKategori = (k) => SEMUA[k] || k