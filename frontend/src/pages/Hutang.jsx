import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import Pesan from '../components/common/Pesan'
import TabelHutang from '../components/hutang/TabelHutang'
import RingkasInvestor from '../components/hutang/RingkasInvestor'

export default function Hutang() {
  const [rows, setRows] = useState([])
  const [error, setError] = useState('')

  const muat = useCallback(() => {
    api('/hutang')
      .then(setRows)
      .catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    muat()
  }, [muat])

  return (
    <div>
      <Pesan error={error} />
      <RingkasInvestor />
      <div className="card">
        <h3>Daftar Hutang</h3>
        <p className="muted">Halaman ini hanya untuk melihat status dan sisa hutang. Pengelolaan dilakukan dari Kas Back Office.</p>
        <TabelHutang rows={rows} />
      </div>
    </div>
  )
}
