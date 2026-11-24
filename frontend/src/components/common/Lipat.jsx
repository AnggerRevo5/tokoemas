import { useState } from 'react'
import { ChevronDown } from 'lucide-react'

// Bagian yang bisa dibuka dan ditutup lewat tombol judul. Isi tetap ada di halaman saat ditutup
// (hanya disembunyikan), jadi isian atau pilihan di dalamnya tidak hilang.
export default function Lipat({ judul, bukaAwal = true, children }) {
  const [buka, setBuka] = useState(bukaAwal)
  return (
    <section style={{ marginBottom: 16 }}>
      <button
        type="button"
        className="ghost"
        aria-expanded={buka}
        onClick={() => setBuka(!buka)}
        style={{ width: '100%', display: 'flex', justifyContent: 'space-between', alignItems: 'center', textAlign: 'left', fontWeight: 600 }}
      >
        <span>{judul}</span>
        <ChevronDown size={18} aria-hidden="true" style={{ transform: buka ? 'rotate(180deg)' : 'none', transition: 'transform 0.18s ease' }} />
      </button>
      <div hidden={!buka} style={{ marginTop: 8 }}>{children}</div>
    </section>
  )
}
