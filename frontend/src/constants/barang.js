export const NAMA_UMUM = ['Cincin', 'Kalung', 'Gelang', 'Anting', 'Liontin', 'Giwang']
export const MEREK_LM = ['Antam', 'UBS', 'Galeri24', 'Lotus Archi']

// semua kadar 24K sampai 1K
export const KARAT = Array.from({ length: 24 }, (_, i) => 24 - i)
export const KARAT_UMUM = [24, 22, 21, 20, 18, 17, 16, 9, 8]
export const KARAT_LAIN = KARAT.filter((n) => !KARAT_UMUM.includes(n))

export const STATUS_LABEL = {
  tersedia: 'Tersedia',
  dikirim: 'Dalam perjalanan',
  cuci: 'Proses cuci',
  lebur: 'Dilebur',
  hasil: 'Hasil lebur',
  kembali: 'Dikembalikan ke back office',
  terjual: 'Terjual',
}

export const STATUS_CLASS = {
  tersedia: 'tag in',
  dikirim: 'tag warn',
  cuci: 'tag info',
  lebur: 'tag info',
  hasil: 'tag gray',
  kembali: 'tag warn',
  terjual: 'tag gray',
}

// penjelasan singkat di bawah form aksi barang
export const KETERANGAN_AKSI = {
  antar: 'Barang berstatus dalam perjalanan sampai back office menekan Sudah sampai.',
  sampai: 'Barang masuk ke stok back office.',
  belum: 'Barang tetap di daftar dengan label belum sampai.',
  terima_toko: 'Stok back office berkurang dan stok toko ini bertambah.',
  belum_toko: 'Barang tetap tercatat di back office dengan label belum diterima.',
  terima_kembali:
    'Barang masuk ke stok back office dengan berat terbaru yang dicatat PJ. Barang hasil cuci berstatus Bersih, lalu bisa dioper ke toko cabang.',
  belum_kembali: 'Barang tetap tercatat dengan label belum diterima.',
  kirim:
    'Barang berstatus dalam perjalanan dan masih tercatat di back office sampai akun toko tujuan menekan Sudah diterima.',
  lebur:
    'Stok back office berkurang dan barang masuk proses lebur. PJ lalu mencatat berat setelah dipotong, leburan kotor, dan cukim di menu Lebur. Modal dihitung dari jumlah harga beli.',
  cuci:
    'Stok back office berkurang dan barang masuk antrean cuci. PJ yang mengirimnya ke pencuci dan mencatat saat selesai di menu Cuci. Barang yang selesai berstatus Bersih.',
  jual: 'Isi harga jual tiap barang. Total harga jual otomatis masuk ke kas toko ini (menu Kas), tidak perlu diinput lagi.',
}
