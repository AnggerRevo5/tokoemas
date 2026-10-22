import { useState } from 'react'
import { api } from '../../services/api'
import { hariIni, gram } from '../../utils/format'
import { KARAT } from '../../constants/barang'
import FormAksi from '../common/FormAksi'

// Form konfirmasi aksi pada satu proses lebur: leburan kotor, cukim, tidak jadi (kembalikan), atau batalkan langkah.
// aksi = { jenis: 'leburan' | 'cukim' | 'kembali' | 'undo', id }; batch = baris lebur yang dipilih.
export default function FormAksiLebur({ aksi, batch, onMulai, onSelesai, onError, onTutup }) {
  const [loading, setLoading] = useState(false)
  const [fTgl, setFTgl] = useState(hariIni())
  const [fCut, setFCut] = useState('')
  const [fLeb, setFLeb] = useState('')
  const [fCuk, setFCuk] = useState('')
  const [fKadar, setFKadar] = useState('')
  const [fNote, setFNote] = useState('')
  const [alasan, setAlasan] = useState('')

  async function kirimAksi(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      let res
      if (aksi.jenis === 'leburan') {
        res = await api(`/lebur/${aksi.id}/leburan`, {
          method: 'POST',
          body: JSON.stringify({ cut_weight: Number(fCut), leburan_weight: Number(fLeb), date: fTgl, note: fNote }),
        })
      } else if (aksi.jenis === 'cukim') {
        res = await api(`/lebur/${aksi.id}/cukim`, {
          method: 'POST',
          body: JSON.stringify({ cukim_weight: Number(fCuk), karat: fKadar, date: fTgl, note: fNote }),
        })
      } else if (aksi.jenis === 'kembali') {
        if (!alasan.trim()) throw new Error('Alasan wajib diisi')
        res = await api(`/lebur/${aksi.id}/kembali`, {
          method: 'POST',
          body: JSON.stringify({ reason: alasan, date: fTgl }),
        })
      } else {
        if (!alasan.trim()) throw new Error('Alasan pembatalan wajib diisi')
        res = await api(`/lebur/${aksi.id}/undo`, {
          method: 'POST',
          body: JSON.stringify({ reason: alasan }),
        })
      }
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const label =
    aksi.jenis === 'undo' ? 'Ya, batalkan' : aksi.jenis === 'kembali' ? 'Ya, kembalikan ke back office' : 'Simpan'

  return (
    <FormAksi
      onSubmit={kirimAksi}
      onTutup={onTutup}
      loading={loading}
      danger={['undo', 'kembali'].includes(aksi.jenis)}
      labelSubmit={label}
    >
      {aksi.jenis === 'leburan' && (
        <>
          <div>
            <b>Lebur #{aksi.id}: catat leburan kotor.</b> Berat awal {gram(batch.source_weight)} g. Isi
            berat setelah barang dipotong dan permata dilepas, lalu berat leburan kotor.
          </div>
          <div className="form-grid">
            <div>
              <label>Tanggal</label>
              <input type="date" value={fTgl} onChange={(e) => setFTgl(e.target.value)} required />
            </div>
            <div>
              <label>Berat setelah dipotong (g)</label>
              <input type="number" step="0.001" min="0.001" value={fCut} onChange={(e) => setFCut(e.target.value)} required />
            </div>
            <div>
              <label>Berat leburan kotor (g)</label>
              <input type="number" step="0.001" min="0.001" value={fLeb} onChange={(e) => setFLeb(e.target.value)} required />
            </div>
            <div>
              <label>Catatan</label>
              <input value={fNote} onChange={(e) => setFNote(e.target.value)} placeholder="Contoh: permata 1,5 g dilepas" />
            </div>
          </div>
        </>
      )}
      {aksi.jenis === 'cukim' && (
        <>
          <div>
            <b>Lebur #{aksi.id}: catat cukim.</b> Berat leburan kotor {gram(batch.leburan_weight)} g. Isi
            berat setelah dimurnikan.
          </div>
          <div className="form-grid">
            <div>
              <label>Tanggal</label>
              <input type="date" value={fTgl} onChange={(e) => setFTgl(e.target.value)} required />
            </div>
            <div>
              <label>Berat cukim (g)</label>
              <input type="number" step="0.001" min="0.001" value={fCuk} onChange={(e) => setFCuk(e.target.value)} required />
            </div>
            <div>
              <label>Kadar (opsional)</label>
              <select value={fKadar} onChange={(e) => setFKadar(e.target.value)}>
                <option value="">Tidak diisi</option>
                {KARAT.map((n) => (
                  <option key={n} value={`${n}K`}>{n}K</option>
                ))}
              </select>
            </div>
            <div>
              <label>Catatan</label>
              <input value={fNote} onChange={(e) => setFNote(e.target.value)} />
            </div>
          </div>
        </>
      )}
      {aksi.jenis === 'kembali' && (
        <>
          <div>
            <b>Lebur #{aksi.id}: tidak jadi dilebur.</b> Semua barang kembali tersedia di stok back office.
            Tulis alasannya:
          </div>
          <div className="form-grid">
            <div>
              <label>Tanggal</label>
              <input type="date" value={fTgl} onChange={(e) => setFTgl(e.target.value)} required />
            </div>
            <div>
              <label>Alasan</label>
              <input value={alasan} onChange={(e) => setAlasan(e.target.value)} placeholder="Contoh: tidak jadi dilebur" autoFocus />
            </div>
          </div>
        </>
      )}
      {aksi.jenis === 'undo' && (
        <>
          <div>
            <b>Batalkan langkah terakhir lebur #{aksi.id}.</b> Tahap kembali satu langkah. Tulis alasannya:
          </div>
          <input value={alasan} onChange={(e) => setAlasan(e.target.value)} placeholder="Contoh: salah input berat" autoFocus />
        </>
      )}
    </FormAksi>
  )
}
