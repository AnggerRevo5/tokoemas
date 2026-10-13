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
  ['bank_pegadaian', 'Bank / Pegadaian'],
  ['perorangan', 'Perorangan'],
  ['pabrik', 'Pabrik'],
]
const labelJenis = Object.fromEntries(JENIS)

export default function Hutang({ me }) {
  const bisaVoid = me && me.role === 'super_admin'

  const [rows, setRows] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [warn, setWarn] = useState('')
  const [loading, setLoading] = useState(false)

  const [mode, setMode] = useState('dana')
  const [jenis, setJenis] = useState('investor')
  const [nama, setNama] = useState('')
  const [pokok, setPokok] = useState('')
  const [mulai, setMulai] = useState(hariIni())
  const [tempo, setTempo] = useState('')
  const [note, setNote] = useState('')

  const [payId, setPayId] = useState(null)
  const [payAmount, setPayAmount] = useState('')
  const [payNote, setPayNote] = useState('')

  const [voidId, setVoidId] = useState(null)
  const [alasan, setAlasan] = useState('')

  const muat = useCallback(() => {
    api('/hutang')
      .then(setRows)
      .catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    muat()
  }, [muat])

  function resetPesan() {
    setError('')
    setInfo('')
    setWarn('')
  }

  async function simpan(e) {
    e.preventDefault()
    resetPesan()
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

  function ubahMode(m) {
    setMode(m)
    setJenis(m === 'dana' ? 'investor' : 'pabrik')
  }

  function mulaiBayar(r) {
    setPayId(r.id)
    setVoidId(null)
    setPayAmount('')
    setPayNote('')
    resetPesan()
  }

  async function konfirmasiBayar() {
    resetPesan()
    try {
      const res = await api(`/hutang/${payId}/bayar`, {
        method: 'POST',
        body: JSON.stringify({ amount: Number(payAmount), note: payNote, paid_date: hariIni() }),
      })
      setInfo(res.lunas ? 'Pembayaran tersimpan, hutang LUNAS' : `Pembayaran tersimpan, sisa ${rupiah(res.sisa)}`)
      if (res.peringatan) setWarn(res.peringatan)
      setPayId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  function mulaiBatal(r) {
    setVoidId(r.id)
    setPayId(null)
    setAlasan('')
    resetPesan()
  }

  async function konfirmasiBatal() {
    if (!alasan.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    resetPesan()
    try {
      const res = await api(`/dana/void/${voidId}`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasan }),
      })
      setInfo(res.message)
      setVoidId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  const payRow = rows.find((r) => r.id === payId)
  const voidRow = rows.find((r) => r.id === voidId)

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}
      {warn && <div className="banner" style={{ marginBottom: 16 }}>{warn}. Pembayaran tetap tersimpan.</div>}

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
              {JENIS.map(([k, l]) => (
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

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Daftar Hutang</h3>

        {payRow && (
          <div className="void-box">
            <div>
              Bayar hutang <b>{payRow.creditor_name}</b> (sisa {rupiah(payRow.sisa)}). Uang otomatis keluar dari
              kas back office.
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
              <button type="button" className="mini primary-fill" onClick={konfirmasiBayar}>Bayar</button>
              <button type="button" className="mini" onClick={() => setPayId(null)}>Tidak jadi</button>
            </div>
          </div>
        )}

        {voidRow && (
          <div className="void-box">
            <div>
              Batalkan hutang <b>{voidRow.creditor_name}</b> ({rupiah(voidRow.principal)}).
              {voidRow.dari_kas && ' Kas back office yang masuk ikut dibatalkan.'} Tulis alasannya:
            </div>
            <input value={alasan} onChange={(e) => setAlasan(e.target.value)} autoFocus />
            <div className="void-actions">
              <button type="button" className="mini danger-fill" onClick={konfirmasiBatal}>Ya, batalkan</button>
              <button type="button" className="mini" onClick={() => setVoidId(null)}>Tidak jadi</button>
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
                    <div className="muted">
                      {labelJenis[r.creditor_type] || r.creditor_type}
                      {' - '}
                      {r.dari_kas ? 'kas masuk otomatis' : 'tanpa uang masuk'}
                    </div>
                    {r.note && <div className="muted">{r.note}</div>}
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
                    )}{' '}
                    {bisaVoid && r.terbayar === 0 && (
                      <button type="button" className="mini" onClick={() => mulaiBatal(r)}>
                        Batalkan
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