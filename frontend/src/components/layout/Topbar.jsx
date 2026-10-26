import { NAMA_ROLE } from '../../constants/roles'
import TombolTema from '../common/TombolTema'

// Bilah atas: tombol menu, judul halaman, ikon tema, akun, dan tombol keluar.
export default function Topbar({ judul, me, onMenu, tema, onTema, onLogout }) {
  return (
    <header className="topbar">
      <button type="button" className="ghost menu-btn" onClick={onMenu} aria-label="Buka atau tutup menu">
        <span />
        <span />
        <span />
      </button>
      <h1>{judul}</h1>
      <TombolTema tema={tema} onUbah={onTema} />
      <div className="user-chip">
        {me.username}
        <small>{NAMA_ROLE[me.role] || me.role}</small>
      </div>
      <button type="button" className="ghost" onClick={onLogout}>Keluar</button>
    </header>
  )
}
