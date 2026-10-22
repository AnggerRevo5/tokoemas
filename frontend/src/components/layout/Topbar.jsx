import { NAMA_ROLE } from '../../constants/roles'

// Bilah atas: tombol menu, judul halaman, tema, akun, dan tombol keluar.
export default function Topbar({ judul, me, onMenu, tema, onTema, onLogout }) {
  return (
    <header className="topbar">
      <button type="button" className="ghost menu-btn" onClick={onMenu} aria-label="Buka atau tutup menu">
        <span />
        <span />
        <span />
      </button>
      <h1>{judul}</h1>
      <button type="button" className="ghost tema-btn" onClick={onTema}>
        {tema === 'gelap' ? 'Mode terang' : 'Mode gelap'}
      </button>
      <div className="user-chip">
        {me.username}
        <small>{NAMA_ROLE[me.role] || me.role}</small>
      </div>
      <button type="button" className="ghost" onClick={onLogout}>Keluar</button>
    </header>
  )
}
