import { useState, useEffect } from 'react'
import { api, unduh } from '../api'

const awalBulan = () => {
  const d = new Date()
  return new Date(d.getFullYear(), d.getMonth(), 1).toLocaleDateString('en-CA')
}
const hariIni = () => new Date().toLocaleDateString('en-CA')

export default function ExportExcel({ me }) {
  const [dari, setDari] = useState(awalBulan())
  const [sampai, setSampai] = useState(hariIni())
  const [locId, setLocId] = useState('0')
  const [locations, setLocations] = useState([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api('/locations').then(setLocations).catch((e) => setError(e.message))
  }, [])

  async function klik() {
    setError('')
    setLoading(true)
    try {
      await unduh(
        `/export?from=${dari}&to=${sampai}&location_id=${locId}`,
        `pembukuan_${dari}_sd_${sampai}.xlsx`
      )
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card">
      <h3>Export ke Excel</h3>
      <p className="muted">
        File berisi dua sheet: Kas dan Stok. Transaksi yang dibatalkan tidak ikut.
      </p>

      <div className="form-grid">
        <div>
          <label>Dari tanggal</label>
          <input type="date" value={dari} onChange={(e) => setDari(e.target.value)} />
        </div>
        <div>
          <label>Sampai tanggal</label>
          <input type="date" value={sampai} onChange={(e) => setSampai(e.target.value)} />
        </div>
        {me.role !== 'cabang' && (
          <div>
            <label>Lokasi</label>
            <select value={locId} onChange={(e) => setLocId(e.target.value)}>
              <option value="0">Semua lokasi</option>
              {locations.map((l) => (
                <option key={l.id} value={l.id}>{l.name}</option>
              ))}
            </select>
          </div>
        )}
      </div>

      {error && <div className="error">{error}</div>}

      <button onClick={klik} disabled={loading}>
        {loading ? 'Menyiapkan file...' : 'Unduh Excel'}
      </button>
    </div>
  )
}
