import { rupiah } from '../../utils/format'
import { labelKategori } from '../../constants/kategori'

// daftar baris "kategori ... nominal" pada laporan laba rugi
export default function BarisRincian({ data }) {
  const daftar = Object.entries(data || {})
  if (daftar.length === 0) return <div className="muted">Tidak ada data</div>
  return daftar.map(([k, v]) => (
    <div className="row-line" key={k}>
      <span>{labelKategori(k)}</span>
      <span>{rupiah(v)}</span>
    </div>
  ))
}
