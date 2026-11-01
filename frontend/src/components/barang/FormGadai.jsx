import { useState } from 'react'
import { api } from '../../services/api'
import { gram, hariIni, rupiah } from '../../utils/format'
import Popup from '../common/Popup'
import Spinner from '../common/Spinner'
import SpesifikasiBarang, { DaftarSpesifikasi } from './SpesifikasiBarang'

// Form menggadaikan satu atau beberapa barang dalam satu akad (satu dana, satu cicilan). Sebelum disimpan muncul card berisi spesifikasi barang dan rincian
// dana serta cicilan; baru setelah dikonfirmasi tercatat.
// onMulai(): bersihkan pesan; onSelesai(pesan); onError(pesan).
export default function FormGadai({ items, onMulai, onSelesai, onError }) {
  const item = items[0]
  const jumlah = items.length
  const totalBerat = items.reduce((s, r) => s + r.weight_gram, 0)
  const [f, setF] = useState({ pegadaian: '', dana: '', tenor: '', cicilan: '', tgl: hariIni(), tempo: '', note: '' })
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)
  const ubah = (k) => (e) => setF((x) => ({ ...x, [k]: e.target.value }))

  const dana = Number(f.dana) || 0
  const tenor = Number(f.tenor) || 0
  const cicilan = Number(f.cicilan) || 0
  const total = tenor * cicilan
  const biaya = total - dana

  function periksa(e) {
    e.preventDefault()
    onMulai()
    if (!f.pegadaian.trim()) return onError('Nama pegadaian wajib diisi')
    if (dana <= 0) return onError('Dana yang diterima wajib diisi')
    if (!Number.isInteger(tenor) || tenor < 1) return onError('Masa cicilan minimal 1 bulan')
    if (cicilan <= 0) return onError('Cicilan per bulan wajib diisi')
    if (total < dana) return onError('Total cicilan lebih kecil dari dana yang diterima')
    setKonfirmasi(true)
  }

  async function simpan() {
    setLoading(true)
    try {
      const res = await api('/gadai', {
        method: 'POST',
        body: JSON.stringify({
          item_ids: items.map((r) => r.id),
          pegadaian: f.pegadaian.trim(),
          dana,
          tenor,
          cicilan,
          date: f.tgl,
          jatuh_tempo_pertama: f.tempo,
          note: f.note.trim(),
        }),
      })
      onSelesai(res.message)
    } catch (err) {
      setKonfirmasi(false)
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="panel-aksi" onSubmit={periksa}>
      <p className="muted" style={{ marginTop: 0 }}>
        {jumlah === 1 ? (
          <>Menggadaikan <b>{item.item_no} {item.item_name}</b> ({item.weight_gram} g)</>
        ) : (
          <>Menggadaikan <b>{jumlah} barang</b> dalam satu gadai ({gram(totalBerat)} g). Dana dan cicilan di bawah berlaku untuk seluruh barang.</>
        )}
      </p>
      {jumlah > 1 && <DaftarSpesifikasi items={items} />}
      <div className="form-grid">
        <div>
          <label>Nama pegadaian</label>
          <input value={f.pegadaian} onChange={ubah('pegadaian')} placeholder="Contoh: Pegadaian Kepanjen" maxLength={100} required />
        </div>
        <div>
          <label>Tanggal gadai</label>
          <input type="date" value={f.tgl} onChange={ubah('tgl')} required />
        </div>
        <div>
          <label>Dana yang diterima (Rp)</label>
          <input type="number" min="1" value={f.dana} onChange={ubah('dana')} required />
        </div>
        <div>
          <label>Masa cicilan (bulan)</label>
          <input type="number" min="1" max="120" step="1" value={f.tenor} onChange={ubah('tenor')} required />
        </div>
        <div>
          <label>Cicilan per bulan (Rp)</label>
          <input type="number" min="1" value={f.cicilan} onChange={ubah('cicilan')} required />
        </div>
        <div>
          <label>Jatuh tempo pertama (kosong = 1 bulan setelah tanggal gadai)</label>
          <input type="date" value={f.tempo} onChange={ubah('tempo')} />
        </div>
        <div className="span2">
          <label>Catatan (opsional)</label>
          <input value={f.note} onChange={ubah('note')} maxLength={200} />
        </div>
      </div>
      {dana > 0 && tenor > 0 && cicilan > 0 && (
        <div className="total-bar">
          <span>
            Total cicilan <b>{rupiah(total)}</b>
            {biaya >= 0 && <> | biaya gadai <b className="utama">{rupiah(biaya)}</b></>}
            {biaya < 0 && <> | <b className="neg">total cicilan lebih kecil dari dana</b></>}
          </span>
        </div>
      )}
      <button disabled={loading}>{loading ? <><Spinner />Memproses...</> : 'Lanjut'}</button>

      {konfirmasi && (
        <Popup
          judul={jumlah === 1 ? `Gadaikan ${item.item_no}?` : `Gadaikan ${jumlah} barang?`}
          labelKonfirmasi="Ya, gadaikan"
          loading={loading}
          onKonfirmasi={simpan}
          onTutup={() => setKonfirmasi(false)}
        >
          <p className="muted">Periksa barang dan rincian gadai sebelum disimpan:</p>
          {jumlah === 1 ? <SpesifikasiBarang item={item} /> : <DaftarSpesifikasi items={items} />}
          <table className="tabel-spek" style={{ marginTop: 12 }}>
            <tbody>
              {jumlah > 1 && <tr><th>Jumlah barang</th><td>{jumlah} barang, {gram(totalBerat)} g</td></tr>}
              <tr><th>Pegadaian</th><td>{f.pegadaian.trim()}</td></tr>
              <tr><th>Tanggal gadai</th><td>{f.tgl}</td></tr>
              <tr><th>Dana diterima</th><td>{rupiah(dana)} (masuk ke kas {item.lokasi})</td></tr>
              <tr><th>Cicilan</th><td>{tenor} x {rupiah(cicilan)}</td></tr>
              <tr><th>Total cicilan</th><td>{rupiah(total)}</td></tr>
              <tr><th>Biaya gadai</th><td>{rupiah(biaya)}</td></tr>
              <tr><th>Jatuh tempo pertama</th><td>{f.tempo || '1 bulan setelah tanggal gadai'}</td></tr>
              {f.note.trim() && <tr><th>Catatan</th><td>{f.note.trim()}</td></tr>}
            </tbody>
          </table>
        </Popup>
      )}
    </form>
  )
}
