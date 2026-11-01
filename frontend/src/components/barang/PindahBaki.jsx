import { useState } from 'react'
import { api } from '../../services/api'
import { gram } from '../../utils/format'
import Popup from '../common/Popup'
import { DaftarSpesifikasi } from './SpesifikasiBarang'

// Pindahkan barang (perhiasan atau logam mulia) yang dicentang ke baki lain di lokasi yang sama.
// baki: daftar baki di lokasi barang. onSelesai(pesan); onError(msg).
export default function PindahBaki({ terpilih, baki, onSelesai, onError }) {
  const [tujuan, setTujuan] = useState('')
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)

  const bakiTujuan = baki.find((b) => String(b.id) === tujuan)
  const masuk = bakiTujuan ? terpilih.filter((r) => r.baki_id !== bakiTujuan.id).length : 0
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
          <label>Pindahkan {terpilih.length} barang ke baki</label>
          <select value={tujuan} onChange={(e) => setTujuan(e.target.value)}>
            <option value="">Pilih baki tujuan</option>
            {baki.map((b) => (
              <option key={b.id} value={b.id}>
                {b.name} ({b.isi} barang)
              </option>
            ))}
          </select>
        </div>
        <div style={{ alignSelf: 'end' }}>
          <button type="button" disabled={!bakiTujuan} onClick={() => setKonfirmasi(true)}>
            Pindah baki
          </button>
        </div>
      </div>

      {konfirmasi && bakiTujuan && (
        <Popup
          judul={`Pindahkan ke ${bakiTujuan.name}?`}
          labelKonfirmasi="Ya, pindahkan"
          loading={loading}
          onKonfirmasi={pindah}
          onTutup={() => setKonfirmasi(false)}
        >
          <p className="muted">
            {terpilih.length} barang, {gram(berat)} gram. Isi {bakiTujuan.name} menjadi {bakiTujuan.isi + masuk} barang.
          </p>
          <DaftarSpesifikasi items={terpilih} />
        </Popup>
      )}
    </div>
  )
}
