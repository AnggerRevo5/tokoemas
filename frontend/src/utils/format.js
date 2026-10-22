export const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)

export const gram = (n) => Number(Number(n).toFixed(3))

export const hariIni = () => new Date().toLocaleDateString('en-CA')

export const awalBulan = () => {
  const d = new Date()
  return new Date(d.getFullYear(), d.getMonth(), 1).toLocaleDateString('en-CA')
}

// ambil bagian tanggal (YYYY-MM-DD) dari string ISO
export const tgl10 = (d) => (d ? d.slice(0, 10) : '')

// pilih/lepas satu id pada objek { [id]: true }
export const toggleId = (pilih, id) => {
  const n = { ...pilih }
  if (n[id]) delete n[id]
  else n[id] = true
  return n
}
