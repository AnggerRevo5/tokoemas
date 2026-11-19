import { useMemo, useState } from 'react'
import { api } from '../../services/api'
import { hariIni, gram, rupiah } from '../../utils/format'
import FilterProduksi, { FILTER_PRODUKSI_KOSONG } from '../common/FilterProduksi'
import BarisKosong from '../common/BarisKosong'
import Spinner from '../common/Spinner'

export default function AntrianLebur({ rows, locations, role, onSelesai, onError }) {
  const [pilih, setPilih] = useState([])
  const [tanggal, setTanggal] = useState(hariIni())
  const [catatan, setCatatan] = useState('')
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState(FILTER_PRODUKSI_KOSONG)
  const bisaBuatBatch = role === 'pj' || role === 'super_admin'
  const bisaTarik = role === 'owner' || role === 'super_admin'
  const terpilih = rows.filter((r) => pilih.includes(r.id))
  const rowsTampil = useMemo(() => {
    const q = filter.q.trim().toLowerCase()
    const dari = filter.dari
    const sampai = filter.sampai

    return rows.filter((r) => {
      const tanggalAntre = r.queued_date.slice(0, 10)
      return (
        (!q || `${r.item_no} ${r.item_name}`.toLowerCase().includes(q)) &&
        (!filter.karats?.length || filter.karats.includes(r.karat)) &&
        (!filter.berat_min || r.weight_gram >= Number(filter.berat_min)) &&
        (!filter.berat_max || r.weight_gram <= Number(filter.berat_max)) &&
        (!filter.asal || String(r.origin_location_id) === filter.asal) &&
        (!dari || tanggalAntre >= dari) &&
        (!sampai || tanggalAntre <= sampai)
      )
    })
  }, [rows, filter])
  const semuaDipilih = rowsTampil.length > 0 && rowsTampil.every((r) => pilih.includes(r.id))

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
    if (semuaDipilih) {
      const terlihat = new Set(rowsTampil.map((r) => r.id))
      setPilih((p) => p.filter((id) => !terlihat.has(id)))
      return
    }

    const berikutnya = [...new Set([...pilih, ...rowsTampil.map((r) => r.id)])]
    if (berikutnya.length > 300) {
      onError('Maksimal 300 barang dalam satu pilihan')
      return
    }
    setPilih(berikutnya)
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
    <details className="card" style={{ marginTop: 16 }} open>
      <summary style={{ cursor: 'pointer', fontWeight: 700 }}>
        Antrian Barang untuk Lebur ({rowsTampil.length} dari {rows.length} barang)
      </summary>
      <p className="muted">
        Owner memasukkan barang ke antrean tanpa mengelompokkannya. PJ memilih barang yang akan menjadi satu batch.
        Barang tetap terkunci dari transaksi lain sampai dibentuk menjadi batch atau owner menariknya dari antrean.
      </p>
      <details style={{ marginBottom: 12 }}>
        <summary style={{ cursor: 'pointer' }}>Cari dan filter barang</summary>
        <FilterProduksi
          f={filter}
          onUbah={setFilter}
          locations={locations}
          labelNama="Nama atau nomor barang"
          multipleKarat
        />
      </details>
      <div className="total-bar">
        <span>Berat antrean tampil: <b>{gram(rowsTampil.reduce((total, r) => total + r.weight_gram, 0))} g</b></span>
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {(bisaBuatBatch || bisaTarik) && (
                <th><input type="checkbox" checked={semuaDipilih} onChange={toggleSemua} aria-label="Pilih semua antrean yang tampil" /></th>
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
            {rowsTampil.length === 0 && (
              <BarisKosong kolom={bisaBuatBatch || bisaTarik ? '8' : '7'}>
                {rows.length === 0 ? 'Belum ada barang dalam antrean' : 'Tidak ada barang yang cocok dengan filter'}
              </BarisKosong>
            )}
            {rowsTampil.map((r) => (
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
    </details>
  )
}
