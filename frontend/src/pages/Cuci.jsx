import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'

const hariIni = () => new Date().toLocaleDateString('en-CA')
const gram = (n) => Number(Number(n).toFixed(3))

const STATUS = {
  menunggu: ['Menunggu PJ', 'tag out'],
  proses: ['Proses', 'tag warn'],
  selesai: ['Selesai', 'tag in'],
  batal: ['Dibatalkan', 'tag gray'],
}

export default function Cuci({ me }) {
  const isCabang = me.role === 'cabang'
  const bisaCatat = me.role === 'pj' || me.role === 'super_admin'
  const bisaVoid = me.role === 'super_admin'

  const [filter, setFilter] = useState('aktif')
  const [rows, setRows] = useState([])
  const [items, setItems] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(false)

  const [aksi, setAksi] = useState(null)
  const [tgl, setTgl] = useState(hariIni())
  const [pencuci, setPencuci] = useState('')
  const [alasan, setAlasan] = useState('')
  const [bAwal, setBAwal] = useState({})

  const [pilih, setPilih] = useState({})
  const [tglItem, setTglItem] = useState(hariIni())
  const [bSesudah, setBSesudah] = useState({})
  const [cari, setCari] = useState('')

  const muat = useCallback(() => {
    api(`/cuci?status=${filter}`).then(setRows).catch((e) => setError(e.message))
    if (!isCabang) {
      api('/cucian/barang')
        .then((d) => {
          setItems(d)
          setPilih((lama) => {
            const baru = {}
            d.forEach((r) => {
              if (lama[r.id] && r.job_status === 'proses') baru[r.id] = true
            })
            return baru
          })
        })
        .catch(() => {})
    }
  }, [filter, isCabang])

  useEffect(() => {
    muat()
  }, [muat])

  function mulaiAksi(jenis, r) {
    setAksi({ jenis, id: r.id, sisa: r.sisa, qty: r.qty })
    setTgl(hariIni())
    setPencuci('')
    setAlasan('')
    setError('')
    setInfo('')
    if (jenis === 'kirim') {
      const awal = {}
      items.filter((i) => i.job_id === r.id).forEach((i) => {
        awal[i.id] = String(gram(i.berat_awal))
      })
      setBAwal(awal)
    }
  }

  const itemJob = aksi && aksi.jenis === 'kirim' ? items.filter((i) => i.job_id === aksi.id) : []

  async function konfirmasi(e) {
    e.preventDefault()
    setError('')
    setInfo('')
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
      setInfo(res.message)
      setAksi(null)
      muat()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const tampilItems = items.filter((r) => {
    const s = cari.trim().toLowerCase()
    if (!s) return true
    return (
      r.item_name.toLowerCase().includes(s) ||
      (r.washer_name || '').toLowerCase().includes(s) ||
      (r.item_no || '').toLowerCase().includes(s)
    )
  })
  const bisaDipilih = tampilItems.filter((r) => r.job_status === 'proses')
  const terpilih = items.filter((r) => pilih[r.id])
  const nilaiSesudah = (r) => (bSesudah[r.id] !== undefined ? bSesudah[r.id] : String(gram(r.berat_awal)))
  const totalAwal = terpilih.reduce((s, r) => s + r.berat_awal, 0)
  const totalSesudah = terpilih.reduce((s, r) => s + (Number(nilaiSesudah(r)) || 0), 0)
  const semuaDipilih = bisaDipilih.length > 0 && bisaDipilih.every((r) => pilih[r.id])

  function toggle(id) {
    setPilih((p) => {
      const n = { ...p }
      if (n[id]) delete n[id]
      else n[id] = true
      return n
    })
  }

  function toggleSemua() {
    if (semuaDipilih) {
      setPilih({})
    } else {
      const n = {}
      bisaDipilih.forEach((r) => (n[r.id] = true))
      setPilih(n)
    }
  }

  async function selesaiBarang(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setLoading(true)
    try {
      const res = await api('/cucian/selesai', {
        method: 'POST',
        body: JSON.stringify({
          finish_date: tglItem,
          items: terpilih.map((r) => ({ item_id: r.id, weight_after: Number(nilaiSesudah(r)) })),
        }),
      })
      setInfo(res.message)
      setPilih({})
      setBSesudah({})
      muat()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}

      <div className="card">
        <h3>Daftar Cuci</h3>
        <p className="muted">
          Owner memilih barang di menu Stok lalu klik Cuci. PJ menimbang tiap barang (berat awal) lalu mengirimnya
          ke pencuci. Saat barang kembali, PJ menimbang lagi (berat sesudah) dan mencatat susutnya. Barang yang
          selesai dikembalikan ke back office dengan berat terbaru, lalu owner mengonfirmasi sudah diterima.
          Kalau tidak jadi dicuci, PJ bisa membatalkan dan mengembalikan barang yang belum selesai.
        </p>

        <div className="form-grid">
          <div>
            <label>Tampilkan</label>
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="aktif">Belum selesai</option>
              <option value="menunggu">Menunggu PJ</option>
              <option value="proses">Proses</option>
              <option value="selesai">Selesai</option>
              <option value="batal">Dibatalkan</option>
              <option value="semua">Semua</option>
            </select>
          </div>
        </div>

        {aksi && (
          <form className="void-box" onSubmit={konfirmasi}>
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
            <div className="void-actions">
              <button
                className={['batal', 'kembali'].includes(aksi.jenis) ? 'mini danger-fill' : 'mini primary-fill'}
                disabled={loading}
              >
                {loading
                  ? 'Memproses...'
                  : aksi.jenis === 'batal'
                    ? 'Ya, batalkan'
                    : aksi.jenis === 'kembali'
                      ? 'Ya, kembalikan ke back office'
                      : 'Kirim ke pencuci'}
              </button>
              <button type="button" className="mini" onClick={() => setAksi(null)}>Tidak jadi</button>
            </div>
          </form>
        )}

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Mulai</th>
                <th>Asal</th>
                <th>Barang</th>
                <th>Pencuci</th>
                <th>Progres dan susut</th>
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
                  <td>
                    #{r.id}
                    <div className="muted">{r.start_date.slice(0, 10)}</div>
                  </td>
                  <td>{r.cabang}</td>
                  <td style={{ whiteSpace: 'normal', minWidth: 180 }}>
                    {r.item_name} - {gram(r.weight_gram)} g ({r.qty} pcs)
                    {r.note && <div className="muted">{r.note}</div>}
                  </td>
                  <td>{r.washer_name || <span className="muted">Belum dikirim</span>}</td>
                  <td>
                    {r.status === 'batal' ? (
                      <>{r.qty - r.cancel_returned} selesai, {r.cancel_returned} dikembalikan</>
                    ) : (
                      <>{r.qty - r.sisa} dari {r.qty} selesai</>
                    )}
                    {r.berat_sesudah > 0 && (
                      <div className="muted">
                        {gram(r.berat_awal_selesai)} g jadi {gram(r.berat_sesudah)} g, susut{' '}
                        {gram(r.berat_awal_selesai - r.berat_sesudah)} g
                      </div>
                    )}
                  </td>
                  <td>
                    <span className={STATUS[r.status][1]}>{STATUS[r.status][0]}</span>
                    {r.finish_date && <div className="muted">{r.finish_date.slice(0, 10)}</div>}
                    {r.status === 'batal' && r.cancel_reason && <div className="muted">{r.cancel_reason}</div>}
                  </td>
                  <td>
                    {bisaCatat && r.status === 'menunggu' && (
                      <button type="button" className="mini primary" onClick={() => mulaiAksi('kirim', r)}>
                        Kirim ke pencuci
                      </button>
                    )}{' '}
                    {bisaCatat && (r.status === 'menunggu' || r.status === 'proses') && r.sisa > 0 && (
                      <button type="button" className="mini" onClick={() => mulaiAksi('kembali', r)}>
                        Batalkan, kembalikan ke back office
                      </button>
                    )}{' '}
                    {bisaVoid && (r.status === 'menunggu' || r.status === 'proses') && r.sisa === r.qty && (
                      <button type="button" className="mini" onClick={() => mulaiAksi('batal', r)}>
                        Salah input (batalkan)
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {!isCabang && (
        <div className="card" style={{ marginTop: 16 }}>
          <h3>Barang di Pencuci</h3>
          <p className="muted">
            {bisaCatat
              ? 'Centang barang yang sudah dikembalikan pencuci, isi berat sesudah cuci (susut tercatat otomatis), lalu klik Selesai cuci. Barang yang belum dikirim ke pencuci belum bisa dipilih.'
              : 'Daftar barang yang sedang dalam proses cuci.'}
          </p>

          <div className="form-grid">
            <div>
              <label>Cari barang atau pencuci</label>
              <input value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Cincin, Adip..." />
            </div>
          </div>

          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  {bisaCatat && (
                    <th>
                      <input type="checkbox" checked={semuaDipilih} onChange={toggleSemua} />
                    </th>
                  )}
                  <th>No</th>
                  <th>Barang</th>
                  <th>Berat awal</th>
                  <th>Asal</th>
                  <th>Pencuci</th>
                  <th>Cuci</th>
                </tr>
              </thead>
              <tbody>
                {tampilItems.length === 0 && (
                  <tr><td colSpan="7" className="muted">Tidak ada barang yang sedang dicuci</td></tr>
                )}
                {tampilItems.map((r) => (
                  <tr key={r.id}>
                    {bisaCatat && (
                      <td>
                        <input
                          type="checkbox"
                          checked={!!pilih[r.id]}
                          disabled={r.job_status !== 'proses'}
                          onChange={() => toggle(r.id)}
                        />
                      </td>
                    )}
                    <td>{r.item_no}</td>
                    <td>{r.item_name} {r.karat}</td>
                    <td>{r.job_status === 'proses' ? `${gram(r.berat_awal)} g` : <span className="muted">belum ditimbang</span>}</td>
                    <td>{r.asal}</td>
                    <td>{r.washer_name || <span className="muted">Belum dikirim</span>}</td>
                    <td>
                      <span className={STATUS[r.job_status][1]}>{STATUS[r.job_status][0]}</span>
                      <div className="muted">#{r.job_id}</div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          {bisaCatat && terpilih.length > 0 && (
            <form className="panel-aksi" onSubmit={selesaiBarang}>
              <div>
                <b>{terpilih.length}</b> barang dipilih. Timbang tiap barang dan isi berat sesudah cuci:
              </div>
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>Barang</th>
                      <th>Berat awal</th>
                      <th>Berat sesudah (g)</th>
                      <th>Susut</th>
                    </tr>
                  </thead>
                  <tbody>
                    {terpilih.map((r) => {
                      const susut = r.berat_awal - (Number(nilaiSesudah(r)) || 0)
                      return (
                        <tr key={r.id}>
                          <td>{r.item_no} {r.item_name} {r.karat}</td>
                          <td>{gram(r.berat_awal)} g</td>
                          <td>
                            <input
                              type="number"
                              step="0.001"
                              min="0.001"
                              max={r.berat_awal}
                              value={nilaiSesudah(r)}
                              onChange={(e) => setBSesudah((p) => ({ ...p, [r.id]: e.target.value }))}
                              required
                            />
                          </td>
                          <td className={susut < 0 ? 'neg' : ''}>{gram(susut)} g</td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
              <p className="muted">
                Total: {gram(totalAwal)} g jadi {gram(totalSesudah)} g, susut {gram(totalAwal - totalSesudah)} g.
                Barang dikembalikan ke back office dengan berat terbaru dan menunggu konfirmasi owner.
              </p>
              <div className="form-grid">
                <div>
                  <label>Tanggal selesai</label>
                  <input type="date" value={tglItem} onChange={(e) => setTglItem(e.target.value)} required />
                </div>
              </div>
              <button disabled={loading}>{loading ? 'Memproses...' : 'Selesai cuci'}</button>
            </form>
          )}
        </div>
      )}
    </div>
  )
}