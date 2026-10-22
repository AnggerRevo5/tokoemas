import { useState } from 'react'
import { NAMA_ROLE } from '../../constants/roles'

const KUNCI = 'menu-terbuka'

function bacaTersimpan() {
  try {
    return JSON.parse(localStorage.getItem(KUNCI)) || {}
  } catch {
    return {}
  }
}

// Sidebar menu berkelompok dengan dropdown. Grup yang berisi halaman aktif terbuka otomatis.
// grup = [{ kode, judul, items: [[kodeMenu, label]] }]; badges = { kodeMenu: jumlah }.
export default function Sidebar({ grup, tab, onPilih, badges, me, tema, onTema }) {
  const [terbuka, setTerbuka] = useState(bacaTersimpan)

  const jumlah = (g) => g.items.reduce((s, [k]) => s + (badges[k] || 0), 0)

  function ubahGrup(kode, bukaSekarang) {
    setTerbuka((p) => {
      const baru = { ...p, [kode]: !bukaSekarang }
      try {
        localStorage.setItem(KUNCI, JSON.stringify(baru))
      } catch {
        // abaikan bila penyimpanan tidak tersedia
      }
      return baru
    })
  }

  return (
    <aside className="sidebar">
      <div className="brand">
        <strong>Pembukuan Toko Emas</strong>
        <span>Bulan Purnama</span>
      </div>

      <nav className="side-nav">
        {grup.map((g) => {
          // grup berisi satu menu tampil sebagai tautan biasa
          if (g.items.length === 1) {
            const [kode, label] = g.items[0]
            return (
              <button key={g.kode} type="button" className={tab === kode ? 'active' : ''} onClick={() => onPilih(kode)}>
                <span>{label}</span>
                {badges[kode] > 0 && <span className="badge">{badges[kode]}</span>}
              </button>
            )
          }
          const adaAktif = g.items.some(([k]) => k === tab)
          const buka = terbuka[g.kode] ?? adaAktif
          return (
            <div key={g.kode} className={buka ? 'side-grup buka' : 'side-grup'}>
              <button type="button" className="side-head" onClick={() => ubahGrup(g.kode, buka)}>
                <span>
                  {g.judul}
                  {!buka && jumlah(g) > 0 && <span className="badge">{jumlah(g)}</span>}
                </span>
              </button>
              <div className="side-sub">
                {g.items.map(([kode, label]) => (
                  <button key={kode} type="button" className={tab === kode ? 'active' : ''} onClick={() => onPilih(kode)}>
                    <span>{label}</span>
                    {badges[kode] > 0 && <span className="badge">{badges[kode]}</span>}
                  </button>
                ))}
              </div>
            </div>
          )
        })}
      </nav>

      <div className="side-user">
        {me.username} - {NAMA_ROLE[me.role] || me.role}
      </div>
      <button type="button" className="tema-btn" onClick={onTema}>
        {tema === 'gelap' ? 'Mode terang' : 'Mode gelap'}
      </button>
    </aside>
  )
}
