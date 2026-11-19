// Setoran investor punya kategori terpisah di form Kas.
export const JENIS_KREDITUR = [
  ['bank_pegadaian', 'Bank / Pegadaian'],
  ['perorangan', 'Perorangan'],
  ['pabrik', 'Pabrik'],
]

// hutang investor lama (sebelum ada menu Investor) tetap tampil dengan labelnya
export const labelJenis = { investor: 'Investor', ...Object.fromEntries(JENIS_KREDITUR) }
