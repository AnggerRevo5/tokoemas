import { useState } from 'react'
import { api } from '../../services/api'
import { hariIni, gram, rupiah } from '../../utils/format'
import BarisKosong from '../common/BarisKosong'
import Spinner from '../common/Spinner'

export default function AntrianLebur({ rows, role, onSelesai, onError }) {
  const [pilih, setPilih] = useState([])
  const [tanggal, setTanggal] = useState(hariIni())
  const [catatan, setCatatan] = useState('')
  const [loading, setLoading] = useState(false)
  const bisaBuatBatch = role === 'pj' || role === 'super_admin'
  const bisaTarik = role === 'owner' || role === 'super_admin'
  const terpilih = rows.filter((r) => pilih.includes(r.id))
  const semuaDipilih = rows.length > 0 && rows.every((r) => pilih.includes(r.id))

  function toggle(id) {
    if (pilih.includes(id)) {
      setPilih((p) => p.filter((n) => n !== id))
      return
    }
    if (pilih.length >= 300) {
      onError('Maksimal 300 barang dalam satu pilihan')
      return
    }
    setPilih((p) => [...p, id])
  }

  function toggleSemua() {
    setPilih(semuaDipilih ? [] : rows.slice(0, 300).map((r) => r.id))
    if (!semuaDipilih && rows.length > 300) onError('Dipilih 300 barang pertama; maksimal 300 barang per batch')
  }

  async function proses(e, batalkan = !bisaBuatBatch) {
    e.preventDefault()
    setLoading(true)
    try {
      const res = await api(batalkan ? '/lebur/antrian/batal' : '/lebur/batch', {
        method: 'POST',
        body: JSON.stringify(
          batalkan
            ? { item_ids: pilih }
            : { item_ids: pilih, start_date: tanggal, note: catatan },
        ),
      })
      setPilih([])
      setCatatan('')
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card" style={{ marginTop: 16 }}>
      <h3>Antrian Barang untuk Lebur</h3>
      <p className="muted">
        Owner memasukkan barang ke antrean tanpa mengelompokkannya. PJ memilih barang yang akan menjadi satu batch.
        Barang tetap terkunci dari transaksi lain sampai dibentuk menjadi batch atau owner menariknya dari antrean.
      </p>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {(bisaBuatBatch || bisaTarik) && (
                <th><input type="checkbox" checked={semuaDipilih} onChange={toggleSemua} aria-label="Pilih semua antrean" /></th>
              )}
              <th>No</th>
              <th>Barang</th>
              <th className="num">Berat</th>
              <th className="num">Modal</th>
              <th>Asal</th>
              <th>Tanggal antre</th>
              <th>Catatan owner</th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && <BarisKosong kolom={bisaBuatBatch || bisaTarik ? '8' : '7'}>Belum ada barang dalam antrean</BarisKosong>}
            {rows.map((r) => (
              <tr key={r.id}>
                {(bisaBuatBatch || bisaTarik) && (
                  <td>
                    <input type="checkbox" checked={pilih.includes(r.id)} onChange={() => toggle(r.id)} disabled={pilih.length >= 300 && !pilih.includes(r.id)} />
                  </td>
                )}
                <td>{r.item_no}</td>
                <td>{r.item_name} {r.karat}</td>
                <td className="num">{gram(r.weight_gram)} g</td>
                <td className="num">{r.harga_beli > 0 ? rupiah(r.harga_beli) : '-'}</td>
                <td>{r.asal}</td>
                <td>{r.queued_date.slice(0, 10)}</td>
                <td>{r.note || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {terpilih.length > 0 && (
        <form className="panel-aksi" onSubmit={proses}>
          <div><b>{terpilih.length}</b> barang dipilih untuk diproses.</div>
          {bisaBuatBatch && (
            <div className="form-grid">
              <div>
                <label>Tanggal mulai batch</label>
                <input type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} required />
              </div>
              <div>
                <label>Catatan batch (opsional)</label>
                <input value={catatan} onChange={(e) => setCatatan(e.target.value)} maxLength="200" />
              </div>
            </div>
          )}
          <div className="aksi-tombol">
            {bisaBuatBatch && <button disabled={loading}>{loading ? <><Spinner />Memproses...</> : 'Jadikan satu batch'}</button>}
            {bisaTarik && (
              <button type="button" className="mini danger-fill" disabled={loading} onClick={(e) => proses(e, true)}>
                {loading ? <><Spinner />Memproses...</> : 'Tarik pilihan dari antrean'}
              </button>
            )}
            <button type="button" className="mini" onClick={() => setPilih([])} disabled={loading}>Batal pilih</button>
          </div>
        </form>
      )}
    </div>
  )
}
