import { useState, useEffect } from 'react'
import { api, unduh } from '../api'

const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)

const gram = (n) => Number(Number(n).toFixed(3))
const hariIni = () => new Date().toLocaleDateString('en-CA')
const awalBulan = () => {
  const d = new Date()
  return new Date(d.getFullYear(), d.getMonth(), 1).toLocaleDateString('en-CA')
}

const LAPORAN = [
  ['kas', 'Kas (uang masuk dan keluar)', 'Keuangan'],
  ['labarugi', 'Laba rugi', 'Keuangan'],
  ['hutang', 'Hutang (posisi saat ini)', 'Keuangan'],
  ['penjualan', 'Penjualan ke penadah (untung atau rugi)', 'Keuangan'],
  ['cuci', 'Cuci', 'Operasional'],
  ['lebur', 'Lebur', 'Operasional'],
  ['stok', 'Riwayat stok', 'Operasional'],
]

const tampil = (v, tipe) => {
  if (v === null || v === undefined || v === '') return ''
  if (tipe === 'rupiah') return rupiah(v)
  if (tipe === 'gram') return String(gram(v))
  return String(v)
}

export default function Laporan({ me }) {
  const [dari, setDari] = useState(awalBulan())
  const [sampai, setSampai] = useState(hariIni())
  const [locId, setLocId] = useState('0')
  const [locations, setLocations] = useState([])
  const [pilih, setPilih] = useState(() => Object.fromEntries(LAPORAN.map(([k]) => [k, true])))
  const [hasil, setHasil] = useState(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api('/locations').then(setLocations).catch((e) => setError(e.message))
  }, [])

  const jenis = LAPORAN.filter(([k]) => pilih[k]).map(([k]) => k).join(',')
  const query = `jenis=${jenis}&from=${dari}&to=${sampai}&location_id=${locId}`

  function toggle(k) {
    setPilih((p) => ({ ...p, [k]: !p[k] }))
    setHasil(null)
  }

  function semua(nilai) {
    setPilih(Object.fromEntries(LAPORAN.map(([k]) => [k, nilai])))
    setHasil(null)
  }

  function cekPilihan() {
    if (!jenis) {
      setError('Pilih minimal satu laporan')
      return false
    }
    return true
  }

  async function ambil() {
    return api(`/laporan/preview?${query}`)
  }

  async function pratinjau() {
    setError('')
    if (!cekPilihan()) return
    setLoading(true)
    try {
      setHasil(await ambil())
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function excel() {
    setError('')
    if (!cekPilihan()) return
    setLoading(true)
    try {
      await unduh(`/laporan/excel?${query}`, `laporan_${dari}_sd_${sampai}.xlsx`)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function cetak() {
    setError('')
    if (!cekPilihan()) return
    setLoading(true)
    try {
      setHasil(await ambil())
      setTimeout(() => window.print(), 500)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const grup = ['Keuangan', 'Operasional']

  return (
    <div>
      <div className="card no-print">
        <h3>Laporan</h3>
        <p className="muted">
          Pilih laporan dan periode, lalu lihat pratinjau, unduh Excel, atau cetak. Untuk PDF, klik Cetak dan
          pilih Simpan sebagai PDF di dialog cetak browser.
        </p>

        <div className="form-grid">
          <div>
            <label>Dari tanggal</label>
            <input type="date" value={dari} onChange={(e) => { setDari(e.target.value); setHasil(null) }} />
          </div>
          <div>
            <label>Sampai tanggal</label>
            <input type="date" value={sampai} onChange={(e) => { setSampai(e.target.value); setHasil(null) }} />
          </div>
          <div>
            <label>Lokasi (untuk Kas dan Riwayat stok)</label>
            <select value={locId} onChange={(e) => { setLocId(e.target.value); setHasil(null) }}>
              <option value="0">Semua lokasi</option>
              {locations.map((l) => (
                <option key={l.id} value={l.id}>{l.name}</option>
              ))}
            </select>
          </div>
        </div>

        {grup.map((g) => (
          <div key={g}>
            <label>{g}</label>
            <div className="cek-grup">
              {LAPORAN.filter(([, , gr]) => gr === g).map(([k, nama]) => (
                <label className="cek-item" key={k}>
                  <input type="checkbox" checked={!!pilih[k]} onChange={() => toggle(k)} />
                  {nama}
                </label>
              ))}
            </div>
          </div>
        ))}

        <div className="aksi-tombol" style={{ marginTop: 14 }}>
          <button type="button" className="mini" onClick={() => semua(true)}>Pilih semua</button>
          <button type="button" className="mini" onClick={() => semua(false)}>Kosongkan</button>
        </div>

        {error && <div className="error">{error}</div>}

        <div className="aksi-tombol">
          <button type="button" className="mini primary" disabled={loading} onClick={pratinjau}>Pratinjau</button>
          <button type="button" className="mini primary" disabled={loading} onClick={excel}>Unduh Excel</button>
          <button type="button" className="mini primary" disabled={loading} onClick={cetak}>Cetak / PDF</button>
        </div>
      </div>

      {hasil && (
        <div className="cetak-area" style={{ marginTop: 16 }}>
          <div className="card cetak-judul">
            <h2 style={{ margin: 0 }}>Pembukuan Toko Emas - Bulan Purnama</h2>
            <div className="muted">
              Periode {hasil.dari} sampai {hasil.sampai}. Dicetak oleh {me.username} pada {hariIni()}.
            </div>
          </div>

          {hasil.tabel.map((t) => (
            <div className="card cetak-blok" key={t.kode} style={{ marginTop: 16 }}>
              <h3>{t.judul}</h3>
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      {t.kolom.map((k, j) => (
                        <th key={j} className={t.tipe[j] === 'teks' ? '' : 'kanan'}>{k}</th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {(t.baris || []).map((b, i) => (
                      <tr key={i} className={(t.tebal || []).includes(i) ? 'baris-tebal' : ''}>
                        {b.map((v, j) => (
                          <td key={j} className={t.tipe[j] === 'teks' ? '' : 'kanan'}>{tampil(v, t.tipe[j])}</td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}