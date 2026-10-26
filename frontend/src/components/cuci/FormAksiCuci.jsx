import { useState } from 'react'
import { api } from '../../services/api'
import { hariIni, gram } from '../../utils/format'
import FormAksi from '../common/FormAksi'

// Form konfirmasi aksi pada satu pekerjaan cuci: kirim ke pencuci, batalkan dan kembalikan, atau batal (salah input).
// aksi = { jenis: 'kirim' | 'kembali' | 'batal', id, sisa, qty }; itemJob = barang pada pekerjaan itu (untuk kirim).
export default function FormAksiCuci({ aksi, itemJob, onMulai, onSelesai, onError, onTutup }) {
  const [loading, setLoading] = useState(false)
  const [tgl, setTgl] = useState(hariIni())
  const [pencuci, setPencuci] = useState('')
  const [alasan, setAlasan] = useState('')
  const [bAwal, setBAwal] = useState(() => {
    const awal = {}
    itemJob.forEach((i) => {
      awal[i.id] = String(gram(i.berat_awal))
    })
    return awal
  })

  async function konfirmasi(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      let res
      if (aksi.jenis === 'kirim') {
        res = await api(`/cuci/${aksi.id}/kirim`, {
          method: 'POST',
          body: JSON.stringify({
            washer_name: pencuci,
            date: tgl,
            weights: itemJob.map((i) => ({ item_id: i.id, weight: Number(bAwal[i.id]) })),
          }),
        })
      } else if (aksi.jenis === 'kembali') {
        if (!alasan.trim()) throw new Error('Alasan wajib diisi')
        res = await api(`/cuci/${aksi.id}/kembali`, {
          method: 'POST',
          body: JSON.stringify({ reason: alasan, date: tgl }),
        })
      } else {
        if (!alasan.trim()) throw new Error('Alasan pembatalan wajib diisi')
        res = await api(`/cuci/${aksi.id}/void`, {
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
    aksi.jenis === 'batal'
      ? 'Ya, batalkan'
      : aksi.jenis === 'kembali'
        ? 'Ya, kembalikan ke back office'
        : 'Kirim ke pencuci'

  return (
    <FormAksi
      onSubmit={konfirmasi}
      onTutup={onTutup}
      loading={loading}
      danger={['batal', 'kembali'].includes(aksi.jenis)}
      labelSubmit={label}
    >
      {aksi.jenis === 'kirim' && (
        <>
          <div>
            Kirim pekerjaan cuci #{aksi.id} ke pencuci. Timbang tiap barang dan isi berat awalnya.
          </div>
          <div className="form-grid">
            <div>
              <label>Nama pencuci</label>
              <input value={pencuci} onChange={(e) => setPencuci(e.target.value)} required autoFocus />
            </div>
            <div>
              <label>Tanggal kirim</label>
              <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
            </div>
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Barang</th>
                  <th>Berat sistem</th>
                  <th>Berat awal ditimbang (g)</th>
                </tr>
              </thead>
              <tbody>
                {itemJob.length === 0 && (
                  <tr><td colSpan="3" className="muted">Tidak ada barang</td></tr>
                )}
                {itemJob.map((i) => (
                  <tr key={i.id}>
                    <td>{i.item_no} {i.item_name} {i.karat}</td>
                    <td>{gram(i.weight_gram)} g</td>
                    <td>
                      <input
                        type="number"
                        step="0.001"
                        min="0.001"
                        value={bAwal[i.id] ?? ''}
                        onChange={(e) => setBAwal((p) => ({ ...p, [i.id]: e.target.value }))}
                        required
                      />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      )}
      {aksi.jenis === 'kembali' && (
        <>
          <div>
            Batalkan cuci #{aksi.id} dan kembalikan {aksi.sisa} barang yang belum selesai ke back office.
            Owner lalu mengonfirmasi penerimaannya. Barang yang sudah selesai tidak terpengaruh. Tulis
            alasannya:
          </div>
          <div className="form-grid">
            <div>
              <label>Tanggal</label>
              <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
            </div>
            <div>
              <label>Alasan</label>
              <input value={alasan} onChange={(e) => setAlasan(e.target.value)} placeholder="Contoh: tidak jadi dicuci" autoFocus />
            </div>
          </div>
        </>
      )}
      {aksi.jenis === 'batal' && (
        <>
          <div>Salah input: batalkan cuci #{aksi.id}. Barang kembali tersedia dan catatan stok dibatalkan. Tulis alasannya:</div>
          <input value={alasan} onChange={(e) => setAlasan(e.target.value)} autoFocus />
        </>
      )}
    </FormAksi>
  )
}
