import { useState } from 'react'
import { api } from '../../services/api'
import { gram } from '../../utils/format'
import Popup from '../common/Popup'
import { DaftarSpesifikasi } from './SpesifikasiBarang'

// Pindahkan perhiasan yang dicentang ke baki lain di lokasi yang sama.
// baki: daftar baki di lokasi barang. onSelesai(pesan); onError(msg).
export default function PindahBaki({ terpilih, baki, kapasitas, onSelesai, onError }) {
  const [tujuan, setTujuan] = useState('')
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)

  const bakiTujuan = baki.find((b) => String(b.id) === tujuan)
  const masuk = bakiTujuan ? terpilih.filter((r) => r.baki_id !== bakiTujuan.id).length : 0
  const sisa = bakiTujuan ? kapasitas - bakiTujuan.isi : 0
  const kurang = bakiTujuan && masuk > sisa
  const berat = terpilih.reduce((s, r) => s + r.weight_gram, 0)

  async function pindah() {
    setLoading(true)
    try {
      const res = await api('/baki/pindah', {
        method: 'POST',
        body: JSON.stringify({ item_ids: terpilih.map((r) => r.id), tray_id: Number(tujuan) }),
      })
      setKonfirmasi(false)
      setTujuan('')
      onSelesai(res.message)
    } catch (err) {
      setKonfirmasi(false)
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card" style={{ marginTop: 16 }}>
      <div className="form-grid">
        <div>
          <label>Pindahkan {terpilih.length} perhiasan ke baki</label>
          <select value={tujuan} onChange={(e) => setTujuan(e.target.value)}>
            <option value="">Pilih baki tujuan</option>
            {baki.map((b) => (
              <option key={b.id} value={b.id} disabled={b.isi >= kapasitas}>
                Baki {b.no} ({b.isi}/{kapasitas}{b.isi >= kapasitas ? ', penuh' : ''})
              </option>
            ))}
          </select>
        </div>
        <div style={{ alignSelf: 'end' }}>
          <button type="button" disabled={!bakiTujuan || kurang} onClick={() => setKonfirmasi(true)}>
            Pindah baki
          </button>
        </div>
      </div>
      {kurang && (
        <p className="neg" style={{ marginTop: 10, marginBottom: 0 }}>
          Baki {bakiTujuan.no} hanya punya sisa tempat {sisa}, sedangkan barang yang dipindah {masuk}.
        </p>
      )}

      {konfirmasi && bakiTujuan && (
        <Popup
          judul={`Pindahkan ke Baki ${bakiTujuan.no}?`}
          labelKonfirmasi="Ya, pindahkan"
          loading={loading}
          onKonfirmasi={pindah}
          onTutup={() => setKonfirmasi(false)}
        >
          <p className="muted">
            {terpilih.length} perhiasan, {gram(berat)} gram. Isi Baki {bakiTujuan.no} menjadi {bakiTujuan.isi + masuk}/{kapasitas}.
          </p>
          <DaftarSpesifikasi items={terpilih} />
        </Popup>
      )}
    </div>
  )
}
