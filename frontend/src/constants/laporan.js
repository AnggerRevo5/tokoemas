// [kode, nama, kelompok] jenis laporan yang bisa dipilih
export const LAPORAN = [
  ['kas', 'Kas (uang masuk dan keluar)', 'Keuangan'],
  ['labarugi', 'Laba rugi', 'Keuangan'],
  ['hutang', 'Hutang (posisi saat ini)', 'Keuangan'],
  ['penjualan', 'Penjualan ke penadah (untung atau rugi)', 'Keuangan'],
  ['penjualan_toko_ringkas', 'Penjualan toko: ringkasan per lokasi dan metode', 'Keuangan'],
  ['penjualan_toko', 'Penjualan toko: rincian per barang', 'Keuangan'],
  ['gadai', 'Gadai: ringkasan dan jadwal cicilan', 'Keuangan'],
  ['investor', 'Investor: ringkasan per periode', 'Keuangan'],
  ['investor_mutasi', 'Investor: mutasi setor, tarik, dan bunga', 'Keuangan'],
  ['cuci', 'Cuci', 'Operasional'],
  ['lebur', 'Lebur', 'Operasional'],
  ['stok', 'Riwayat stok', 'Operasional'],
]

export const KELOMPOK_LAPORAN = ['Keuangan', 'Operasional']
