import { useEffect, useState } from 'react'
import { api } from '../../services/api'
import { rupiah } from '../../utils/format'

// Ringkasan baca-saja dana investor di halaman Hutang. Datanya dikelola di menu Investor (setor, tarik pokok,
// dan bunga otomatis tercatat di kas), jadi tidak diinput lagi di sini. Akun tanpa akses Investor tidak melihatnya.
export default function RingkasInvestor() {
  const [rows, setRows] = useState([])

  useEffect(() => {
    api('/investor')
      .then(setRows)
      .catch(() => setRows([]))
  }, [])

  const aktif = rows.filter((r) => r.pokok > 0)
  if (aktif.length === 0) return null
  const pokok = aktif.reduce((s, r) => s + r.pokok, 0)
  const bunga = aktif.reduce((s, r) => s + r.bunga_belum, 0)

  return (
    <div className="card" style={{ marginBottom: 16 }}>
      <h3>Dana investor</h3>
      <p style={{ margin: 0 }}>
        {aktif.length} investor, pokok yang ditanggung <b>{rupiah(pokok)}</b>, bunga belum dibayar <b>{rupiah(bunga)}</b>.
      </p>
      <p className="muted" style={{ marginBottom: 0 }}>
        Termasuk kewajiban toko dan ikut di laporan Hutang. Setor, tarik pokok, dan bunga dicatat di menu Investor dan
        kasnya ikut otomatis, jadi tidak perlu dimasukkan lagi di sini.
      </p>
    </div>
  )
}
