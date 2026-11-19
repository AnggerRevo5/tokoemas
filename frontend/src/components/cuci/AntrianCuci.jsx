import { useMemo, useState } from 'react'
import { api } from '../../services/api'
import { hariIni, gram } from '../../utils/format'
import FilterProduksi, { FILTER_PRODUKSI_KOSONG } from '../common/FilterProduksi'
import BarisKosong from '../common/BarisKosong'
import Spinner from '../common/Spinner'

export default function AntrianCuci({ rows, locations, role, onSelesai, onError }) {
  const [pilih, setPilih] = useState([])
  const [tanggal, setTanggal] = useState(hariIni())
  const [catatan, setCatatan] = useState('')
  const [loading, setLoading] = useState(false)
  const [filter, setFilter] = useState(FILTER_PRODUKSI_KOSONG)
  const bisaBuatPekerjaan = role === 'pj' || role === 'super_admin'
  const bisaTarik = role === 'owner' || role === 'super_admin'
  const terpilih = rows.filter((r) => pilih.includes(r.id))
  const asalTerpilih = terpilih[0]?.origin_location_id
  const rowsTampil = useMemo(() => {
    const q = filter.q.trim().toLowerCase()
    return rows.filter((r) => {
      const tanggalAntre = r.queued_date.slice(0, 10)
      return (
        (!q || `${r.item_no} ${r.item_name}`.toLowerCase().includes(q)) &&
        (!filter.karat || r.karat === filter.karat) &&
        (!filter.berat_min || r.weight_gram >= Number(filter.berat_min)) &&
        (!filter.berat_max || r.weight_gram <= Number(filter.berat_max)) &&
        (!filter.asal || String(r.origin_location_id) === filter.asal) &&
        (!filter.dari || tanggalAntre >= filter.dari) &&
        (!filter.sampai || tanggalAntre <= filter.sampai)
      )
    })
  }, [rows, filter])
  const asalSama = terpilih.every((r) => r.origin_location_id === asalTerpilih)
  const bisaPilihTampil = rowsTampil
  const semuaDipilih = bisaPilihTampil.length > 0 && bisaPilihTampil.every((r) => pilih.includes(r.id))

  function toggle(row) {
    if (pilih.includes(row.id)) {
      setPilih((p) => p.filter((id) => id !== row.id))
      return
    }
    if (pilih.length >= 300) {
      onError('Maksimal 300 barang dalam satu pekerjaan cuci')
      return
    }
    setPilih((p) => [...p, row.id])
  }

  function toggleSemua() {
    if (semuaDipilih) {
      const terlihat = new Set(bisaPilihTampil.map((r) => r.id))
      setPilih((p) => p.filter((id) => !terlihat.has(id)))
      return
    }
    const berikutnya = [...new Set([...pilih, ...bisaPilihTampil.map((r) => r.id)])]
    if (berikutnya.length > 300) {
      onError('Maksimal 300 barang dalam satu pekerjaan cuci')
      return
    }
    setPilih(berikutnya)
  }

  async function proses(e, batalkan) {
    e.preventDefault()
    setLoading(true)
    try {
      const res = await api(batalkan ? '/cuci/antrian/batal' : '/cuci/batch', {
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
        Antrean Barang untuk Cuci ({rowsTampil.length} dari {rows.length} barang)
      </summary>
      <p className="muted">
        Owner memasukkan barang satu per satu ke antrean. PJ memilih barang dari lokasi asal yang sama untuk
        membentuk satu pekerjaan cuci. Barang tetap terkunci dari transaksi lain sampai dibentuk atau ditarik owner.
      </p>
      <details style={{ marginBottom: 12 }}>
        <summary style={{ cursor: 'pointer' }}>Cari dan filter barang</summary>
        <FilterProduksi f={filter} onUbah={setFilter} locations={locations} labelNama="Nama atau nomor barang" />
      </details>
      <div className="total-bar">
        <span>Berat antrean tampil: <b>{gram(rowsTampil.reduce((total, r) => total + r.weight_gram, 0))} g</b></span>
        {asalTerpilih && <span>Lokasi asal pilihan: <b>{asalSama ? terpilih[0].asal : 'campuran'}</b></span>}
      </div>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {(bisaBuatPekerjaan || bisaTarik) && (
                <th>
                  <input
                    type="checkbox"
                    checked={semuaDipilih}
                    onChange={toggleSemua}
                    aria-label="Pilih semua barang yang tampil"
                  />
                </th>
              )}
              <th>No</th>
              <th>Barang</th>
              <th className="num">Berat</th>
              <th>Asal</th>
              <th>Tanggal antre</th>
              <th>Catatan owner</th>
            </tr>
          </thead>
          <tbody>
            {rowsTampil.length === 0 && (
              <BarisKosong kolom={bisaBuatPekerjaan || bisaTarik ? '7' : '6'}>
                {rows.length === 0 ? 'Belum ada barang dalam antrean' : 'Tidak ada barang yang cocok dengan filter'}
              </BarisKosong>
            )}
            {rowsTampil.map((r) => (
              <tr key={r.id}>
                {(bisaBuatPekerjaan || bisaTarik) && (
                  <td>
                    <input
                      type="checkbox"
                      checked={pilih.includes(r.id)}
                      disabled={pilih.length >= 300 && !pilih.includes(r.id)}
                      onChange={() => toggle(r)}
                    />
                  </td>
                )}
                <td>{r.item_no}</td>
                <td>{r.item_name} {r.karat}</td>
                <td className="num">{gram(r.weight_gram)} g</td>
                <td>{r.asal}</td>
                <td>{r.queued_date.slice(0, 10)}</td>
                <td>{r.note || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {terpilih.length > 0 && (
        <form className="panel-aksi" onSubmit={(e) => proses(e, false)}>
          <div>
            <b>{terpilih.length}</b> barang dipilih
            {asalSama ? ` dari lokasi asal ${terpilih[0].asal}.` : ' dari beberapa lokasi asal.'}
          </div>
          {bisaBuatPekerjaan && !asalSama && (
            <p className="muted">Satu pekerjaan cuci hanya dapat berisi barang dari lokasi asal yang sama.</p>
          )}
          {bisaBuatPekerjaan && (
            <div className="form-grid">
              <div>
                <label>Tanggal mulai pekerjaan</label>
                <input type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} required />
              </div>
              <div>
                <label>Catatan pekerjaan (opsional)</label>
                <input value={catatan} onChange={(e) => setCatatan(e.target.value)} maxLength="200" />
              </div>
            </div>
          )}
          <div className="aksi-tombol">
            {bisaBuatPekerjaan && (
              <button disabled={loading || !asalSama}>
                {loading ? <><Spinner />Memproses...</> : 'Jadikan satu pekerjaan cuci'}
              </button>
            )}
            {bisaTarik && (
              <button
                type="button"
                className="mini danger-fill"
                disabled={loading}
                onClick={(e) => proses(e, true)}
              >
                {loading ? <><Spinner />Memproses...</> : 'Tarik pilihan dari antrean'}
              </button>
            )}
            <button type="button" className="mini" onClick={() => setPilih([])} disabled={loading}>
              Batal pilih
            </button>
          </div>
        </form>
      )}
    </details>
  )
}
