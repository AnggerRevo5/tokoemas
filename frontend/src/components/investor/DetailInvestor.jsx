import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { m } from 'motion/react'
import { X } from 'lucide-react'
import { api } from '../../services/api'
import { rupiah, hariIni, tgl10 } from '../../utils/format'
import Spinner from '../common/Spinner'
import BarisKosong from '../common/BarisKosong'
import Pesan from '../common/Pesan'
import KotakBatal from '../common/KotakBatal'

const JENIS = [
  ['ambil', 'Ambil dana (pokok + bunga)'],
  ['setor', 'Setor / top-up pokok'],
  ['tarik_pokok', 'Tarik pokok'],
  ['bayar_bunga', 'Bayar bunga'],
]
const LABEL = Object.fromEntries(JENIS)
const WARNA = { setor: 'in', tarik_pokok: 'out', bayar_bunga: 'info' }

const PETUNJUK = {
  ambil: 'Satu pengambilan dicatat sekaligus sebagai tarik pokok dan bayar bunga. Isi salah satu atau keduanya.',
  setor: 'Top-up di tengah siklus baru ikut dihitung bunganya di siklus berikutnya.',
  tarik_pokok: 'Penarikan langsung mengurangi dasar bunga di jatuh tempo berikutnya. Bunga tidak ikut terambil.',
  bayar_bunga: 'Mencairkan bunga yang sudah terkumpul. Pokok tidak berubah.',
}

// Isi card: ringkasan, form pencatatan, lalu tab riwayat dan bunga per siklus.
function IsiKelola({ data, onSimpan, onPesan }) {
  const { investor: inv, riwayat, bunga } = data
  const [tab, setTab] = useState('riwayat')
  const [jenis, setJenis] = useState('ambil')
  const [jumlah, setJumlah] = useState('')
  const [pokok, setPokok] = useState('')
  const [bunganya, setBunganya] = useState('')
  const [tanggal, setTanggal] = useState(hariIni())
  const [note, setNote] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const [batalRow, setBatalRow] = useState(null)
  const [alasan, setAlasan] = useState('')

  async function catat(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const res =
        jenis === 'ambil'
          ? await api(`/investor/${inv.id}/ambil`, {
              method: 'POST',
              body: JSON.stringify({ pokok: Number(pokok) || 0, bunga: Number(bunganya) || 0, tx_date: tanggal, note }),
            })
          : await api(`/investor/${inv.id}/gerak`, {
              method: 'POST',
              body: JSON.stringify({ jenis, amount: Number(jumlah), tx_date: tanggal, note }),
            })
      setJumlah('')
      setPokok('')
      setBunganya('')
      setNote('')
      onPesan(res.message)
      onSimpan()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function konfirmasiBatal() {
    if (!alasan.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    setError('')
    try {
      const res = await api(`/investor-gerak/${batalRow.id}/void`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasan }),
      })
      setBatalRow(null)
      onPesan(res.message)
      onSimpan()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <>
      <p className="muted">
        Setor pertama {tgl10(inv.start_date)}, bunga {inv.rate_percent}% per bulan, jatuh tempo tiap tanggal{' '}
        {Number(tgl10(inv.start_date).slice(8))} (bulan yang lebih pendek memakai hari terakhir bulan itu).
      </p>

      <div className="form-grid" style={{ margin: '12px 0' }}>
        <div><label>Pokok</label><b>{rupiah(inv.pokok)}</b></div>
        <div><label>Bunga terkumpul</label><b>{rupiah(inv.bunga_total)}</b></div>
        <div><label>Sudah dibayar</label><b>{rupiah(inv.bunga_dibayar)}</b></div>
        <div><label>Belum dibayar</label><b>{rupiah(inv.bunga_belum)}</b></div>
      </div>

      <Pesan error={error} />

      <form onSubmit={catat}>
        <div className="section-title">Catat setor, tarik, atau bayar bunga</div>
        <div className="form-grid">
          <div>
            <label>Jenis</label>
            <select value={jenis} onChange={(e) => setJenis(e.target.value)}>
              {JENIS.map(([k, l]) => (
                <option key={k} value={k}>{l}</option>
              ))}
            </select>
          </div>
          {jenis === 'ambil' ? (
            <>
              <div>
                <label>Pokok yang ditarik (Rp)</label>
                <input type="number" min="0" step="1" value={pokok} max={inv.pokok} onChange={(e) => setPokok(e.target.value)} placeholder="0" />
                <div className="muted">Pokok saat ini {rupiah(inv.pokok)}</div>
              </div>
              <div>
                <label>Bunga yang diberikan (Rp)</label>
                <input type="number" min="0" step="1" value={bunganya} max={inv.bunga_belum} onChange={(e) => setBunganya(e.target.value)} placeholder="0" />
                <div className="muted">
                  Belum dibayar {rupiah(inv.bunga_belum)}
                  {inv.bunga_belum > 0 && (
                    <>
                      {' - '}
                      <button type="button" className="mini" onClick={() => setBunganya(String(inv.bunga_belum))}>Isi semua</button>
                    </>
                  )}
                </div>
              </div>
            </>
          ) : (
            <div>
              <label>Nominal (Rp)</label>
              <input type="number" min="1" step="1" value={jumlah} onChange={(e) => setJumlah(e.target.value)} required />
            </div>
          )}
          <div>
            <label>Tanggal</label>
            <input type="date" value={tanggal} min={tgl10(inv.start_date)} max={hariIni()} onChange={(e) => setTanggal(e.target.value)} required />
          </div>
          <div>
            <label>Catatan</label>
            <input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
          </div>
        </div>
        {jenis === 'ambil' && (Number(pokok) > 0 || Number(bunganya) > 0) && (
          <p>Total yang diserahkan ke investor: <b>{rupiah((Number(pokok) || 0) + (Number(bunganya) || 0))}</b></p>
        )}
        <p className="muted">{PETUNJUK[jenis]}</p>
        <button disabled={loading}>{loading ? <><Spinner />Menyimpan...</> : 'Simpan catatan'}</button>
      </form>

      <div className="tab-mini" role="tablist">
        <button type="button" role="tab" aria-selected={tab === 'riwayat'} className={tab === 'riwayat' ? 'mini aktif' : 'mini'} onClick={() => setTab('riwayat')}>
          Riwayat
        </button>
        <button type="button" role="tab" aria-selected={tab === 'bunga'} className={tab === 'bunga' ? 'mini aktif' : 'mini'} onClick={() => setTab('bunga')}>
          Bunga per siklus
        </button>
      </div>

      {batalRow && (
        <KotakBatal onKonfirmasi={konfirmasiBatal} onTutup={() => setBatalRow(null)}>
          <div>
            Batalkan catatan {LABEL[batalRow.jenis]?.toLowerCase() || batalRow.jenis} <b>{rupiah(batalRow.amount)}</b>{' '}
            tanggal {tgl10(batalRow.tx_date)}. Bunga dihitung ulang otomatis. Tulis alasannya:
          </div>
          <input value={alasan} onChange={(e) => setAlasan(e.target.value)} autoFocus />
        </KotakBatal>
      )}

      {tab === 'riwayat' ? (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Tanggal</th>
                <th>Jenis</th>
                <th className="num">Nominal</th>
                <th>Catatan</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {riwayat.length === 0 && <BarisKosong kolom="5">Belum ada catatan</BarisKosong>}
              {riwayat.map((r) => (
                <tr key={r.id} style={r.is_void ? { opacity: 0.55 } : undefined}>
                  <td>{tgl10(r.tx_date)}</td>
                  <td><span className={`tag ${WARNA[r.jenis] || 'gray'}`}>{LABEL[r.jenis] || r.jenis}</span></td>
                  <td className="num" style={r.is_void ? { textDecoration: 'line-through' } : undefined}>{rupiah(r.amount)}</td>
                  <td>
                    {r.note}
                    <div className="muted">dicatat {r.pencatat} - {r.created_at}</div>
                    {r.is_void && <div className="muted">Dibatalkan: {r.void_reason}</div>}
                  </td>
                  <td>
                    {!r.is_void && (
                      <button type="button" className="mini" onClick={() => { setBatalRow(r); setAlasan(''); setError('') }}>
                        Batalkan
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Jatuh tempo</th>
                <th className="num">Dasar bunga</th>
                <th className="num">Bunga</th>
              </tr>
            </thead>
            <tbody>
              {bunga.length === 0 && <BarisKosong kolom="3">Belum ada siklus yang jatuh tempo</BarisKosong>}
              {bunga.map((b) => (
                <tr key={b.siklus}>
                  <td>{tgl10(b.jatuh_tempo)}</td>
                  <td className="num">{rupiah(b.dasar)}</td>
                  <td className="num"><b>{rupiah(b.bunga)}</b></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  )
}

// Card Kelola investor di tengah layar. data = respons GET /investor/:id (null selama dimuat).
// Ditutup dengan tombol silang, klik latar, atau Esc.
export default function DetailInvestor({ nama, data, onTutup, onSimpan, onPesan }) {
  useEffect(() => {
    const esc = (e) => e.key === 'Escape' && onTutup()
    window.addEventListener('keydown', esc)
    return () => window.removeEventListener('keydown', esc)
  }, [onTutup])

  return createPortal(
    <div className="popup-latar latar-lebar" onMouseDown={(e) => e.target === e.currentTarget && onTutup()}>
      <m.div
        className="popup popup-lebar"
        role="dialog"
        aria-modal="true"
        aria-label={`Kelola investor ${nama}`}
        initial={{ opacity: 0, scale: 0.97, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.18, ease: 'easeOut' }}
      >
        <div className="popup-kepala">
          <h3>{nama}</h3>
          <button type="button" className="mini popup-tutup" onClick={onTutup} aria-label="Tutup">
            <X size={16} aria-hidden="true" />
          </button>
        </div>
        {data ? (
          <IsiKelola data={data} onSimpan={onSimpan} onPesan={onPesan} />
        ) : (
          <p className="muted"><Spinner />Memuat...</p>
        )}
      </m.div>
    </div>,
    document.body,
  )
}
