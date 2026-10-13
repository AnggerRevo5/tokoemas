import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'

export default function Cuci({ me }) {
  const bisaSelesai = me.role !== 'cabang'
  const bisaVoid = me.role === 'super_admin'

  const [rows, setRows] = useState([])
  const [filter, setFilter] = useState('proses')
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [aksi, setAksi] = useState(null)
  const [alasan, setAlasan] = useState('')

  const muat = useCallback(() => {
    const q = filter === 'semua' ? '' : `?status=${filter}`
    api(`/cuci${q}`).then(setRows).catch((e) => setError(e.message))
  }, [filter])

  useEffect(() => {
    muat()
  }, [muat])

  function mulaiAksi(jenis, r) {
    setAksi({ jenis, id: r.id, sisa: r.sisa, nama: r.washer_name })
    setAlasan('')
    setError('')
    setInfo('')
  }

  async function konfirmasi() {
    setError('')
    try {
      let res
      if (aksi.jenis === 'selesai') {
        res = await api(`/cuci/${aksi.id}/selesai`, { method: 'POST', body: '{}' })
      } else {
        if (!alasan.trim()) {
          setError('Alasan pembatalan wajib diisi')
          return
        }
        res = await api(`/cuci/${aksi.id}/void`, { method: 'POST', body: JSON.stringify({ reason: alasan }) })
      }
      setInfo(res.message)
      setAksi(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}

      <div className="card">
        <h3>Daftar Cuci</h3>
        <p className="muted">
          Memulai cuci: buka menu Stok, centang barang di back office, lalu klik Cuci. Kalau pencuci
          baru mengembalikan sebagian: di menu Stok, filter Proses cuci, centang barang yang sudah
          kembali, lalu klik Selesai cuci. Barang yang selesai berstatus Bersih dan kembali ke stok
          back office, lalu dikirim ke toko pilihan lewat tombol Kirim ke toko.
        </p>

        <div className="form-grid">
          <div>
            <label>Status</label>
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="proses">Proses</option>
              <option value="selesai">Selesai</option>
              <option value="semua">Semua</option>
            </select>
          </div>
        </div>

        {aksi && (
          <div className="void-box">
            {aksi.jenis === 'selesai' ? (
              <div>
                Tandai semua sisa barang ({aksi.sisa}) di {aksi.nama} selesai? Barang kembali ke stok
                back office dengan status Bersih.
              </div>
            ) : (
              <>
                <div>Batalkan cuci #{aksi.id}. Tulis alasannya:</div>
                <input value={alasan} onChange={(e) => setAlasan(e.target.value)} autoFocus />
              </>
            )}
            <div className="void-actions">
              <button
                type="button"
                className={aksi.jenis === 'selesai' ? 'mini primary-fill' : 'mini danger-fill'}
                onClick={konfirmasi}
              >
                {aksi.jenis === 'selesai' ? 'Ya, selesai semua' : 'Ya, batalkan'}
              </button>
              <button type="button" className="mini" onClick={() => setAksi(null)}>Tidak jadi</button>
            </div>
          </div>
        )}

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Mulai</th>
                <th>Asal</th>
                <th>Barang</th>
                <th>Pencuci</th>
                <th>Progres</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && (
                <tr><td colSpan="7" className="muted">Belum ada data</td></tr>
              )}
              {rows.map((r) => (
                <tr key={r.id}>
                  <td>{r.start_date.slice(0, 10)}</td>
                  <td>{r.cabang}</td>
                  <td>
                    {r.item_name} - {r.weight_gram} g ({r.qty} pcs)
                    {r.note && <div className="muted">{r.note}</div>}
                  </td>
                  <td>{r.washer_name}</td>
                  <td>{r.qty - r.sisa} dari {r.qty} selesai</td>
                  <td>
                    <span className={r.status === 'selesai' ? 'tag in' : 'tag warn'}>
                      {r.status === 'selesai' ? 'Selesai' : 'Proses'}
                    </span>
                    {r.finish_date && <div className="muted">{r.finish_date.slice(0, 10)}</div>}
                  </td>
                  <td>
                    {bisaSelesai && r.status === 'proses' && r.sisa > 0 && (
                      <button type="button" className="mini primary" onClick={() => mulaiAksi('selesai', r)}>
                        Selesai semua
                      </button>
                    )}{' '}
                    {bisaVoid && r.status === 'proses' && r.sisa === r.qty && (
                      <button type="button" className="mini" onClick={() => mulaiAksi('batal', r)}>
                        Batalkan
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}