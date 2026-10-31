import { useState } from 'react'
import { api } from '../../services/api'
import { hariIni } from '../../utils/format'
import Spinner from '../common/Spinner'

// Form investor baru beserta setoran pertamanya. Tanggal setor pertama menjadi tanggal jatuh tempo bunga tiap bulan.
// onMulai: bersihkan pesan; onSukses(res): setelah tersimpan; onError(msg).
export default function FormInvestor({ onMulai, onSukses, onError }) {
  const [loading, setLoading] = useState(false)
  const [nama, setNama] = useState('')
  const [jumlah, setJumlah] = useState('')
  const [tanggal, setTanggal] = useState(hariIni())
  const [persen, setPersen] = useState('1')
  const [note, setNote] = useState('')
  const [catatKas, setCatatKas] = useState(true)

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      const res = await api('/investor', {
        method: 'POST',
        body: JSON.stringify({
          name: nama,
          amount: Number(jumlah),
          start_date: tanggal,
          rate_percent: Number(persen),
          note,
          catat_kas: catatKas,
        }),
      })
      setNama('')
      setJumlah('')
      setNote('')
      onSukses(res)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="card" onSubmit={simpan}>
      <h3>Investor Baru</h3>
      <p className="muted">
        Bunga {persen || 1}% per bulan dari pokok, jatuh tempo tiap bulan di tanggal setor pertama. Pokok dan bunga
        dicatat terpisah; bunga dicairkan sesuai kesepakatan dengan owner.
      </p>

      <div className="form-section">
        <div className="section-title">Data investor</div>
        <div className="form-grid">
          <div>
            <label>Nama investor</label>
            <input value={nama} onChange={(e) => setNama(e.target.value)} required maxLength={100} />
          </div>
          <div>
            <label>Setoran pertama (Rp)</label>
            <input type="number" min="1" step="1" value={jumlah} onChange={(e) => setJumlah(e.target.value)} required />
          </div>
          <div>
            <label>Tanggal setor</label>
            <input type="date" value={tanggal} max={hariIni()} onChange={(e) => setTanggal(e.target.value)} required />
          </div>
          <div>
            <label>Bunga per bulan (%)</label>
            <input type="number" min="0.01" max="100" step="0.01" value={persen} onChange={(e) => setPersen(e.target.value)} />
          </div>
          <div className="span2">
            <label>Catatan</label>
            <input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
          </div>
        </div>
      </div>

      <label className="cek-item" style={{ margin: '10px 0' }}>
        <input type="checkbox" checked={catatKas} onChange={(e) => setCatatKas(e.target.checked)} />
        Catat setoran ini juga sebagai kas masuk back office (cukup diisi di sini, tidak perlu lagi di menu Hutang)
      </label>

      <button disabled={loading}>{loading ? <><Spinner />Menyimpan...</> : 'Simpan investor'}</button>
    </form>
  )
}
