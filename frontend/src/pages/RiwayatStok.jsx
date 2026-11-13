import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { gram, hariIni, awalBulan } from '../utils/format'
import { labelKategori } from '../constants/kategori'
import Pesan from '../components/common/Pesan'
import BarisKosong from '../components/common/BarisKosong'

const KOSONG = { dari: '', sampai: '', arah: '', kelompok: '', q: '' }
const KELOMPOK = [
  ['oper', 'Oper dan kiriman (toko, back office, BP)'],
  ['masuk', 'Barang masuk'],
  ['jual', 'Penjualan'],
  ['olah', 'Cuci dan lebur'],
]
const fmt = (d) => d.toLocaleDateString('en-CA')
const PERIODE = [
  ['Hari ini', () => [hariIni(), hariIni()]],
  ['7 hari', () => [fmt(new Date(Date.now() - 6 * 86400000)), hariIni()]],
  ['Bulan ini', () => [awalBulan(), hariIni()]],
  ['Bulan lalu', () => {
    const n = new Date()
    return [fmt(new Date(n.getFullYear(), n.getMonth() - 1, 1)), fmt(new Date(n.getFullYear(), n.getMonth(), 0))]
  }],
]

// Riwayat pergerakan stok (oper, barang masuk, jual, cuci, lebur) dengan filter.
// Akun cabang hanya melihat tokonya sendiri; owner bisa memilih lokasi.
export default function RiwayatStok({ me }) {
  const isCabang = me.role === 'cabang'
  const [locations, setLocations] = useState([])
  const [locId, setLocId] = useState(isCabang ? String(me.location_id) : '0')
  const [filter, setFilter] = useState(KOSONG)
  const [cari, setCari] = useState('')
  const [limit, setLimit] = useState(200)
  const [data, setData] = useState({ rows: [], masuk_pcs: 0, keluar_pcs: 0, masuk_gram: 0, keluar_gram: 0, ada_lagi: false })
  const [error, setError] = useState('')

  useEffect(() => {
    api('/locations').then(setLocations).catch((e) => setError(e.message))
  }, [])

  // pencarian ditunda sebentar supaya tidak memanggil server di tiap ketukan
  useEffect(() => {
    const t = setTimeout(() => {
      setFilter((f) => (f.q === cari ? f : { ...f, q: cari }))
      setLimit(200)
    }, 350)
    return () => clearTimeout(t)
  }, [cari])

  const ubah = useCallback((bagian) => {
    setFilter((f) => ({ ...f, ...bagian }))
    setLimit(200)
  }, [])

  useEffect(() => {
    const p = new URLSearchParams({ location_id: locId, limit })
    Object.entries({ from: filter.dari, to: filter.sampai, arah: filter.arah, kelompok: filter.kelompok, q: filter.q }).forEach(
      ([k, v]) => v && p.set(k, v),
    )
    api(`/stok/riwayat?${p}`).then(setData).catch((e) => setError(e.message))
  }, [locId, filter, limit])

  const aktif = JSON.stringify(filter) !== JSON.stringify(KOSONG)

  return (
    <div>
      <Pesan error={error} />
      <div className="card">
        <h3>Riwayat Pergerakan Stok</h3>
        <div className="panel-aksi" style={{ marginBottom: 16 }}>
          <div className="form-grid">
            {!isCabang && (
              <div>
                <label>Lokasi</label>
                <select value={locId} onChange={(e) => { setLocId(e.target.value); setLimit(200) }}>
                  <option value="0">Semua lokasi</option>
                  {locations.map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
                </select>
              </div>
            )}
            <div>
              <label>Dari tanggal</label>
              <input type="date" value={filter.dari} onChange={(e) => ubah({ dari: e.target.value })} />
            </div>
            <div>
              <label>Sampai tanggal</label>
              <input type="date" value={filter.sampai} onChange={(e) => ubah({ sampai: e.target.value })} />
            </div>
            <div>
              <label>Arus</label>
              <select value={filter.arah} onChange={(e) => ubah({ arah: e.target.value })}>
                <option value="">Semua</option>
                <option value="in">Masuk</option>
                <option value="out">Keluar</option>
              </select>
            </div>
            <div>
              <label>Jenis</label>
              <select value={filter.kelompok} onChange={(e) => ubah({ kelompok: e.target.value })}>
                <option value="">Semua</option>
                {KELOMPOK.map(([k, l]) => <option key={k} value={k}>{l}</option>)}
              </select>
            </div>
            <div>
              <label>Cari nama barang, nomor, atau catatan</label>
              <input value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Contoh: Cincin, BP Pecinan" />
            </div>
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', marginTop: 12 }}>
            {PERIODE.map(([label, hitung]) => (
              <button key={label} type="button" className="mini" onClick={() => { const [d, s] = hitung(); ubah({ dari: d, sampai: s }) }}>
                {label}
              </button>
            ))}
            {aktif && <button type="button" className="mini" onClick={() => { setFilter(KOSONG); setCari(''); setLimit(200) }}>Reset filter</button>}
          </div>
          <div className="total-bar">
            <span>Masuk: <b className="utama">{data.masuk_pcs}</b> pcs, {gram(data.masuk_gram)} g</span>
            <span>Keluar: <b className="utama">{data.keluar_pcs}</b> pcs, {gram(data.keluar_gram)} g</span>
          </div>
        </div>

        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Tanggal</th>
                <th>Lokasi</th>
                <th>Kejadian</th>
                <th>Barang</th>
                <th className="num">Berat</th>
                <th>Catatan</th>
                <th>Oleh</th>
              </tr>
            </thead>
            <tbody>
              {data.rows.length === 0 && <BarisKosong kolom="7">Belum ada data</BarisKosong>}
              {data.rows.map((r) => (
                <tr key={r.id}>
                  <td>{r.transaction_date}</td>
                  <td>{r.lokasi}</td>
                  <td>
                    <span className={r.direction === 'in' ? 'tag in' : 'tag out'}>{r.direction === 'in' ? 'Masuk' : 'Keluar'}</span>{' '}
                    {labelKategori(r.category)}
                  </td>
                  <td>
                    {r.item_no && <span className="muted">{r.item_no} </span>}
                    {r.item_name} {r.karat}
                    {r.qty > 1 && <span className="muted"> ({r.qty} pcs)</span>}
                  </td>
                  <td className="num">{r.weight_gram > 0 ? `${gram(r.weight_gram)} g` : '-'}</td>
                  <td style={{ whiteSpace: 'normal', minWidth: 160 }}>{r.note}</td>
                  <td>{r.oleh}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {data.ada_lagi && (
          <button type="button" className="ghost" style={{ marginTop: 12 }} onClick={() => setLimit((l) => l + 200)}>
            Muat lebih banyak
          </button>
        )}
      </div>
    </div>
  )
}
