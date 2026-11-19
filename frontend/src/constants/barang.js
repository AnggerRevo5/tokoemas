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
  cuci_antrean: 'Antrean cuci (menunggu PJ)',
  lebur: 'Dilebur',
  lebur_antrean: 'Antrean lebur (menunggu PJ)',
  hasil: 'Hasil lebur',
  kembali: 'Dikembalikan ke back office',
  terjual: 'Terjual',
  digadai: 'Digadai',
  tidak_ditebus: 'Tidak ditebus',
  di_bp: 'Di BP',
}

// pihak luar tanpa akun, tujuan oper barang dan mitra tukar (sama dengan services.DaftarBP di backend)
export const BP_LIST = ['BP Induk', 'BP Pecinan', 'BP Iss', 'BP Lawang']

export const STATUS_CLASS = {
  tersedia: 'tag in',
  dikirim: 'tag warn',
  cuci: 'tag info',
  cuci_antrean: 'tag warn',
  lebur: 'tag info',
  lebur_antrean: 'tag warn',
  hasil: 'tag gray',
  kembali: 'tag warn',
  terjual: 'tag gray',
  digadai: 'tag warn',
  tidak_ditebus: 'tag gray',
  di_bp: 'tag info',
}

// penjelasan singkat di bawah form aksi barang
export const KETERANGAN_AKSI = {
  antar: 'Barang berstatus Disiapkan dan masih bisa dibatalkan. Setelah diserahkan ke kurir, barang dalam perjalanan sampai back office menekan Sudah sampai.',
  tarik_bp: 'Barang kembali tersedia di stok back office dan stok bertambah lagi.',
  serah: 'Barang berstatus dalam perjalanan. Kiriman tidak bisa dibatalkan lagi, penerima yang mengonfirmasi.',
  batal_kirim: 'Barang kembali tersedia di stok lokasi asal. Stok tidak berubah.',
  sampai: 'Barang masuk ke stok back office.',
  belum: 'Barang tetap di daftar dengan label belum sampai.',
  terima_toko: 'Stok back office berkurang dan stok toko ini bertambah.',
  belum_toko: 'Barang tetap tercatat di back office dengan label belum diterima.',
  terima_kembali:
    'Barang masuk ke stok back office dengan berat terbaru yang dicatat PJ. Barang hasil cuci berstatus Bersih, lalu bisa dioper ke toko cabang.',
  belum_kembali: 'Barang tetap tercatat dengan label belum diterima.',
  kirim:
    'Barang berstatus Disiapkan dan masih bisa dibatalkan. Setelah diserahkan ke kurir, barang tercatat di back office sampai akun toko tujuan menekan Sudah diterima. Untuk BP (tanpa akun), stok back office langsung berkurang saat diserahkan ke kurir.',
  lebur:
    'Barang masuk antrean PJ untuk dilebur dan tidak bisa dipakai untuk transaksi lain. PJ memilih barang mana yang digabung menjadi satu batch. Stok berkurang saat PJ membentuk batch.',
  cuci:
    'Barang masuk antrean PJ untuk dicuci dan tidak bisa dipakai untuk transaksi lain. PJ memilih barang mana yang digabung menjadi satu pekerjaan cuci. Stok berkurang saat PJ membentuk pekerjaan, lalu PJ mengirimnya ke pencuci dan mencatat saat selesai. Barang yang selesai berstatus Bersih.',
  gadai:
    'Dana yang diterima masuk ke kas toko ini dan barang berstatus Digadai sampai lunas. Cicilan tiap bulan dikonfirmasi dari card barang (klik nama barangnya).',
  jual: 'Isi harga jual tiap barang. Total harga jual otomatis masuk ke kas toko ini (menu Kas), tidak perlu diinput lagi.',
}
