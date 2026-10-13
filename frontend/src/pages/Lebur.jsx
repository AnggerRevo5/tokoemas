import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'

const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)

const hariIni = () => new Date().toLocaleDateString('en-CA')
const awalBulan = () => {
  const d = new Date()
  return new Date(d.getFullYear(), d.getMonth(), 1).toLocaleDateString('en-CA')
}
const gram = (n) => Number(Number(n).toFixed(3))
const KADAR = Array.from({ length: 24 }, (_, i) => 24 - i)

const TAHAP = {
  proses: ['Proses lebur', 'tag warn'],
  leburan_kotor: ['Leburan kotor', 'tag warn'],
  cukim: ['Cukim siap jual', 'tag in'],
  terjual: ['Terjual', 'tag gray'],
}

export default function Lebur({ me }) {
  const bisaCatat = me.role === 'pj' || me.role === 'super_admin'
  const bisaUndo = me.role === 'super_admin'

  const [filter, setFilter] = useState('berjalan')
  const [batches, setBatches] = useState([])
  const [siap, setSiap] = useState([])
  const [penj, setPenj] = useState({ rows: [], total_modal: 0, total_jual: 0, total_untung: 0 })
  const [dari, setDari] = useState(awalBulan())
  const [sampai, setSampai] = useState(hariIni())
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(false)

  const [aksi, setAksi] = useState(null)
  const [fTgl, setFTgl] = useState(hariIni())
  const [fCut, setFCut] = useState('')
  const [fLeb, setFLeb] = useState('')
  const [fCuk, setFCuk] = useState('')
  const [fKadar, setFKadar] = useState('')
  const [fNote, setFNote] = useState('')
  const [alasan, setAlasan] = useState('')

  const [pilih, setPilih] = useState({})
  const [penadah, setPenadah] = useState('')
  const [hargaJual, setHargaJual] = useState('')
  const [jualTgl, setJualTgl] = useState(hariIni())
  const [jualNote, setJualNote] = useState('')
  const [voidSale, setVoidSale] = useState(null)
  const [alasanSale, setAlasanSale] = useState('')

  const muat = useCallback(() => {
    api(`/lebur?status=${filter}`).then(setBatches).catch((e) => setError(e.message))
    api('/penjualan/siap')
      .then((d) => {
        setSiap(d)
        setPilih((lama) => {
          const baru = {}
          d.forEach((r) => {
            if (lama[r.id]) baru[r.id] = true
          })
          return baru
        })
      })
      .catch((e) => setError(e.message))
    api(`/penjualan?from=${dari}&to=${sampai}`).then(setPenj).catch((e) => setError(e.message))
  }, [filter, dari, sampai])

  useEffect(() => {
    muat()
  }, [muat])

  const batchAksi = aksi ? batches.find((b) => b.id === aksi.id) : null

  function bukaAksi(jenis, b) {
    setAksi({ jenis, id: b.id })
    setFTgl(hariIni())
    setFCut('')
    setFLeb('')
    setFCuk('')
    setFKadar('')
    setFNote('')
    setAlasan('')
    setError('')
    setInfo('')
  }

  async function kirimAksi(e) {
    e.preventDefault()
    setError('')
    setInfo('')
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
      } else {
        if (!alasan.trim()) throw new Error('Alasan pembatalan wajib diisi')
        res = await api(`/lebur/${aksi.id}/undo`, {
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

  const terpilih = siap.filter((r) => pilih[r.id])
  const nCukim = terpilih.filter((r) => r.jenis === 'hasil_lebur').length
  const nLM = terpilih.filter((r) => r.jenis === 'logam_mulia').length
  const modalPilih = terpilih.reduce((s, r) => s + r.harga_beli, 0)
  const beratPilih = terpilih.reduce((s, r) => s + r.weight_gram, 0)
  const salahPilih =
    nCukim > 0 && nLM > 0 ? 'Cukim dan logam mulia dijual terpisah.' : nCukim > 1 ? 'Cukim dijual satu per satu.' : ''
  const untungPreview = Number(hargaJual || 0) - modalPilih

  function toggle(id) {
    setPilih((p) => {
      const n = { ...p }
      if (n[id]) delete n[id]
      else n[id] = true
      return n
    })
  }

  async function jual(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setLoading(true)
    try {
      const res = await api('/penjualan', {
        method: 'POST',
        body: JSON.stringify({
          item_ids: terpilih.map((r) => r.id),
          penadah_name: penadah,
          sale_total: Number(hargaJual),
          sale_date: jualTgl,
          note: jualNote,
        }),
      })
      setInfo(`${res.message} (untung atau rugi ${rupiah(res.untung)})`)
      setPilih({})
      setPenadah('')
      setHargaJual('')
      setJualNote('')
      muat()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function batalJual() {
    if (!alasanSale.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    setError('')
    try {
      const res = await api(`/penjualan/void/${voidSale}`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasanSale }),
      })
      setInfo(res.message)
      setVoidSale(null)
      setAlasanSale('')
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  const rantai = (b) =>
    [b.source_weight, b.cut_weight, b.leburan_weight, b.cukim_weight]
      .filter((x) => x !== null && x !== undefined)
      .map((x) => gram(x))
      .join(' → ')

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}

      <div className="card">
        <h3>Proses Lebur</h3>
        <p className="muted">
          Owner memulai lebur dari menu Stok (centang perhiasan di back office, klik Lebur). PJ mencatat:
          berat setelah dipotong dan leburan kotor, lalu cukim. Cukim dijual ke penadah di bagian bawah.
          Angka berat di tabel: awal → setelah dipotong → leburan kotor → cukim.
        </p>

        <div className="form-grid">
          <div>
            <label>Tampilkan</label>
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="berjalan">Sedang berjalan</option>
              <option value="terjual">Sudah terjual</option>
              <option value="semua">Semua</option>
            </select>
          </div>
        </div>

        {aksi && batchAksi && (
          <form className="void-box" onSubmit={kirimAksi}>
            {aksi.jenis === 'leburan' && (
              <>
                <div>
                  <b>Lebur #{aksi.id}: catat leburan kotor.</b> Berat awal {gram(batchAksi.source_weight)} g. Isi
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
                  <b>Lebur #{aksi.id}: catat cukim.</b> Berat leburan kotor {gram(batchAksi.leburan_weight)} g. Isi
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
                      {KADAR.map((n) => (
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
            {aksi.jenis === 'undo' && (
              <>
                <div>
                  <b>Batalkan langkah terakhir lebur #{aksi.id}.</b> Tahap kembali satu langkah. Tulis alasannya:
                </div>
                <input value={alasan} onChange={(e) => setAlasan(e.target.value)} placeholder="Contoh: salah input berat" autoFocus />
              </>
            )}
            <div className="void-actions">
              <button className={aksi.jenis === 'undo' ? 'mini danger-fill' : 'mini primary-fill'} disabled={loading}>
                {loading ? 'Memproses...' : aksi.jenis === 'undo' ? 'Ya, batalkan' : 'Simpan'}
              </button>
              <button type="button" className="mini" onClick={() => setAksi(null)}>Tidak jadi</button>
            </div>
          </form>
        )}

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Lebur</th>
                <th>Barang</th>
                <th>Berat (g)</th>
                <th>Modal</th>
                <th>Tahap</th>
                <th>Penjualan</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {batches.length === 0 && (
                <tr><td colSpan="7" className="muted">Belum ada data</td></tr>
              )}
              {batches.map((b) => (
                <tr key={b.id}>
                  <td>
                    #{b.id}
                    <div className="muted">{b.start_date.slice(0, 10)}</div>
                  </td>
                  <td style={{ whiteSpace: 'normal', minWidth: 220 }}>
                    {b.item_count} barang
                    <div className="muted">{b.barang}</div>
                    {b.note && <div className="muted">{b.note}</div>}
                  </td>
                  <td>
                    {rantai(b)}
                    {b.susut !== null && b.susut !== undefined && (
                      <div className="muted">susut {gram(b.susut)} g</div>
                    )}
                  </td>
                  <td>{b.modal > 0 ? rupiah(b.modal) : '-'}</td>
                  <td>
                    <span className={TAHAP[b.status][1]}>{TAHAP[b.status][0]}</span>
                    {b.leburan_by && <div className="muted">leburan: {b.leburan_by}</div>}
                    {b.cukim_by && <div className="muted">cukim: {b.cukim_by}</div>}
                  </td>
                  <td>
                    {b.sale_total !== null && b.sale_total !== undefined ? (
                      <>
                        {rupiah(b.sale_total)}
                        <div className="muted">{b.penadah}</div>
                        <div className={b.untung < 0 ? 'neg' : ''}>
                          <b>{b.untung < 0 ? 'Rugi' : 'Untung'} {rupiah(Math.abs(b.untung))}</b>
                        </div>
                      </>
                    ) : (
                      '-'
                    )}
                  </td>
                  <td>
                    {bisaCatat && b.status === 'proses' && (
                      <button type="button" className="mini primary" onClick={() => bukaAksi('leburan', b)}>
                        Catat leburan kotor
                      </button>
                    )}
                    {bisaCatat && b.status === 'leburan_kotor' && (
                      <button type="button" className="mini primary" onClick={() => bukaAksi('cukim', b)}>
                        Catat cukim
                      </button>
                    )}{' '}
                    {bisaUndo && b.status !== 'terjual' && (
                      <button type="button" className="mini" onClick={() => bukaAksi('undo', b)}>
                        Batalkan langkah
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Siap Dijual ke Penadah</h3>
        <p className="muted">
          Berisi cukim dan logam mulia di back office. Satu penjualan berisi satu cukim, atau beberapa
          logam mulia. Uang masuk kas back office.
        </p>

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                {bisaCatat && <th></th>}
                <th>No</th>
                <th>Barang</th>
                <th>Berat</th>
                <th>Modal</th>
                <th>Asal</th>
              </tr>
            </thead>
            <tbody>
              {siap.length === 0 && (
                <tr><td colSpan="6" className="muted">Belum ada barang siap jual</td></tr>
              )}
              {siap.map((r) => (
                <tr key={r.id}>
                  {bisaCatat && (
                    <td>
                      <input type="checkbox" checked={!!pilih[r.id]} onChange={() => toggle(r.id)} />
                    </td>
                  )}
                  <td>{r.item_no}</td>
                  <td>{r.item_name} {r.karat}</td>
                  <td>{r.weight_gram} g</td>
                  <td>{r.harga_beli > 0 ? rupiah(r.harga_beli) : '-'}</td>
                  <td>{r.batch_id ? `Lebur #${r.batch_id}` : 'Beli dari customer'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {bisaCatat && terpilih.length > 0 && (
          <form className="panel-aksi" onSubmit={jual}>
            <div>
              <b>{terpilih.length}</b> barang dipilih ({gram(beratPilih)} g), modal {rupiah(modalPilih)}
            </div>
            {salahPilih && <div className="error">{salahPilih}</div>}
            <div className="form-grid">
              <div>
                <label>Nama penadah</label>
                <input value={penadah} onChange={(e) => setPenadah(e.target.value)} required />
              </div>
              <div>
                <label>Harga jual (Rp)</label>
                <input type="number" min="1" value={hargaJual} onChange={(e) => setHargaJual(e.target.value)} required />
              </div>
              <div>
                <label>Tanggal</label>
                <input type="date" value={jualTgl} onChange={(e) => setJualTgl(e.target.value)} required />
              </div>
              <div>
                <label>Catatan</label>
                <input value={jualNote} onChange={(e) => setJualNote(e.target.value)} />
              </div>
            </div>
            {hargaJual && (
              <p className={untungPreview < 0 ? 'neg' : 'muted'}>
                {untungPreview < 0 ? 'Rugi' : 'Untung'}: <b>{rupiah(Math.abs(untungPreview))}</b> (harga jual dikurangi modal)
              </p>
            )}
            <button disabled={loading || !!salahPilih}>{loading ? 'Memproses...' : 'Jual ke penadah'}</button>
          </form>
        )}
      </div>

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Riwayat Penjualan ke Penadah</h3>
        <div className="form-grid">
          <div>
            <label>Dari tanggal</label>
            <input type="date" value={dari} onChange={(e) => setDari(e.target.value)} />
          </div>
          <div>
            <label>Sampai tanggal</label>
            <input type="date" value={sampai} onChange={(e) => setSampai(e.target.value)} />
          </div>
        </div>

        <div className="grid" style={{ margin: '16px 0' }}>
          <div>
            <div className="label">Total modal</div>
            <div className="big">{rupiah(penj.total_modal)}</div>
          </div>
          <div>
            <div className="label">Total harga jual</div>
            <div className="big">{rupiah(penj.total_jual)}</div>
          </div>
          <div>
            <div className="label">{penj.total_untung < 0 ? 'Rugi' : 'Untung'}</div>
            <div className={penj.total_untung < 0 ? 'big neg' : 'big'}>{rupiah(Math.abs(penj.total_untung))}</div>
          </div>
        </div>

        {voidSale && (
          <div className="void-box">
            <div>Batalkan penjualan #{voidSale}. Barang dan kas dikembalikan. Tulis alasannya:</div>
            <input value={alasanSale} onChange={(e) => setAlasanSale(e.target.value)} autoFocus />
            <div className="void-actions">
              <button type="button" className="mini danger-fill" onClick={batalJual}>Ya, batalkan</button>
              <button type="button" className="mini" onClick={() => setVoidSale(null)}>Tidak jadi</button>
            </div>
          </div>
        )}

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Tanggal</th>
                <th>Penadah</th>
                <th>Barang</th>
                <th>Modal</th>
                <th>Harga jual</th>
                <th>Untung / rugi</th>
                <th>PJ</th>
                {bisaUndo && <th></th>}
              </tr>
            </thead>
            <tbody>
              {penj.rows.length === 0 && (
                <tr><td colSpan="8" className="muted">Belum ada penjualan</td></tr>
              )}
              {penj.rows.map((r) => (
                <tr key={r.id}>
                  <td>{r.sale_date.slice(0, 10)}</td>
                  <td>{r.penadah_name}</td>
                  <td style={{ whiteSpace: 'normal', minWidth: 200 }}>
                    {r.barang}
                    {r.note && <div className="muted">{r.note}</div>}
                  </td>
                  <td>{rupiah(r.modal)}</td>
                  <td>{rupiah(r.sale_total)}</td>
                  <td className={r.untung < 0 ? 'neg' : ''}>
                    <b>{r.untung < 0 ? 'Rugi ' : 'Untung '}{rupiah(Math.abs(r.untung))}</b>
                  </td>
                  <td>{r.pj}</td>
                  {bisaUndo && (
                    <td>
                      <button
                        type="button"
                        className="mini"
                        onClick={() => {
                          setVoidSale(r.id)
                          setAlasanSale('')
                          setError('')
                          setInfo('')
                        }}
                      >
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