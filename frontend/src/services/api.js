// Sesi login disimpan backend di cookie HttpOnly (tidak bisa dibaca JavaScript).
// Cookie te_csrf boleh dibaca: nilainya dikirim ulang di header X-CSRF-Token untuk request yang mengubah data.
const COOKIE_CSRF = 'te_csrf'
const TOKEN_LAMA = 'token'

// hapus token lama yang dulu disimpan di localStorage (sebelum pindah ke cookie)
try {
  localStorage.removeItem(TOKEN_LAMA)
} catch {
  // abaikan bila penyimpanan tidak tersedia
}

const bacaCookie = (nama) => {
  const awal = `${nama}=`
  const c = document.cookie.split('; ').find((x) => x.startsWith(awal))
  return c ? decodeURIComponent(c.slice(awal.length)) : ''
}

// true bila browser masih memegang sesi login (dipakai untuk menentukan tampilan awal)
export const adaSesi = () => bacaCookie(COOKIE_CSRF) !== ''

// hapus penanda sesi di browser (cookie sesi HttpOnly dihapus oleh backend)
export const hapusSesiLokal = () => {
  document.cookie = `${COOKIE_CSRF}=; Max-Age=0; Path=/; SameSite=Strict`
}

export async function api(path, options = {}) {
  const headers = { 'Content-Type': 'application/json' }
  const method = (options.method || 'GET').toUpperCase()
  if (method !== 'GET' && method !== 'HEAD') {
    headers['X-CSRF-Token'] = bacaCookie(COOKIE_CSRF)
  }

  const res = await fetch(`/api${path}`, { ...options, headers, credentials: 'same-origin' })
  const data = await res.json().catch(() => ({}))

  if (res.status === 401 && path !== '/login') {
    hapusSesiLokal()
    window.location.reload()
  }
  if (!res.ok) throw new Error(data.error || 'Terjadi kesalahan')
  return data
}

export async function unduh(path, namaFile) {
  const res = await fetch(`/api${path}`, { credentials: 'same-origin' })
  if (!res.ok) {
    const d = await res.json().catch(() => ({}))
    throw new Error(d.error || 'Gagal mengunduh')
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = namaFile
  a.click()
  URL.revokeObjectURL(url)
}
