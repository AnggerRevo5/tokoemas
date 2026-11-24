export const NAMA_ROLE = {
  super_admin: 'Super Admin',
  owner: 'Owner',
  cabang: 'Akun Cabang',
  pj: 'Penanggung Jawab (PJ)',
}

// urutan role di halaman Akun
export const URUTAN_ROLE = ['super_admin', 'owner', 'cabang', 'pj']

// ringkasan kewenangan tiap role (semua akun dengan role yang sama punya fitur yang sama)
export const DESKRIPSI_ROLE = {
  super_admin:
    'Akses penuh ke semua menu, bisa membatalkan transaksi, serta mengelola akun, password, dan cabang.',
  owner:
    'Semua menu operasional dan laporan. Mengoper barang ke cabang dan mengonfirmasi barang yang masuk ke back office; edit atau pembatalan data kas/stok lama hanya oleh Super Admin.',
  cabang:
    'Operasional satu toko (cabangnya sendiri): input kas dan stok, cuci, dan service. Edit atau pembatalan data kas/stok yang sudah tercatat hanya oleh Super Admin.',
  pj: 'Penanggung jawab: mencatat proses lebur, cuci, dan service di back office, serta mengurus talangan (kirim ke BO, penukaran dengan BP, terima dan setor uang).',
}

// kelompok menu di sidebar (urutan tampil)
const GRUP_MENU = [
  { kode: 'dashboard', judul: 'Dashboard', menu: ['dashboard'] },
  { kode: 'transaksi', judul: 'Transaksi', menu: ['kas', 'stok', 'riwayat_stok', 'talangan'] },
  { kode: 'produksi', judul: 'Produksi', menu: ['cuci', 'lebur', 'service'] },
  { kode: 'keuangan', judul: 'Keuangan', menu: ['hutang', 'investor', 'labarugi', 'laporan'] },
  { kode: 'pengaturan', judul: 'Pengaturan', menu: ['akun'] },
]

// menu tab sesuai role: [kode, label]
export function menuUntuk(role) {
  if (role === 'pj') return [['lebur', 'Lebur'], ['cuci', 'Cuci'], ['service', 'Service'], ['talangan', 'Talangan']]

  const menu = [
    ['dashboard', 'Dashboard'],
    ['kas', 'Kas'],
    ['stok', 'Stok'],
    ['riwayat_stok', 'Riwayat Stok'],
    ['talangan', 'Talangan'],
    ['cuci', 'Cuci'],
    ['service', 'Service'],
  ]
  if (role !== 'cabang') {
    menu.push(['lebur', 'Lebur'], ['laporan', 'Laporan'], ['hutang', 'Hutang'], ['investor', 'Investor'], ['labarugi', 'Laba Rugi'])
  }
  if (role === 'super_admin') {
    menu.push(['akun', 'Akun'])
  }
  return menu
}

// menu sesuai role yang sudah dikelompokkan untuk sidebar: [{ kode, judul, items: [[kode, label]] }]
export function grupMenuUntuk(role) {
  const label = Object.fromEntries(menuUntuk(role))
  return GRUP_MENU.map((g) => ({
    kode: g.kode,
    judul: g.judul,
    items: g.menu.filter((k) => label[k]).map((k) => [k, label[k]]),
  })).filter((g) => g.items.length > 0)
}
