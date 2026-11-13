import { useState, useEffect } from 'react'
import { rupiah, hariIni, awalBulan } from '../../utils/format'

export const FILTER_KAS_KOSONG = { dari: '', sampai: '', arah: '', kelompok: '', metode: '', q: '', batal: false }

const KELOMPOK = [
  ['penjualan', 'Penjualan'],
  ['penadah', 'Jual ke penadah'],
  ['tukar_bp', 'Tukar BP'],
  ['pembelian', 'Pembelian dari customer'],
  ['gadai', 'Gadai'],
  ['hutang', 'Hutang dan investor'],
  ['service', 'Service'],
  ['operasional', 'Pengeluaran dan operasional'],
  ['transfer_kas', 'Transfer dan tambahan kas'],
]

const fmt = (d) => d.toLocaleDateString('en-CA')

// pilihan cepat periode: [label, () => [dari, sampai]]
const PERIODE = [
  ['Hari ini', () => [hariIni(), hariIni()]],
  ['7 hari', () => [fmt(new Date(Date.now() - 6 * 86400000)), hariIni()]],
  ['Bulan ini', () => [awalBulan(), hariIni()]],
  ['Bulan lalu', () => {
    const n = new Date()
    return [fmt(new Date(n.getFullYear(), n.getMonth() - 1, 1)), fmt(new Date(n.getFullYear(), n.getMonth(), 0))]
  }],
]

// Panel filter riwayat kas (terpisah dari form input kas) beserta ringkasan total hasil filter.
// filter: nilai saat ini; onUbah(sebagian): ubah sebagian filter; onReset(); ringkas = { masuk, keluar }.
export default function FilterKas({ filter, onUbah, onReset, ringkas }) {
  const [cari, setCari] = useState(filter.q)

  // pencarian teks ditunda sebentar supaya tidak memanggil server di tiap ketukan
  useEffect(() => {
    const t = setTimeout(() => {
      if (cari !== filter.q) onUbah({ q: cari })
    }, 350)
    return () => clearTimeout(t)
  }, [cari, filter.q, onUbah])

  // filter direset dari luar: kosongkan juga kolom pencarian
  useEffect(() => {
    if (filter.q === '') setCari('')
  }, [filter.q])

  const aktif = JSON.stringify(filter) !== JSON.stringify(FILTER_KAS_KOSONG)

  return (
    <div className="panel-aksi" style={{ marginBottom: 16 }}>
      <div className="form-grid">
        <div>
          <label>Dari tanggal</label>
          <input type="date" value={filter.dari} onChange={(e) => onUbah({ dari: e.target.value })} />
        </div>
        <div>
          <label>Sampai tanggal</label>
          <input type="date" value={filter.sampai} onChange={(e) => onUbah({ sampai: e.target.value })} />
        </div>
        <div>
          <label>Arus</label>
          <select value={filter.arah} onChange={(e) => onUbah({ arah: e.target.value })}>
            <option value="">Semua</option>
            <option value="in">Masuk</option>
            <option value="out">Keluar</option>
          </select>
        </div>
        <div>
          <label>Jenis transaksi</label>
          <select value={filter.kelompok} onChange={(e) => onUbah({ kelompok: e.target.value })}>
            <option value="">Semua</option>
            {KELOMPOK.map(([k, l]) => <option key={k} value={k}>{l}</option>)}
          </select>
        </div>
        <div>
          <label>Metode bayar</label>
          <select value={filter.metode} onChange={(e) => onUbah({ metode: e.target.value })}>
            <option value="">Semua</option>
            <option value="tunai">Tunai</option>
            <option value="transfer">Transfer</option>
            <option value="qris">QRIS/Debit</option>
          </select>
        </div>
        <div>
          <label>Cari catatan</label>
          <input value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Contoh: Pecinan" />
        </div>
      </div>

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, alignItems: 'center', marginTop: 12 }}>
        {PERIODE.map(([label, hitung]) => (
          <button key={label} type="button" className="mini" onClick={() => { const [d, s] = hitung(); onUbah({ dari: d, sampai: s }) }}>
            {label}
          </button>
        ))}
        <label style={{ display: 'flex', alignItems: 'center', gap: 6, margin: '0 0 0 8px' }}>
          <input type="checkbox" checked={filter.batal} onChange={(e) => onUbah({ batal: e.target.checked })} />
          Termasuk yang dibatalkan
        </label>
        {aktif && <button type="button" className="mini" onClick={onReset}>Reset filter</button>}
      </div>

      <div className="total-bar">
        <span>Masuk: <b className="utama">{rupiah(ringkas.masuk)}</b></span>
        <span>Keluar: <b className="utama">{rupiah(ringkas.keluar)}</b></span>
        <span>Bersih: <b className="utama">{rupiah(ringkas.masuk - ringkas.keluar)}</b></span>
      </div>
      <p className="muted" style={{ margin: '6px 0 0' }}>
        Total mengikuti filter di atas dan tidak menghitung transaksi yang dibatalkan.
      </p>
    </div>
  )
}
