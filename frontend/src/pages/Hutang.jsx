import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'

const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)

const hariIni = () => new Date().toLocaleDateString('en-CA')

const JENIS = [
  ['investor', 'Investor'],
  ['perorangan', 'Perorangan'],
  ['pabrik', 'Pabrik'],
  ['bank_pegadaian', 'Bank / Pegadaian'],
]
const labelJenis = Object.fromEntries(JENIS)

export default function Hutang() {
  const [rows, setRows] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(false)

  const [jenis, setJenis] = useState('investor')
  const [nama, setNama] = useState('')
  const [pokok, setPokok] = useState('')
  const [mulai, setMulai] = useState(hariIni())
  const [tempo, setTempo] = useState('')
  const [note, setNote] = useState('')

  const [payId, setPayId] = useState(null)
  const [payAmount, setPayAmount] = useState('')
  const [payNote, setPayNote] = useState('')

  const muat = useCallback(() => {
    api('/hutang')
      .then(setRows)
      .catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    muat()
  }, [muat])

  async function simpan(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setLoading(true)
    try {
      const res = await api('/hutang', {
        method: 'POST',
        body: JSON.stringify({
          creditor_type: jenis,
          creditor_name: nama,
          principal: Number(pokok),
          start_date: mulai,
          due_date: tempo,
          note,
        }),
      })
      setInfo(res.message)
      setNama('')
      setPokok('')
      setTempo('')
      setNote('')
      muat()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  function mulaiBayar(r) {
    setPayId(r.id)
    setPayAmount('')
    setPayNote('')
    setError('')
    setInfo('')
  }

  async function konfirmasiBayar() {
    setError('')
    try {
      const res = await api(`/hutang/${payId}/bayar`, {
        method: 'POST',
        body: JSON.stringify({
          amount: Number(payAmount),
          note: payNote,
          paid_date: hariIni(),
        }),
      })
      setInfo(res.lunas ? 'Pembayaran tersimpan, hutang LUNAS' : `Pembayaran tersimpan, sisa ${rupiah(res.sisa)}`)
      setPayId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  const payRow = rows.find((r) => r.id === payId)

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}

      <form className="card" onSubmit={simpan}>
        <h3>Catat Hutang Baru</h3>
        <p className="muted">
          Kalau uang hutangnya masuk ke kas, catat juga di menu Kas (Back Office, uang masuk
          dari investor atau pinjaman).
        </p>

        <div className="form-grid">
          <div>
            <label>Jenis kreditur</label>
            <select value={jenis} onChange={(e) => setJenis(e.target.value)}>
              {JENIS.map(([k, l]) => (
                <option key={k} value={k}>{l}</option>
              ))}
            </select>
          </div>
          <div>
            <label>Nama kreditur</label>
            <input value={nama} onChange={(e) => setNama(e.target.value)} required />
          </div>
          <div>
            <label>Nominal hutang (Rp)</label>
            <input type="number" min="1" value={pokok} onChange={(e) => setPokok(e.target.value)} required />
          </div>
          <div>
            <label>Tanggal mulai</label>
            <input type="date" value={mulai} onChange={(e) => setMulai(e.target.value)} required />
          </div>
          <div>
            <label>Jatuh tempo (opsional)</label>
            <input type="date" value={tempo} onChange={(e) => setTempo(e.target.value)} />
          </div>
          <div>
            <label>Catatan</label>
            <input value={note} onChange={(e) => setNote(e.target.value)} />
          </div>
        </div>

        <button disabled={loading}>{loading ? 'Menyimpan...' : 'Simpan Hutang'}</button>
      </form>

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Daftar Hutang</h3>

        {payRow && (
          <div className="void-box">
            <div>
              Bayar hutang <b>{payRow.creditor_name}</b> (sisa {rupiah(payRow.sisa)}). Uang akan
              otomatis keluar dari kas Back Office.
            </div>
            <input
              type="number"
              min="1"
              value={payAmount}
              onChange={(e) => setPayAmount(e.target.value)}
              placeholder="Nominal pembayaran (Rp)"
              autoFocus
            />
            <input
              style={{ marginTop: 8 }}
              value={payNote}
              onChange={(e) => setPayNote(e.target.value)}
              placeholder="Catatan (opsional), contoh: cicilan 2"
            />
            <div className="void-actions">
              <button type="button" className="mini primary-fill" onClick={konfirmasiBayar}>
                Bayar
              </button>
              <button type="button" className="mini" onClick={() => setPayId(null)}>
                Tidak jadi
              </button>
            </div>
          </div>
        )}

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Kreditur</th>
                <th>Total</th>
                <th>Terbayar</th>
                <th>Sisa</th>
                <th>Jatuh tempo</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && (
                <tr><td colSpan="7" className="muted">Belum ada hutang</td></tr>
              )}
              {rows.map((r) => (
                <tr key={r.id}>
                  <td>
                    {r.creditor_name}
                    <div className="muted">{labelJenis[r.creditor_type] || r.creditor_type}</div>
                  </td>
                  <td>{rupiah(r.principal)}</td>
                  <td>{rupiah(r.terbayar)}</td>
                  <td><b>{rupiah(r.sisa)}</b></td>
                  <td>{r.due_date ? r.due_date.slice(0, 10) : '-'}</td>
                  <td>
                    <span className={r.status === 'lunas' ? 'tag in' : 'tag warn'}>
                      {r.status === 'lunas' ? 'Lunas' : 'Aktif'}
                    </span>
                  </td>
                  <td>
                    {r.status === 'aktif' && (
                      <button type="button" className="mini primary" onClick={() => mulaiBayar(r)}>
                        Bayar
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}