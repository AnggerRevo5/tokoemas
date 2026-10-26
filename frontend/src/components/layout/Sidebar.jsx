import { useState } from 'react'
import { AnimatePresence, m } from 'motion/react'
import { ChevronDown } from 'lucide-react'
import { NAMA_ROLE } from '../../constants/roles'
import BadgeAngka from '../common/BadgeAngka'
import { IKON_GRUP, IKON_MENU } from './ikonMenu'

const KUNCI = 'menu-terbuka'

function bacaTersimpan() {
  try {
    return JSON.parse(localStorage.getItem(KUNCI)) || {}
  } catch {
    return {}
  }
}

// satu tombol menu; menu aktif diberi indikator emas tipis yang bergeser halus antar menu
function TombolMenu({ kode, label, aktif, n, onPilih }) {
  const Ikon = IKON_MENU[kode]
  return (
    <button
      type="button"
      className={aktif ? 'active' : ''}
      onClick={() => onPilih(kode)}
      aria-current={aktif ? 'page' : undefined}
    >
      {aktif && (
        <m.span layoutId="nav-indikator" className="nav-indikator" transition={{ duration: 0.25, ease: 'easeOut' }} />
      )}
      {Ikon && <Ikon size={18} strokeWidth={1.75} aria-hidden="true" />}
      <span className="nav-label">{label}</span>
      <BadgeAngka n={n} />
    </button>
  )
}

// Sidebar menu berkelompok dengan dropdown. Grup yang berisi halaman aktif terbuka otomatis.
// grup = [{ kode, judul, items: [[kodeMenu, label]] }]; badges = { kodeMenu: jumlah }.
export default function Sidebar({ grup, tab, onPilih, badges, me }) {
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
        <img className="brand-mark" src="/logo.png" alt="" aria-hidden="true" />
        <div>
          <strong>Pembukuan Toko Emas</strong>
          <span>Bulan Purnama</span>
        </div>
      </div>

      <nav className="side-nav">
        {grup.map((g) => {
          // grup berisi satu menu tampil sebagai tautan biasa
          if (g.items.length === 1) {
            const [kode, label] = g.items[0]
            return (
              <TombolMenu key={g.kode} kode={kode} label={label} aktif={tab === kode} n={badges[kode]} onPilih={onPilih} />
            )
          }
          const adaAktif = g.items.some(([k]) => k === tab)
          const buka = terbuka[g.kode] ?? adaAktif
          const IkonGrup = IKON_GRUP[g.kode]
          return (
            <div key={g.kode} className={buka ? 'side-grup buka' : 'side-grup'}>
              <button type="button" className="side-head" onClick={() => ubahGrup(g.kode, buka)} aria-expanded={buka}>
                {IkonGrup && <IkonGrup size={18} strokeWidth={1.75} aria-hidden="true" />}
                <span className="nav-label">{g.judul}</span>
                {!buka && <BadgeAngka n={jumlah(g)} />}
                <ChevronDown className="chevron" size={16} aria-hidden="true" />
              </button>
              <AnimatePresence initial={false}>
                {buka && (
                  <m.div
                    className="side-sub"
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: 'auto', opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    transition={{ duration: 0.2, ease: 'easeOut' }}
                  >
                    <div className="side-sub-isi">
                      {g.items.map(([kode, label]) => (
                        <TombolMenu key={kode} kode={kode} label={label} aktif={tab === kode} n={badges[kode]} onPilih={onPilih} />
                      ))}
                    </div>
                  </m.div>
                )}
              </AnimatePresence>
            </div>
          )
        })}
      </nav>

      <div className="side-user">
        <span className="avatar">{(me.username || '?').slice(0, 1)}</span>
        <span>
          {me.username} - {NAMA_ROLE[me.role] || me.role}
        </span>
      </div>
    </aside>
  )
}
