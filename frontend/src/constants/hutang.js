// dana investor dicatat lewat menu Investor, jadi tidak ada di pilihan hutang baru
export const JENIS_KREDITUR = [
  ['bank_pegadaian', 'Bank / Pegadaian'],
  ['perorangan', 'Perorangan'],
  ['pabrik', 'Pabrik'],
]

// hutang investor lama (sebelum ada menu Investor) tetap tampil dengan labelnya
export const labelJenis = { investor: 'Investor', ...Object.fromEntries(JENIS_KREDITUR) }
