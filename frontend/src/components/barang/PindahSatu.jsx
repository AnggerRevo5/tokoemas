import { useState } from 'react'
import { api } from '../../services/api'
import Popup from '../common/Popup'
import SpesifikasiBarang from './SpesifikasiBarang'

// Popup pindah baki untuk satu perhiasan (tombol "Pindah" di baris tabel).
// baki: daftar baki di lokasi barang. onSelesai(pesan); onTutup(); onError(msg).
export default function PindahSatu({ item, baki, kapasitas, onSelesai, onTutup, onError }) {
  const [tujuan, setTujuan] = useState('')
  const [loading, setLoading] = useState(false)
  const bakiTujuan = baki.find((b) => String(b.id) === tujuan)

  async function pindah() {
    setLoading(true)
    try {
      const res = await api('/baki/pindah', {
        method: 'POST',
        body: JSON.stringify({ item_ids: [item.id], tray_id: Number(tujuan) }),
      })
      onSelesai(res.message)
    } catch (err) {
      onTutup()
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Popup
      judul={`Pindah baki ${item.item_no}`}
      labelKonfirmasi={bakiTujuan ? `Ya, pindah ke Baki ${bakiTujuan.no}` : 'Pilih baki tujuan'}
      loading={loading}
      nonaktif={!bakiTujuan}
      onKonfirmasi={pindah}
      onTutup={onTutup}
    >
      <SpesifikasiBarang item={item} />
      <label style={{ display: 'block', marginTop: 12 }}>Pindah ke baki</label>
      <select value={tujuan} onChange={(e) => setTujuan(e.target.value)}>
        <option value="">Pilih baki tujuan</option>
        {baki.map((b) => {
          const sekarang = b.id === item.baki_id
          const penuh = b.isi >= kapasitas
          return (
            <option key={b.id} value={b.id} disabled={sekarang || penuh}>
              Baki {b.no} ({b.isi}/{kapasitas}{sekarang ? ', baki sekarang' : penuh ? ', penuh' : ''})
            </option>
          )
        })}
      </select>
      {baki.length === 0 && (
        <p className="muted" style={{ marginTop: 10, marginBottom: 0 }}>Belum ada baki di lokasi ini. Tambah baki dulu.</p>
      )}
    </Popup>
  )
}
