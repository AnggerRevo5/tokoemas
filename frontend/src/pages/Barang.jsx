import { useState, useEffect, useCallback, useRef } from 'react'
import { api } from '../api'
import { KATEGORI } from '../kategori'

const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)

const hariIni = () => new Date().toLocaleDateString('en-CA')
const gram = (n) => Number(Number(n).toFixed(3))

const NAMA_UMUM = ['Cincin', 'Kalung', 'Gelang', 'Anting', 'Liontin', 'Giwang']
const MEREK_LM = ['Antam', 'UBS', 'Galeri24', 'Lotus Archi']
const KARAT_UMUM = [24, 22, 21, 20, 18, 17, 16, 9, 8]
const KARAT_LAIN = Array.from({ length: 24 }, (_, i) => 24 - i).filter((n) => !KARAT_UMUM.includes(n))

const STATUS_LABEL = {
  tersedia: 'Tersedia',
  dikirim: 'Dalam perjalanan',
  cuci: 'Proses cuci',
  lebur: 'Dilebur',
  hasil: 'Hasil lebur',
  kembali: 'Dikembalikan ke back office',
  terjual: 'Terjual',
}
const STATUS_CLASS = {
  tersedia: 'tag in',
  dikirim: 'tag warn',
  cuci: 'tag warn',
  lebur: 'tag gray',
  hasil: 'tag gray',
  kembali: 'tag warn',
  terjual: 'tag gray',
}

const bisaPilih = (r) => r.status === 'tersedia' || r.status === 'dikirim' || r.status === 'cuci' || r.status === 'kembali'

const barisBaru = (dari) => ({
  jenis: dari ? dari.jenis : 'perhiasan',
  item_name: dari ? dari.item_name : '',
  brand: dari ? dari.brand : '',
  karat: dari ? dari.karat : '',
  weight_gram: '',
  harga_beli: '',
  note: '',
})

export default function Barang({ me, onUbah }) {
  const isCabang = me.role === 'cabang'
  const bisaVoid = me.role === 'super_admin'

  const [locations, setLocations] = useState([])
  const [locId, setLocId] = useState(isCabang ? String(me.location_id) : '')
  const [status, setStatus] = useState('tersedia')
  const [jenisF, setJenisF] = useState('semua')
  const [cari, setCari] = useState('')
  const [data, setData] = useState({ rows: [], jumlah: 0, total_gram: 0, total_beli: 0 })
  const [notif, setNotif] = useState({ menunggu: 0, belum_sampai: 0, keluar: 0 })
  const [pilih, setPilih] = useState({})
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [loading, setLoading] = useState(false)

  const [formBuka, setFormBuka] = useState(false)
  const [fLoc, setFLoc] = useState(isCabang ? String(me.location_id) : '')
  const [fKat, setFKat] = useState('')
  const [fTgl, setFTgl] = useState(hariIni())
  const [fNote, setFNote] = useState('')
  const [fPosisi, setFPosisi] = useState('toko')
  const [baris, setBaris] = useState([barisBaru()])
  const [fokus, setFokus] = useState(false)
  const wadah = useRef(null)

  const [aksi, setAksi] = useState(null)
  const [tgl, setTgl] = useState(hariIni())
  const [target, setTarget] = useState('')
  const [catatan, setCatatan] = useState('')

  const [voidId, setVoidId] = useState(null)
  const [alasan, setAlasan] = useState('')

  const tipeLok = (id) => (locations.find((l) => l.id === Number(id)) || {}).type
  const cabangList = locations.filter((l) => l.type === 'cabang')

  useEffect(() => {
    api('/locations')
      .then((d) => {
        setLocations(d)
        const b = d.find((l) => l.type === 'back_office')
        const c = d.find((l) => l.type === 'cabang')
        if (!isCabang && b) {
          setLocId(String(b.id))
          setFLoc(String(b.id))
        }
        if (c) setTarget(String(c.id))
      })
      .catch((e) => setError(e.message))
  }, [])

  const fTipe = tipeLok(fLoc) || 'cabang'
  const fOpsi = KATEGORI.stok[fTipe].in
  const wajibHarga = fKat === 'beli_customer'

  useEffect(() => {
    if (fOpsi.length) setFKat(fOpsi[0][0])
  }, [fTipe])

  const muat = useCallback(() => {
    if (locId === '') return
    const p = new URLSearchParams({ location_id: locId, status })
    if (jenisF !== 'semua') p.set('jenis', jenisF)
    if (cari.trim()) p.set('q', cari.trim())
    api(`/barang?${p.toString()}`)
      .then((d) => {
        setData(d)
        setPilih((lama) => {
          const baru = {}
          d.rows.forEach((r) => {
            if (lama[r.id] && bisaPilih(r)) baru[r.id] = true
          })
          return baru
        })
      })
      .catch((e) => setError(e.message))
    api('/barang/notif').then(setNotif).catch(() => {})
  }, [locId, status, jenisF, cari])

  useEffect(() => {
    muat()
  }, [muat])

  useEffect(() => {
    if (fokus && wadah.current) {
      const el = wadah.current.querySelectorAll('.in-berat')
      if (el.length) el[el.length - 1].focus()
      setFokus(false)
    }
  }, [baris, fokus])

  const bisaDipilih = data.rows.filter(bisaPilih)
  const terpilih = data.rows.filter((r) => pilih[r.id])
  const beratTerpilih = terpilih.reduce((s, r) => s + r.weight_gram, 0)
  const adaLM = terpilih.some((r) => r.jenis === 'logam_mulia')
  const statusSet = new Set(terpilih.map((r) => r.status))
  const tipeTerpilih = new Set(terpilih.map((r) => tipeLok(r.location_id)))

  let mode = null
  if (terpilih.length > 0) {
    if (statusSet.size > 1) {
      mode = 'campur'
    } else if (statusSet.has('dikirim')) {
      const keToko = terpilih.filter((r) => r.tujuan_id > 0).length
      if (keToko === terpilih.length) mode = 'transit_toko'
      else if (keToko === 0) mode = 'transit'
      else mode = 'campur'
    } else if (statusSet.has('kembali')) {
      mode = 'kembali'
    } else if (statusSet.has('cuci')) {
      mode = 'cuci'
    } else {
      mode = tipeTerpilih.size > 1 ? 'campur' : [...tipeTerpilih][0]
    }
  }

  let aksiTersedia = []
  if (mode === 'transit' && !isCabang) {
    aksiTersedia = [['sampai', 'Sudah sampai'], ['belum', 'Belum sampai']]
  } else if (mode === 'transit_toko') {
    aksiTersedia = [['terima_toko', 'Sudah diterima'], ['belum_toko', 'Belum diterima']]
  } else if (mode === 'kembali' && !isCabang) {
    aksiTersedia = [['terima_kembali', 'Sudah diterima'], ['belum_kembali', 'Belum diterima']]
  } else if (mode === 'cuci' && !isCabang) {
    aksiTersedia = []
  } else if (mode === 'back_office' && !isCabang) {
    aksiTersedia = adaLM
      ? [['kirim', 'Jual (oper ke toko)']]
      : [['kirim', 'Jual (oper ke toko)'], ['cuci', 'Cuci'], ['lebur', 'Lebur']]
  } else if (mode === 'cabang') {
    aksiTersedia = [['antar', 'Antar ke back office'], ['jual', 'Jual']]
  }

  useEffect(() => {
    if (aksi && !aksiTersedia.some(([k]) => k === aksi)) setAksi(null)
  }, [mode, adaLM])

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

  function ubahBaris(i, k, v) {
    setBaris((b) => b.map((x, idx) => (idx === i ? { ...x, [k]: v } : x)))
  }
  function tambahBaris() {
    setBaris((b) => [...b, barisBaru(b[b.length - 1])])
  }
  function hapusBaris(i) {
    setBaris((b) => (b.length === 1 ? b : b.filter((_, idx) => idx !== i)))
  }
  function enterBaris(e, dariHarga) {
    if (e.key !== 'Enter') return
    e.preventDefault()
    if (!dariHarga && wajibHarga) {
      const h = e.currentTarget.parentElement.querySelector('.in-harga')
      if (h) h.focus()
      return
    }
    tambahBaris()
    setFokus(true)
  }

  const totalGram = baris.reduce((s, b) => s + Number(b.weight_gram || 0), 0)
  const totalBeli = baris.reduce((s, b) => s + Number(b.harga_beli || 0), 0)

  function selesai() {
    muat()
    if (onUbah) onUbah()
  }

  async function simpanMasuk(e) {
    e.preventDefault()
    setError('')
    setInfo('')
    setLoading(true)
    try {
      const res = await api('/barang', {
        method: 'POST',
        body: JSON.stringify({
          location_id: Number(fLoc),
          category: fKat,
          entry_date: fTgl,
          note: fNote,
          posisi: wajibHarga ? fPosisi : '',
          items: baris.map((b) => ({
            jenis: b.jenis,
            item_name: b.item_name,
            brand: b.brand,
            karat: b.karat,
            weight_gram: Number(b.weight_gram),
            harga_beli: Number(b.harga_beli || 0),
            note: b.note,
          })),
        }),
      })
      setInfo(res.message + (res.kas_keluar > 0 ? ` - kas keluar ${rupiah(res.kas_keluar)}` : ''))
      setBaris([barisBaru()])
      setFNote('')
      setFPosisi('toko')
      setFormBuka(false)
      if (!isCabang) setLocId(fLoc)
      setStatus(fPosisi === 'bo' && wajibHarga ? 'dikirim' : 'tersedia')
      selesai()
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
    const ids = terpilih.map((r) => r.id)
    try {
      let res
      if (aksi === 'kirim') {
        res = await api('/barang/oper', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, target_location_id: Number(target), date: tgl }),
        })
      } else if (aksi === 'terima_toko' || aksi === 'belum_toko') {
        res = await api('/barang/terima-toko', {
          method: 'POST',
          body: JSON.stringify({
            item_ids: ids,
            result: aksi === 'terima_toko' ? 'sampai' : 'belum',
            date: tgl,
            note: catatan,
          }),
        })
      } else if (aksi === 'antar') {
        res = await api('/barang/kirim-bo', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, date: tgl }),
        })
      } else if (aksi === 'sampai') {
        res = await api('/barang/terima', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, result: 'sampai', date: tgl }),
        })
      } else if (aksi === 'belum') {
        res = await api('/barang/terima', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, result: 'belum', date: tgl, note: catatan }),
        })
      } else if (aksi === 'terima_kembali' || aksi === 'belum_kembali') {
        res = await api('/barang/terima-kembali', {
          method: 'POST',
          body: JSON.stringify({
            item_ids: ids,
            result: aksi === 'terima_kembali' ? 'sampai' : 'belum',
            date: tgl,
            note: catatan,
          }),
        })
      } else if (aksi === 'cuci') {
        res = await api('/cuci', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, start_date: tgl }),
        })
      } else if (aksi === 'lebur') {
        res = await api('/lebur', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, start_date: tgl, note: catatan }),
        })
      } else {
        res = await api('/barang/jual', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, date: tgl }),
        })
      }
      setInfo(res.message)
      setAksi(null)
      setPilih({})
      setCatatan('')
      selesai()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  function mulaiBatal(id) {
    setVoidId(id)
    setAlasan('')
    setError('')
    setInfo('')
  }

  async function konfirmasiBatal() {
    if (!alasan.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    setError('')
    try {
      const res = await api(`/barang/void/${voidId}`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasan }),
      })
      setInfo(res.message)
      setVoidId(null)
      selesai()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}

      {(notif.menunggu > 0 || notif.keluar > 0) && (
        <div className="banner">
          <div>
            {isCabang ? (
              <>
                {notif.menunggu > 0 && (
                  <div>
                    <b>{notif.menunggu}</b> barang dioper ke toko ini, menunggu konfirmasi kamu
                    {notif.belum_sampai > 0 && <> ({notif.belum_sampai} ditandai belum diterima)</>}
                  </div>
                )}
                {notif.keluar > 0 && (
                  <div>
                    <b>{notif.keluar}</b> barang sudah diantar, menunggu konfirmasi back office
                  </div>
                )}
              </>
            ) : (
              <>
                {notif.menunggu > 0 && (
                  <div>
                    <b>{notif.menunggu}</b> barang dari toko menunggu konfirmasi
                    {notif.belum_sampai > 0 && <> ({notif.belum_sampai} ditandai belum sampai)</>}
                  </div>
                )}
                {notif.keluar > 0 && (
                  <div>
                    <b>{notif.keluar}</b> barang dioper ke toko, menunggu konfirmasi toko tujuan
                  </div>
                )}
              </>
            )}
          </div>
          <button
            type="button"
            className="mini primary"
            onClick={() => {
              if (!isCabang) setLocId('0')
              setStatus('dikirim')
            }}
          >
            Lihat
          </button>
        </div>
      )}

      <div style={{ marginBottom: 16 }}>
        <button type="button" className="ghost" onClick={() => setFormBuka(!formBuka)}>
          {formBuka ? 'Tutup form' : '+ Tambah barang masuk'}
        </button>
      </div>

      {formBuka && (
        <form
          className="card"
          style={{ marginBottom: 16 }}
          onSubmit={simpanMasuk}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && e.target.tagName === 'INPUT') e.preventDefault()
          }}
        >
          <h3>Tambah Barang Masuk</h3>
          <p className="muted">
            Satu baris = satu barang. Tekan Enter di kolom terakhir untuk menambah baris baru
            (jenis, nama, dan kadar terisi otomatis).
          </p>

          <div className="form-grid">
            {!isCabang && (
              <div>
                <label>Lokasi</label>
                <select value={fLoc} onChange={(e) => setFLoc(e.target.value)}>
                  {locations.map((l) => (
                    <option key={l.id} value={l.id}>{l.name}</option>
                  ))}
                </select>
              </div>
            )}
            <div>
              <label>Asal barang</label>
              <select value={fKat} onChange={(e) => setFKat(e.target.value)}>
                {fOpsi.map(([k, l]) => (
                  <option key={k} value={k}>{l}</option>
                ))}
              </select>
            </div>
            {wajibHarga && (
              <div>
                <label>Posisi barang</label>
                <select value={fPosisi} onChange={(e) => setFPosisi(e.target.value)}>
                  <option value="toko">Tetap di toko</option>
                  <option value="bo">Antar ke back office</option>
                </select>
              </div>
            )}
            <div>
              <label>Tanggal</label>
              <input type="date" value={fTgl} onChange={(e) => setFTgl(e.target.value)} required />
            </div>
            <div>
              <label>Catatan untuk semua barang</label>
              <input value={fNote} onChange={(e) => setFNote(e.target.value)} />
            </div>
          </div>

          {wajibHarga && (
            <p className="muted">
              Harga beli wajib diisi. Kas toko otomatis keluar sebesar total harga beli.
              {fPosisi === 'bo' && ' Barang berstatus dalam perjalanan sampai back office mengonfirmasi.'}
            </p>
          )}

          <datalist id="nama-umum">
            {NAMA_UMUM.map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>
          <datalist id="merek-lm">
            {MEREK_LM.map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>

          <div ref={wadah}>
            <div className="baris-grid baris-head">
              <span>Jenis</span>
              <span>Nama / merek</span>
              <span>Kadar</span>
              <span>Berat (gram)</span>
              <span>Harga beli (Rp)</span>
              <span>Keterangan</span>
              <span></span>
            </div>
            {baris.map((b, i) => (
              <div className="baris-grid" key={i}>
                <select value={b.jenis} onChange={(e) => ubahBaris(i, 'jenis', e.target.value)}>
                  <option value="perhiasan">Perhiasan</option>
                  <option value="logam_mulia">Logam mulia</option>
                </select>

                {b.jenis === 'logam_mulia' ? (
                  <input
                    list="merek-lm"
                    value={b.brand}
                    onChange={(e) => ubahBaris(i, 'brand', e.target.value)}
                    placeholder="Merek (Antam...)"
                    required
                  />
                ) : (
                  <input
                    list="nama-umum"
                    value={b.item_name}
                    onChange={(e) => ubahBaris(i, 'item_name', e.target.value)}
                    placeholder="Cincin"
                    required
                  />
                )}

                {b.jenis === 'logam_mulia' ? (
                  <input value="24K" disabled />
                ) : (
                  <select value={b.karat} onChange={(e) => ubahBaris(i, 'karat', e.target.value)} required>
                    <option value="">Kadar</option>
                    <optgroup label="Umum">
                      {KARAT_UMUM.map((n) => (
                        <option key={n} value={`${n}K`}>{n}K</option>
                      ))}
                    </optgroup>
                    <optgroup label="Lainnya">
                      {KARAT_LAIN.map((n) => (
                        <option key={n} value={`${n}K`}>{n}K</option>
                      ))}
                    </optgroup>
                  </select>
                )}

                <input
                  className="in-berat"
                  type="number"
                  step="0.001"
                  min="0.001"
                  value={b.weight_gram}
                  onChange={(e) => ubahBaris(i, 'weight_gram', e.target.value)}
                  onKeyDown={(e) => enterBaris(e, false)}
                  required
                />
                <input
                  className="in-harga"
                  type="number"
                  min="0"
                  value={b.harga_beli}
                  onChange={(e) => ubahBaris(i, 'harga_beli', e.target.value)}
                  onKeyDown={(e) => enterBaris(e, true)}
                  placeholder={wajibHarga ? 'Rp' : 'opsional'}
                  required={wajibHarga}
                />
                <input
                  value={b.note}
                  onChange={(e) => ubahBaris(i, 'note', e.target.value)}
                  placeholder="Polos, ukir..."
                />
                <button type="button" className="mini" onClick={() => hapusBaris(i)}>Hapus</button>
              </div>
            ))}
          </div>

          <button
            type="button"
            className="ghost"
            style={{ marginTop: 10 }}
            onClick={() => {
              tambahBaris()
              setFokus(true)
            }}
          >
            + Tambah baris
          </button>

          <p className="muted">
            Total: {baris.length} barang, {gram(totalGram)} gram
            {totalBeli > 0 && <>, harga beli {rupiah(totalBeli)}</>}
          </p>

          <button disabled={loading}>{loading ? 'Menyimpan...' : 'Simpan semua'}</button>
        </form>
      )}

      <div className="card">
        <div className="form-grid">
          {!isCabang && (
            <div>
              <label>Lokasi</label>
              <select
                value={locId}
                onChange={(e) => {
                  setLocId(e.target.value)
                  setPilih({})
                }}
              >
                <option value="0">Semua lokasi</option>
                {locations.map((l) => (
                  <option key={l.id} value={l.id}>{l.name}</option>
                ))}
              </select>
            </div>
          )}
          <div>
            <label>Status</label>
            <select value={status} onChange={(e) => setStatus(e.target.value)}>
              <option value="tersedia">Tersedia</option>
              <option value="dikirim">Dalam perjalanan</option>
              <option value="cuci">Proses cuci</option>
              <option value="lebur">Dilebur</option>
              <option value="hasil">Hasil lebur</option>
              <option value="terjual">Terjual</option>
              <option value="semua">Semua</option>
            </select>
          </div>
          <div>
            <label>Jenis</label>
            <select value={jenisF} onChange={(e) => setJenisF(e.target.value)}>
              <option value="semua">Semua</option>
              <option value="perhiasan">Perhiasan</option>
              <option value="logam_mulia">Logam mulia</option>
              <option value="hasil_lebur">Hasil lebur</option>
            </select>
          </div>
          <div>
            <label>Cari nama, keterangan, atau pencuci</label>
            <input value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Cincin, Adip..." />
          </div>
        </div>
        <p className="muted" style={{ marginBottom: 0 }}>
          {data.jumlah} barang, {gram(data.total_gram)} gram
          {data.total_beli > 0 && <>, harga beli {rupiah(data.total_beli)}</>}
        </p>
      </div>

      {terpilih.length > 0 && (
        <div className="card action-bar" style={{ marginTop: 16 }}>
          <div>
            <b>{terpilih.length}</b> barang dipilih ({gram(beratTerpilih)} gram)
          </div>
          {mode === 'campur' && (
            <div className="muted">Pilih barang dengan status, lokasi, dan arah kiriman yang sama.</div>
          )}
          {mode === 'transit' && isCabang && (
            <div className="muted">Menunggu konfirmasi dari back office.</div>
          )}
          {adaLM && mode === 'back_office' && (
            <div className="muted">
              Logam mulia hanya bisa dikirim ke toko. Untuk dijual ke penadah, PJ menjualnya lewat menu Lebur.
            </div>
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
            <button
              type="button"
              className="mini"
              onClick={() => {
                setPilih({})
                setAksi(null)
              }}
            >
              Batal pilih
            </button>
          </div>

          {aksi && (
            <form className="panel-aksi" onSubmit={jalankan}>
              <div className="form-grid">
                <div>
                  <label>{aksi === 'selesai_cuci' ? 'Tanggal selesai' : 'Tanggal'}</label>
                  <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
                </div>

                {aksi === 'kirim' && (
                  <div>
                    <label>Oper ke toko</label>
                    <select value={target} onChange={(e) => setTarget(e.target.value)}>
                      {cabangList.map((l) => (
                        <option key={l.id} value={l.id}>{l.name}</option>
                      ))}
                    </select>
                  </div>
                )}

                {(aksi === 'belum' || aksi === 'belum_toko' || aksi === 'belum_kembali' || aksi === 'lebur') && (
                  <div className="span2">
                    <label>Catatan (opsional)</label>
                    <input
                      value={catatan}
                      onChange={(e) => setCatatan(e.target.value)}
                      placeholder={
                        aksi === 'lebur'
                          ? 'Contoh: ada permata, dilepas dulu'
                          : aksi === 'belum_toko'
                            ? 'Contoh: belum ada kurir yang datang'
                            : 'Contoh: masih di kurir'
                      }
                    />
                  </div>
                )}
              </div>

              {aksi === 'antar' && (
                <p className="muted">Barang berstatus dalam perjalanan sampai back office menekan Sudah sampai.</p>
              )}
              {aksi === 'sampai' && <p className="muted">Barang masuk ke stok back office.</p>}
              {aksi === 'belum' && (
                <p className="muted">Barang tetap di daftar dengan label belum sampai.</p>
              )}
              {aksi === 'terima_toko' && (
                <p className="muted">Stok back office berkurang dan stok toko ini bertambah.</p>
              )}
              {aksi === 'belum_toko' && (
                <p className="muted">Barang tetap tercatat di back office dengan label belum diterima.</p>
              )}
              {aksi === 'terima_kembali' && (
                <p className="muted">
                  Barang masuk ke stok back office dengan berat terbaru yang dicatat PJ. Barang hasil cuci
                  berstatus Bersih, lalu bisa dioper ke Kepanjen atau Dinoyo.
                </p>
              )}
              {aksi === 'belum_kembali' && (
                <p className="muted">Barang tetap tercatat dengan label belum diterima.</p>
              )}
              {aksi === 'kirim' && (
                <p className="muted">
                  Barang berstatus dalam perjalanan dan masih tercatat di back office sampai akun toko tujuan
                  menekan Sudah diterima.
                </p>
              )}
              {aksi === 'lebur' && (
                <p className="muted">
                  Stok back office berkurang dan barang masuk proses lebur. PJ lalu mencatat berat setelah
                  dipotong, leburan kotor, dan cukim di menu Lebur. Modal dihitung dari jumlah harga beli.
                </p>
              )}
              {aksi === 'cuci' && (
                <p className="muted">
                  Stok back office berkurang dan barang masuk antrean cuci. PJ yang mengirimnya ke pencuci dan
                  mencatat saat selesai di menu Cuci. Barang yang selesai berstatus Bersih.
                </p>
              )}
              {aksi === 'jual' && <p className="muted">Omzet penjualan dicatat di menu Kas.</p>}

              <button disabled={loading}>{loading ? 'Memproses...' : 'Konfirmasi'}</button>
            </form>
          )}
        </div>
      )}

      <div className="card" style={{ marginTop: 16 }}>
        {voidId && (
          <div className="void-box">
            <div>Batalkan barang #{voidId} (salah input). Tulis alasannya:</div>
            <input
              value={alasan}
              onChange={(e) => setAlasan(e.target.value)}
              placeholder="Contoh: salah input berat"
              autoFocus
            />
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
                <th>
                  <input type="checkbox" checked={semuaDipilih} onChange={toggleSemua} />
                </th>
                <th>No</th>
                <th>Barang</th>
                <th>Berat</th>
                <th>Kondisi</th>
                <th>Harga beli</th>
                <th>Keterangan</th>
                <th>Lokasi</th>
                <th>Asal</th>
                <th>Status</th>
                <th>Masuk</th>
                {bisaVoid && <th></th>}
              </tr>
            </thead>
            <tbody>
              {data.rows.length === 0 && (
                <tr><td colSpan="12" className="muted">Belum ada barang</td></tr>
              )}
              {data.rows.map((r) => (
                <tr key={r.id}>
                  <td>
                    <input
                      type="checkbox"
                      checked={!!pilih[r.id]}
                      disabled={!bisaPilih(r)}
                      onChange={() => toggle(r.id)}
                    />
                  </td>
                  <td>{r.item_no}</td>
                  <td>{r.item_name} {r.karat}</td>
                  <td>{r.weight_gram} g</td>
                  <td>
                    <span className={r.kondisi === 'kotor' ? 'tag warn' : 'tag in'}>
                      {r.kondisi === 'kotor' ? 'Kotor' : 'Bersih'}
                    </span>
                  </td>
                  <td>{r.harga_beli > 0 ? rupiah(r.harga_beli) : '-'}</td>
                  <td>{r.note}</td>
                  <td>{r.lokasi}</td>
                  <td>{r.asal}</td>
                  <td>
                    {r.status === 'dikirim' && r.belum_sampai ? (
                      <span className="tag out">{r.tujuan_id > 0 ? 'Belum diterima' : 'Belum sampai'}</span>
                    ) : (
                      <span className={STATUS_CLASS[r.status]}>{STATUS_LABEL[r.status]}</span>
                    )}
                    {r.status === 'dikirim' && (
                      <div className="muted">
                        {r.tujuan_id > 0 ? `menuju ${r.tujuan}` : 'menuju Back Office'}
                        {r.kirim_tgl && `, ${r.kirim_tgl.slice(0, 10)}`}
                      </div>
                    )}
                    {r.status === 'dikirim' && r.belum_sampai && r.belum_sampai_note && (
                      <div className="muted">{r.belum_sampai_note}</div>
                    )}
                    {r.status === 'cuci' && (
                      <div className="muted">
                        {r.pencuci || 'Menunggu PJ mengirim ke pencuci'}
                        {r.pencuci && r.cuci_tgl && `, sejak ${r.cuci_tgl.slice(0, 10)}`}
                      </div>
                    )}
                    {r.status === 'hasil' && <div className="muted">dikelola di menu Lebur</div>}
                    {r.status === 'kembali' && (
                      <div className="muted">
                        dari PJ, menunggu konfirmasi back office
                        {r.belum_sampai && ' (ditandai belum diterima)'}
                      </div>
                    )}
                  </td>
                  <td>{r.entry_date.slice(0, 10)}</td>
                  {bisaVoid && (
                    <td>
                      {(r.status === 'tersedia' || r.status === 'dikirim') && (
                        <button type="button" className="mini" onClick={() => mulaiBatal(r.id)}>
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