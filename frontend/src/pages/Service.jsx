import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'

const hariIni = () => new Date().toLocaleDateString('en-CA')
const gram = (n) => Number(Number(n).toFixed(3))
const NAMA_UMUM = ['Cincin', 'Kalung', 'Gelang', 'Anting', 'Liontin', 'Giwang']
const KARAT = Array.from({ length: 24 }, (_, i) => 24 - i)

const STATUS = {
  di_toko: ['Di toko', 'tag in'],
  dikirim_bo: ['Dalam perjalanan ke back office', 'tag warn'],
  di_bo: ['Di back office', 'tag in'],
  diproses: ['Sedang diservis', 'tag warn'],
  selesai: ['Selesai servis', 'tag in'],
  dikirim_toko: ['Dalam perjalanan ke toko', 'tag warn'],
  siap_diambil: ['Siap diambil', 'tag in'],
  diambil: ['Sudah diambil', 'tag gray'],
}

const aksiUntuk = (status, role) => {
  const admin = role === 'owner' || role === 'super_admin'
  const pj = role === 'pj' || role === 'super_admin'
  const toko = role === 'cabang' || admin
  switch (status) {
    case 'di_toko':
      return toko ? [['kirim_bo', 'Antar ke back office']] : []
    case 'dikirim_bo':
      return admin ? [['terima_bo', 'Sudah sampai'], ['belum_sampai', 'Belum sampai']] : []
    case 'di_bo':
      return pj ? [['proses', 'Kirim ke penyervice']] : []
    case 'diproses':
      return pj ? [['selesai', 'Selesai servis']] : []
    case 'selesai':
      return admin ? [['oper', 'Oper ke cabang']] : []
    case 'dikirim_toko':
      return toko ? [['terima_toko', 'Sudah diterima di toko']] : []
    case 'siap_diambil':
      return toko ? [['diambil', 'Diambil customer']] : []
    default:
      return []
  }
}

const tgl10 = (d) => (d ? d.slice(0, 10) : '')

export default function Service({ me, onUbah }) {
  const isCabang = me.role === 'cabang'
  const isPJ = me.role === 'pj'
  const bisaVoid = me.role === 'super_admin'

  const [locations, setLocations] = useState([])
  const [rows, setRows] = useState([])
  const [status, setStatus] = useState('aktif')
  const [cari, setCari] = useState('')
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(false)
  const [pilih, setPilih] = useState({})

  const [formBuka, setFormBuka] = useState(false)
  const [fLoc, setFLoc] = useState(isCabang ? String(me.location_id) : '')
  const [fCust, setFCust] = useState('')
  const [fHp, setFHp] = useState('')
  const [fNama, setFNama] = useState('')
  const [fKarat, setFKarat] = useState('')
  const [fBerat, setFBerat] = useState('')
  const [fKeluhan, setFKeluhan] = useState('')
  const [fNote, setFNote] = useState('')
  const [fTgl, setFTgl] = useState(hariIni())
  const [fPosisi, setFPosisi] = useState('bo')

  const [aksi, setAksi] = useState(null)
  const [tgl, setTgl] = useState(hariIni())
  const [penyervice, setPenyervice] = useState('')
  const [target, setTarget] = useState('')
  const [beratSetelah, setBeratSetelah] = useState('')
  const [catatan, setCatatan] = useState('')

  const [voidId, setVoidId] = useState(null)
  const [alasan, setAlasan] = useState('')

  const cabangList = locations.filter((l) => l.type === 'cabang')

  useEffect(() => {
    api('/locations')
      .then((d) => {
        setLocations(d)
        const c = d.find((l) => l.type === 'cabang')
        if (c) {
          if (!isCabang) setFLoc(String(c.id))
          setTarget(String(c.id))
        }
      })
      .catch((e) => setError(e.message))
  }, [])

  const muat = useCallback(() => {
    const p = new URLSearchParams({ status })
    if (cari.trim()) p.set('q', cari.trim())
    api(`/service?${p.toString()}`)
      .then((d) => {
        setRows(d)
        setPilih((lama) => {
          const baru = {}
          d.forEach((r) => {
            if (lama[r.id]) baru[r.id] = true
          })
          return baru
        })
      })
      .catch((e) => setError(e.message))
    if (onUbah) onUbah()
  }, [status, cari])

  useEffect(() => {
    muat()
  }, [muat])

  const terpilih = rows.filter((r) => pilih[r.id])
  const statusSet = new Set(terpilih.map((r) => r.status))
  const statusPilih = statusSet.size === 1 ? [...statusSet][0] : null
  const aksiTersedia = statusPilih ? aksiUntuk(statusPilih, me.role) : []

  useEffect(() => {
    if (aksi && !aksiTersedia.some(([k]) => k === aksi)) setAksi(null)
  }, [statusPilih])

  function toggle(id) {
    setPilih((p) => {
      const n = { ...p }
      if (n[id]) delete n[id]
      else n[id] = true
      return n
    })
  }

  async function simpan(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setLoading(true)
    try {
      const res = await api('/service', {
        method: 'POST',
        body: JSON.stringify({
          location_id: Number(fLoc),
          customer_name: fCust,
          customer_phone: fHp,
          item_name: fNama,
          karat: fKarat,
          weight_gram: Number(fBerat),
          complaint: fKeluhan,
          note: fNote,
          received_date: fTgl,
          posisi: fPosisi,
        }),
      })
      setInfo(res.message)
      setFCust('')
      setFHp('')
      setFNama('')
      setFKarat('')
      setFBerat('')
      setFKeluhan('')
      setFNote('')
      setFormBuka(false)
      setStatus('aktif')
      muat()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function jalankan(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setLoading(true)
    try {
      const res = await api('/service/aksi', {
        method: 'POST',
        body: JSON.stringify({
          action: aksi,
          ids: terpilih.map((r) => r.id),
          date: tgl,
          service_name: penyervice,
          target_location_id: Number(target),
          weight_after: Number(beratSetelah || 0),
          note: catatan,
        }),
      })
      setInfo(res.message)
      setAksi(null)
      setPilih({})
      setPenyervice('')
      setBeratSetelah('')
      setCatatan('')
      muat()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function konfirmasiBatal() {
    if (!alasan.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    setError('')
    try {
      const res = await api(`/service/void/${voidId}`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasan }),
      })
      setInfo(res.message)
      setVoidId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  const riwayat = (r) => {
    const out = [`Diterima ${tgl10(r.received_date)} di ${r.origin}`]
    if (r.sent_bo_date) out.push(`Diantar ke back office ${tgl10(r.sent_bo_date)}`)
    if (r.arrived_bo_date) out.push(`Sampai di back office ${tgl10(r.arrived_bo_date)}`)
    if (r.service_name) out.push(`Penyervice: ${r.service_name}, sejak ${tgl10(r.service_start_date)}`)
    if (r.service_end_date) out.push(`Selesai servis ${tgl10(r.service_end_date)}`)
    if (r.sent_toko_date) out.push(`Dioper ke ${r.pickup} ${tgl10(r.sent_toko_date)}`)
    if (r.pickup_date) out.push(`Diambil customer ${tgl10(r.pickup_date)}`)
    return out
  }

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}

      {!isPJ && (
        <div style={{ marginBottom: 16 }}>
          <button type="button" className="ghost" onClick={() => setFormBuka(!formBuka)}>
            {formBuka ? 'Tutup form' : '+ Terima barang service'}
          </button>
        </div>
      )}

      {formBuka && (
        <form className="card" style={{ marginBottom: 16 }} onSubmit={simpan}>
          <h3>Terima Barang Service</h3>
          <p className="muted">
            Barang milik customer: tidak masuk stok dan tidak mempengaruhi kas. Nomor nota dibuat otomatis.
          </p>

          <datalist id="nama-service">
            {NAMA_UMUM.map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>

          <div className="form-grid">
            {!isCabang && (
              <div>
                <label>Diterima di toko</label>
                <select value={fLoc} onChange={(e) => setFLoc(e.target.value)}>
                  {cabangList.map((l) => (
                    <option key={l.id} value={l.id}>{l.name}</option>
                  ))}
                </select>
              </div>
            )}
            <div>
              <label>Nama customer</label>
              <input value={fCust} onChange={(e) => setFCust(e.target.value)} required />
            </div>
            <div>
              <label>No. HP (opsional)</label>
              <input value={fHp} onChange={(e) => setFHp(e.target.value)} />
            </div>
            <div>
              <label>Nama barang</label>
              <input list="nama-service" value={fNama} onChange={(e) => setFNama(e.target.value)} placeholder="Cincin" required />
            </div>
            <div>
              <label>Kadar (opsional)</label>
              <select value={fKarat} onChange={(e) => setFKarat(e.target.value)}>
                <option value="">Tidak diketahui</option>
                {KARAT.map((n) => (
                  <option key={n} value={`${n}K`}>{n}K</option>
                ))}
              </select>
            </div>
            <div>
              <label>Berat (gram)</label>
              <input type="number" step="0.001" min="0.001" value={fBerat} onChange={(e) => setFBerat(e.target.value)} required />
            </div>
            <div className="span2">
              <label>Pekerjaan service</label>
              <input value={fKeluhan} onChange={(e) => setFKeluhan(e.target.value)} placeholder="Contoh: ganti mata, poles" required />
            </div>
            <div>
              <label>Posisi barang</label>
              <select value={fPosisi} onChange={(e) => setFPosisi(e.target.value)}>
                <option value="bo">Antar ke back office</option>
                <option value="toko">Tetap di toko</option>
              </select>
            </div>
            <div>
              <label>Tanggal</label>
              <input type="date" value={fTgl} onChange={(e) => setFTgl(e.target.value)} required />
            </div>
            <div className="span2">
              <label>Keterangan (ciri barang, kondisi)</label>
              <input value={fNote} onChange={(e) => setFNote(e.target.value)} />
            </div>
          </div>

          <button disabled={loading}>{loading ? 'Menyimpan...' : 'Simpan'}</button>
        </form>
      )}

      <div className="card">
        <div className="form-grid">
          <div>
            <label>Status</label>
            <select value={status} onChange={(e) => { setStatus(e.target.value); setPilih({}) }}>
              <option value="aktif">Belum diambil</option>
              {Object.entries(STATUS).map(([k, [l]]) => (
                <option key={k} value={k}>{l}</option>
              ))}
              <option value="semua">Semua</option>
            </select>
          </div>
          <div>
            <label>Cari customer, barang, nota, atau penyervice</label>
            <input value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Rina, SRV-00001, Anto..." />
          </div>
        </div>
        <p className="muted" style={{ marginBottom: 0 }}>{rows.length} barang service</p>
      </div>

      {terpilih.length > 0 && (
        <div className="card action-bar" style={{ marginTop: 16 }}>
          <div><b>{terpilih.length}</b> barang service dipilih</div>
          {!statusPilih && <div className="muted">Pilih barang dengan status yang sama.</div>}
          {statusPilih && aksiTersedia.length === 0 && (
            <div className="muted">Tidak ada langkah yang bisa kamu lakukan untuk status ini.</div>
          )}

          <div className="aksi-tombol">
            {aksiTersedia.map(([k, l]) => (
              <button
                key={k}
                type="button"
                className={aksi === k ? 'mini primary-fill' : 'mini primary'}
                onClick={() => {
                  setAksi(k)
                  setError('')
                  setInfo('')
                }}
              >
                {l}
              </button>
            ))}
            <button type="button" className="mini" onClick={() => { setPilih({}); setAksi(null) }}>
              Batal pilih
            </button>
          </div>

          {aksi && (
            <form className="panel-aksi" onSubmit={jalankan}>
              <div className="form-grid">
                <div>
                  <label>Tanggal</label>
                  <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
                </div>

                {aksi === 'proses' && (
                  <div>
                    <label>Nama penyervice</label>
                    <input value={penyervice} onChange={(e) => setPenyervice(e.target.value)} required />
                  </div>
                )}

                {aksi === 'oper' && (
                  <div>
                    <label>Customer mengambil di</label>
                    <select value={target} onChange={(e) => setTarget(e.target.value)}>
                      {cabangList.map((l) => (
                        <option key={l.id} value={l.id}>{l.name}</option>
                      ))}
                    </select>
                  </div>
                )}

                {aksi === 'selesai' && terpilih.length === 1 && (
                  <div>
                    <label>Berat setelah servis (g, opsional)</label>
                    <input type="number" step="0.001" min="0" value={beratSetelah} onChange={(e) => setBeratSetelah(e.target.value)} />
                  </div>
                )}

                {(aksi === 'belum_sampai' || aksi === 'selesai') && (
                  <div className="span2">
                    <label>Catatan (opsional)</label>
                    <input value={catatan} onChange={(e) => setCatatan(e.target.value)} />
                  </div>
                )}
              </div>

              {aksi === 'proses' && <p className="muted">Barang dikirim ke penyervice dan berstatus sedang diservis.</p>}
              {aksi === 'selesai' && <p className="muted">Barang kembali di back office, lalu owner mengopernya ke cabang.</p>}
              {aksi === 'oper' && <p className="muted">Pegawai membawa barang ke cabang yang dipilih.</p>}

              <button disabled={loading}>{loading ? 'Memproses...' : 'Konfirmasi'}</button>
            </form>
          )}
        </div>
      )}

      <div className="card" style={{ marginTop: 16 }}>
        {voidId && (
          <div className="void-box">
            <div>Batalkan barang service #{voidId} (salah input). Tulis alasannya:</div>
            <input value={alasan} onChange={(e) => setAlasan(e.target.value)} autoFocus />
            <div className="void-actions">
              <button type="button" className="mini danger-fill" onClick={konfirmasiBatal}>Ya, batalkan</button>
              <button type="button" className="mini" onClick={() => setVoidId(null)}>Tidak jadi</button>
            </div>
          </div>
        )}

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th></th>
                <th>Nota</th>
                <th>Customer</th>
                <th>Barang</th>
                <th>Pekerjaan</th>
                <th>Status</th>
                <th>Posisi</th>
                <th>Riwayat</th>
                {bisaVoid && <th></th>}
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && (
                <tr><td colSpan="9" className="muted">Belum ada barang service</td></tr>
              )}
              {rows.map((r) => (
                <tr key={r.id}>
                  <td>
                    <input
                      type="checkbox"
                      checked={!!pilih[r.id]}
                      disabled={aksiUntuk(r.status, me.role).length === 0}
                      onChange={() => toggle(r.id)}
                    />
                  </td>
                  <td>{r.job_no}</td>
                  <td>
                    {r.customer_name}
                    {r.customer_phone && <div className="muted">{r.customer_phone}</div>}
                  </td>
                  <td style={{ whiteSpace: 'normal', minWidth: 160 }}>
                    {r.item_name} {r.karat} - {gram(r.weight_gram)} g
                    {r.weight_after !== null && r.weight_after !== undefined && (
                      <div className="muted">setelah servis {gram(r.weight_after)} g</div>
                    )}
                    {r.note && <div className="muted">{r.note}</div>}
                  </td>
                  <td style={{ whiteSpace: 'normal', minWidth: 160 }}>
                    {r.complaint}
                    {r.service_note && <div className="muted">{r.service_note}</div>}
                  </td>
                  <td>
                    {r.status === 'dikirim_bo' && r.belum_sampai ? (
                      <span className="tag out">Belum sampai</span>
                    ) : (
                      <span className={STATUS[r.status][1]}>{STATUS[r.status][0]}</span>
                    )}
                    {r.status === 'dikirim_bo' && r.belum_sampai && r.belum_sampai_note && (
                      <div className="muted">{r.belum_sampai_note}</div>
                    )}
                    {r.service_name && ['diproses'].includes(r.status) && (
                      <div className="muted">oleh {r.service_name}</div>
                    )}
                  </td>
                  <td>
                    {r.location}
                    {r.pickup && <div className="muted">diambil di {r.pickup}</div>}
                  </td>
                  <td style={{ whiteSpace: 'normal', minWidth: 220 }}>
                    {riwayat(r).map((t, i) => (
                      <div className="muted" key={i}>{t}</div>
                    ))}
                  </td>
                  {bisaVoid && (
                    <td>
                      {['di_toko', 'dikirim_bo', 'di_bo'].includes(r.status) && (
                        <button type="button" className="mini" onClick={() => { setVoidId(r.id); setAlasan(''); setError(''); setInfo('') }}>
                          Batalkan
                        </button>
                      )}
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}