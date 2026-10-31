import { useState, useEffect } from 'react'
import { api } from '../../services/api'
import { hariIni } from '../../utils/format'
import { NAMA_UMUM, KARAT } from '../../constants/barang'
import PilihanLokasi from '../common/PilihanLokasi'
import { m } from 'motion/react'
import { Plus, X } from 'lucide-react'
import Spinner from '../common/Spinner'

// Tombol dan form "Terima barang service" dari customer.
// onMulai: bersihkan pesan; onSukses(res): setelah tersimpan; onError(msg).
export default function FormTerimaService({ isCabang, me, cabangList, onMulai, onSukses, onError }) {
  const [formBuka, setFormBuka] = useState(false)
  const [loading, setLoading] = useState(false)
  const [fLoc, setFLoc] = useState(isCabang ? String(me.location_id) : '')
  const [fCust, setFCust] = useState('')
  const [fHp, setFHp] = useState('')
  const [fNama, setFNama] = useState('')
  const [fKarat, setFKarat] = useState('')
  const [fBerat, setFBerat] = useState('')
  const [fKeluhan, setFKeluhan] = useState('')
  const [fNote, setFNote] = useState('')
  const [fTgl, setFTgl] = useState(hariIni())
  const [fPosisi, setFPosisi] = useState('bo')

  // pilih cabang pertama sebagai tempat terima bila akun bukan cabang
  useEffect(() => {
    if (!isCabang && cabangList.length > 0) setFLoc(String(cabangList[0].id))
  }, [isCabang, cabangList.length])

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      const res = await api('/service', {
        method: 'POST',
        body: JSON.stringify({
          location_id: Number(fLoc),
          customer_name: fCust,
          customer_phone: fHp,
          item_name: fNama,
          karat: fKarat,
          weight_gram: Number(fBerat),
          complaint: fKeluhan,
          note: fNote,
          received_date: fTgl,
          posisi: fPosisi,
        }),
      })
      setFCust('')
      setFHp('')
      setFNama('')
      setFKarat('')
      setFBerat('')
      setFKeluhan('')
      setFNote('')
      setFormBuka(false)
      onSukses(res)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <button type="button" className="ghost" onClick={() => setFormBuka(!formBuka)}>
          {formBuka ? <><X size={16} aria-hidden="true" />Tutup form</> : <><Plus size={16} aria-hidden="true" />Terima barang service</>}
        </button>
      </div>

      {formBuka && (
        <m.form
          className="card"
          style={{ marginBottom: 16 }}
          onSubmit={simpan}
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.22, ease: 'easeOut' }}
        >
          <h3>Terima Barang Service</h3>
          <p className="muted">
            Barang milik customer: tidak masuk stok dan tidak mempengaruhi kas. Nomor nota dibuat otomatis.
          </p>

          <datalist id="nama-service">
            {NAMA_UMUM.map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>

          <div className="form-section">
          <div className="section-title">Customer</div>
          <div className="form-grid">
            {!isCabang && (
              <div>
                <label>Diterima di toko</label>
                <PilihanLokasi locations={cabangList} value={fLoc} onChange={(e) => setFLoc(e.target.value)} />
              </div>
            )}
            <div>
              <label>Nama customer</label>
              <input value={fCust} onChange={(e) => setFCust(e.target.value)} required />
            </div>
            <div>
              <label>No. HP (opsional)</label>
              <input value={fHp} onChange={(e) => setFHp(e.target.value)} />
            </div>
          </div>
          </div>

          <div className="form-section">
          <div className="section-title">Barang</div>
          <div className="form-grid">
            <div>
              <label>Nama barang</label>
              <input list="nama-service" value={fNama} onChange={(e) => setFNama(e.target.value)} placeholder="Cincin" required />
            </div>
            <div>
              <label>Kadar (opsional)</label>
              <select value={fKarat} onChange={(e) => setFKarat(e.target.value)}>
                <option value="">Tidak diketahui</option>
                {KARAT.map((n) => (
                  <option key={n} value={`${n}K`}>{n}K</option>
                ))}
              </select>
            </div>
            <div>
              <label>Berat (gram)</label>
              <input type="number" step="0.001" min="0.001" value={fBerat} onChange={(e) => setFBerat(e.target.value)} required />
            </div>
            <div className="span2">
              <label>Pekerjaan service</label>
              <input value={fKeluhan} onChange={(e) => setFKeluhan(e.target.value)} placeholder="Contoh: ganti mata, poles" required />
            </div>
          </div>
          </div>

          <div className="form-section">
          <div className="section-title">Penanganan</div>
          <div className="form-grid">
            <div>
              <label>Posisi barang</label>
              <select value={fPosisi} onChange={(e) => setFPosisi(e.target.value)}>
                <option value="bo">Antar ke back office</option>
                <option value="toko">Tetap di toko</option>
              </select>
            </div>
            <div>
              <label>Tanggal</label>
              <input type="date" value={fTgl} onChange={(e) => setFTgl(e.target.value)} required />
            </div>
            <div className="span2">
              <label>Keterangan (ciri barang, kondisi)</label>
              <input value={fNote} onChange={(e) => setFNote(e.target.value)} />
            </div>
          </div>
          </div>

          <button disabled={loading}>{loading ? <><Spinner />Menyimpan...</> : 'Simpan'}</button>
        </m.form>
      )}
    </>
  )
}
