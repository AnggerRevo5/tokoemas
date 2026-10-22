import { useState } from 'react'
import { api } from '../../services/api'
import { hariIni } from '../../utils/format'
import { JENIS_KREDITUR } from '../../constants/hutang'

// Form terima dana baru atau catat hutang tanpa uang masuk.
// onMulai: bersihkan pesan; onSukses(res): setelah tersimpan; onError(msg).
export default function FormHutang({ onMulai, onSukses, onError }) {
  const [loading, setLoading] = useState(false)
  const [mode, setMode] = useState('dana')
  const [jenis, setJenis] = useState('investor')
  const [nama, setNama] = useState('')
  const [pokok, setPokok] = useState('')
  const [mulai, setMulai] = useState(hariIni())
  const [tempo, setTempo] = useState('')
  const [note, setNote] = useState('')

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    const body = {
      creditor_type: jenis,
      creditor_name: nama,
      principal: Number(pokok),
      start_date: mulai,
      due_date: tempo,
      note,
    }
    try {
      const res = await api(mode === 'dana' ? '/dana/terima' : '/hutang', {
        method: 'POST',
        body: JSON.stringify(body),
      })
      setNama('')
      setPokok('')
      setTempo('')
      setNote('')
      onSukses(res)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  function ubahMode(m) {
    setMode(m)
    setJenis(m === 'dana' ? 'investor' : 'pabrik')
  }

  return (
    <form className="card" onSubmit={simpan}>
      <h3>{mode === 'dana' ? 'Terima Dana Baru' : 'Catat Hutang Tanpa Uang Masuk'}</h3>
      <p className="muted">
        {mode === 'dana'
          ? 'Sekali simpan: kas back office bertambah dan hutang tercatat atas nama pemberi dana.'
          : 'Untuk hutang yang tidak membawa uang tunai, misalnya barang dari pabrik yang belum dibayar. Kas tidak berubah.'}
      </p>

      <div className="form-grid">
        <div>
          <label>Jenis pencatatan</label>
          <select value={mode} onChange={(e) => ubahMode(e.target.value)}>
            <option value="dana">Dana masuk (kas bertambah)</option>
            <option value="tanpa">Hutang tanpa uang masuk</option>
          </select>
        </div>
        <div>
          <label>Pemberi dana / kreditur</label>
          <select value={jenis} onChange={(e) => setJenis(e.target.value)}>
            {JENIS_KREDITUR.map(([k, l]) => (
              <option key={k} value={k}>{l}</option>
            ))}
          </select>
        </div>
        <div>
          <label>Nama</label>
          <input value={nama} onChange={(e) => setNama(e.target.value)} required />
        </div>
        <div>
          <label>Nominal (Rp)</label>
          <input type="number" min="1" value={pokok} onChange={(e) => setPokok(e.target.value)} required />
        </div>
        <div>
          <label>Tanggal</label>
          <input type="date" value={mulai} onChange={(e) => setMulai(e.target.value)} required />
        </div>
        <div>
          <label>Jatuh tempo (opsional)</label>
          <input type="date" value={tempo} onChange={(e) => setTempo(e.target.value)} />
        </div>
        <div className="span2">
          <label>Catatan</label>
          <input value={note} onChange={(e) => setNote(e.target.value)} />
        </div>
      </div>

      <button disabled={loading}>
        {loading ? 'Menyimpan...' : mode === 'dana' ? 'Simpan: kas masuk dan hutang tercatat' : 'Simpan hutang'}
      </button>
    </form>
  )
}
