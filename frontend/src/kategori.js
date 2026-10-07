export const KATEGORI = {
  kas: {
    cabang: {
      in: [
        ['omzet', 'Hasil jual (omzet)'],
        ['tambah_kas', 'Tambahan kas'],
      ],
      out: [
        ['pengeluaran_toko', 'Pengeluaran toko'],
        ['pembelian_customer', 'Pembelian barang customer'],
        ['transfer_bo', 'Transfer kas ke back office'],
      ],
    },
    back_office: {
      in: [
        ['jual_barang', 'Pemasukan dari jual barang'],
        ['investor', 'Pemasukan dari investor'],
        ['pinjaman', 'Pinjaman'],
        ['jual_aset', 'Penjualan aset'],
        ['kembali_cabang', 'Barang kembali cabang'],
      ],
      out: [
        ['kas_toko', 'Kas toko'],
        ['bulanan_investor', 'Bulanan investor'],
        ['operasional', 'Pengeluaran operasional'],
        ['kembali_cabang', 'Barang kembali cabang'],
      ],
    },
  },
  stok: {
    cabang: {
      in: [['barang_masuk', 'Barang masuk']],
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
        ['cucian', 'Barang cucian'],
        ['barang_toko', 'Barang toko'],
      ],
      out: [
        ['ke_toko', 'Ke toko (Kepanjen/Dinoyo)'],
        ['lebur', 'Lebur'],
        ['jual', 'Jual'],
        ['cuci', 'Cuci'],
      ],
    },
  },
}

const SEMUA = {
  dari_toko: 'Pemasukan dari toko',
  bayar_hutang: 'Pembayaran hutang',
  kembali_cabang: 'Barang kembali cabang',
}
Object.values(KATEGORI).forEach((j) =>
  Object.values(j).forEach((t) =>
    Object.values(t).forEach((arr) => arr.forEach(([k, l]) => (SEMUA[k] = l)))
  )
)

export const labelKategori = (k) => SEMUA[k] || k