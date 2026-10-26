import { useState } from 'react'

// Angka di menu (barang yang perlu ditindaklanjuti). Berdenyut lembut sekali hanya saat jumlahnya bertambah.
export default function BadgeAngka({ n }) {
  const jumlah = n || 0
  const [lalu, setLalu] = useState(jumlah)
  const [denyut, setDenyut] = useState(0)

  if (jumlah !== lalu) {
    setLalu(jumlah)
    if (jumlah > lalu) setDenyut(denyut + 1)
  }

  if (jumlah <= 0) return null
  return (
    <span key={denyut} className={denyut > 0 ? 'badge denyut' : 'badge'}>
      {jumlah}
    </span>
  )
}
