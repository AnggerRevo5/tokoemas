import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'
import { KATEGORI, labelKategori } from '../kategori'

const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)

const hariIni = () => new Date().toLocaleDateString('en-CA')

export default function Transaksi({ jenis, me }) {
  const isKas = jenis === 'kas'
  const bisaVoid = me.role === 'super_admin' || me.role === 'cabang'

  const [locations, setLocations] = useState([])
  const [locId, setLocId] = useState(me.role === 'cabang' ? me.location_id : 1)
  const [direction, setDirection] = useState('in')
  const [category, setCategory] = useState('')
  const [target, setTarget] = useState('')
  const [tanggal, setTanggal] = useState(hariIni())
  const [amount, setAmount] = useState('')
  const [itemName, setItemName] = useState('')
  const [berat, setBerat] = useState('')
  const [karat, setKarat] = useState('')
  const [qty, setQty] = useState(1)
  const [note, setNote] = useState('')
  const [rows, setRows] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(false)
  const [voidId, setVoidId] = useState(null)
  const [voidReason, setVoidReason] = useState('')

  const lokasi = locations.find((l) => l.id === Number(locId))
  const tipe = lokasi ? lokasi.type : 'cabang'
  const opsi = KATEGORI[jenis][tipe][direction]
  const cabangList = locations.filter((l) => l.type === 'cabang')

  useEffect(() => {
    api('/locations').then(setLocations).catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    const c = locations.find((l) => l.type === 'cabang')
    if (c) setTarget(String(c.id))
  }, [locations])

  useEffect(() => {
    setCategory(opsi[0][0])
  }, [tipe, direction, jenis])

  const muat = useCallback(() => {
    api(`/${jenis}?location_id=${locId}`)
      .then(setRows)
      .catch((e) => setError(e.message))
  }, [jenis, locId])

  useEffect(() => {
    muat()
  }, [muat])

  async function simpan(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setLoading(true)
    const body = isKas
      ? {
          location_id: Number(locId),
          direction,
          category,
          amount: Number(amount),
          note,
          transaction_date: tanggal,
        }
      : {
          location_id: Number(locId),
          target_location_id: category === 'ke_toko' ? Number(target) : 0,
          direction,
          category,
          item_name: itemName,
          weight_gram: Number(berat),
          karat,
          qty: Number(qty),
          note,
          transaction_date: tanggal,
        }
    try {
      const res = await api(`/${jenis}`, { method: 'POST', body: JSON.stringify(body) })
      setInfo(res.message)
      setAmount('')
      setItemName('')
      setBerat('')
      setKarat('')
      setQty(1)
      setNote('')
      muat()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  function mulaiBatal(id) {
    setVoidId(id)
    setVoidReason('')
    setError('')
    setInfo('')
  }

  async function konfirmasiBatal() {
    if (!voidReason.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    setError('')
    try {
      await api(`/${jenis}/${voidId}/void`, {
        method: 'POST',
        body: JSON.stringify({ reason: voidReason }),
      })
      setInfo('Transaksi dibatalkan')
      setVoidId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}

      <form className="card" onSubmit={simpan}>
        <h3>{isKas ? 'Input Kas' : 'Input Stok Emas'}</h3>

        <div className="form-grid">
          {me.role !== 'cabang' && (
            <div>
              <label>Lokasi</label>
              <select value={locId} onChange={(e) => setLocId(e.target.value)}>
                {locations.map((l) => (
                  <option key={l.id} value={l.id}>{l.name}</option>
                ))}
              </select>
            </div>
          )}

          <div>
            <label>Arah</label>
            <select value={direction} onChange={(e) => setDirection(e.target.value)}>
              <option value="in">{isKas ? 'Uang masuk' : 'Barang masuk'}</option>
              <option value="out">{isKas ? 'Uang keluar' : 'Barang keluar'}</option>
            </select>
          </div>

          <div>
            <label>Kategori</label>
            <select value={category} onChange={(e) => setCategory(e.target.value)}>
              {opsi.map(([k, l]) => (
                <option key={k} value={k}>{l}</option>
              ))}
            </select>
          </div>

          {!isKas && category === 'ke_toko' && (
            <div>
              <label>Kirim ke toko</label>
              <select value={target} onChange={(e) => setTarget(e.target.value)}>
                {cabangList.map((l) => (
                  <option key={l.id} value={l.id}>{l.name}</option>
                ))}
              </select>
            </div>
          )}

          <div>
            <label>Tanggal</label>
            <input type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} required />
          </div>

          {isKas ? (
            <div>
              <label>Nominal (Rp)</label>
              <input type="number" min="1" value={amount} onChange={(e) => setAmount(e.target.value)} required />
            </div>
          ) : (
            <>
              <div>
                <label>Nama barang</label>
                <input value={itemName} onChange={(e) => setItemName(e.target.value)} placeholder="Cincin, kalung, gelang..." required />
              </div>
              <div>
                <label>Berat total (gram)</label>
                <input type="number" step="0.001" min="0.001" value={berat} onChange={(e) => setBerat(e.target.value)} required />
              </div>
              <div>
                <label>Kadar</label>
                <input value={karat} onChange={(e) => setKarat(e.target.value)} placeholder="22K" />
              </div>
              <div>
                <label>Jumlah (pcs)</label>
                <input type="number" min="1" value={qty} onChange={(e) => setQty(e.target.value)} required />
              </div>
            </>
          )}

          <div className="span2">
            <label>Catatan</label>
            <input value={note} onChange={(e) => setNote(e.target.value)} />
          </div>
        </div>

        {!isKas && category === 'ke_toko' && (
          <p className="muted">Otomatis tercatat sebagai barang masuk di toko tujuan.</p>
        )}
        {!isKas && category === 'kembali_bo' && (
          <p className="muted">Otomatis tercatat sebagai barang masuk di Back Office.</p>
        )}

        <button disabled={loading}>{loading ? 'Menyimpan...' : 'Simpan'}</button>
      </form>

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Riwayat{lokasi ? ` - ${lokasi.name}` : ''}</h3>

        {voidId && (
          <div className="void-box">
            <div>Batalkan transaksi #{voidId}. Tulis alasannya:</div>
            <input
              value={voidReason}
              onChange={(e) => setVoidReason(e.target.value)}
              placeholder="Contoh: salah input nominal"
              autoFocus
            />
            <div className="void-actions">
              <button type="button" className="mini danger-fill" onClick={konfirmasiBatal}>
                Ya, batalkan
              </button>
              <button type="button" className="mini" onClick={() => setVoidId(null)}>
                Tidak jadi
              </button>
            </div>
          </div>
        )}

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Tanggal</th>
                <th>Kategori</th>
                <th>{isKas ? 'Nominal' : 'Barang'}</th>
                <th>Catatan</th>
                {bisaVoid && <th></th>}
              </tr>
            </thead>
            <tbody>
              {rows.length === 0 && (
                <tr><td colSpan="5" className="muted">Belum ada data</td></tr>
              )}
              {rows.map((r) => (
                <tr key={r.id}>
                  <td>{r.transaction_date.slice(0, 10)}</td>
                  <td>
                    <span className={r.direction === 'in' ? 'tag in' : 'tag out'}>
                      {r.direction === 'in' ? 'Masuk' : 'Keluar'}
                    </span>{' '}
                    {labelKategori(r.category)}
                  </td>
                  <td>
                    {isKas
                      ? rupiah(r.amount)
                      : `${r.item_name} - ${r.weight_gram} g ${r.karat} (${r.qty} pcs)`}
                  </td>
                  <td>{r.note}</td>
                  {bisaVoid && (
                    <td>
                      <button type="button" className="mini" onClick={() => mulaiBatal(r.id)}>
                        Batalkan
                      </button>
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