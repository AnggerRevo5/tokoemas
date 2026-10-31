import { LogOut, Menu } from 'lucide-react'
import { NAMA_ROLE } from '../../constants/roles'
import TombolTema from '../common/TombolTema'

// Bilah atas: tombol menu, judul halaman, ikon tema, akun, dan tombol keluar.
export default function Topbar({ judul, me, onMenu, tema, onTema, onLogout }) {
  return (
    <header className="topbar">
      <button type="button" className="ghost menu-btn" onClick={onMenu} aria-label="Buka atau tutup menu">
        <Menu size={20} strokeWidth={1.9} aria-hidden="true" />
      </button>
      <h1>{judul}</h1>
      <TombolTema tema={tema} onUbah={onTema} />
      <div className="user-chip">
        <span className="avatar" aria-hidden="true">{(me.username || '?').slice(0, 1)}</span>
        <div>
          {me.username}
          <small>{NAMA_ROLE[me.role] || me.role}</small>
        </div>
      </div>
      <button type="button" className="ghost tombol-keluar" onClick={onLogout} aria-label="Keluar" title="Keluar">
        <LogOut size={16} strokeWidth={1.9} aria-hidden="true" />
        <span>Keluar</span>
      </button>
    </header>
  )
}
