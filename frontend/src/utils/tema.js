const KUNCI = 'tema'

// Tema awal: pilihan yang tersimpan, atau mengikuti pengaturan perangkat.
export function temaAwal() {
  try {
    const t = localStorage.getItem(KUNCI)
    if (t === 'terang' || t === 'gelap') return t
  } catch {
    // penyimpanan browser tidak tersedia, pakai tema perangkat
  }
  const gelap = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches
  return gelap ? 'gelap' : 'terang'
}

export function terapkanTema(tema) {
  document.documentElement.dataset.tema = tema
}

export function simpanTema(tema) {
  try {
    localStorage.setItem(KUNCI, tema)
  } catch {
    // abaikan bila penyimpanan tidak tersedia
  }
  terapkanTema(tema)
}
