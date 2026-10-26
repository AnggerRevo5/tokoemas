import { useState, useEffect, useCallback } from 'react'
import { api } from '../../services/api'
import { menuUntuk, grupMenuUntuk } from '../../constants/roles'
import { simpanTema, temaAwal } from '../../utils/tema'
import Sidebar from './Sidebar'
import Topbar from './Topbar'
import AreaKonten from './AreaKonten'
import Dashboard from '../dashboard/Dashboard'
import Transaksi from '../../pages/Transaksi'
import Hutang from '../../pages/Hutang'
import LabaRugi from '../../pages/LabaRugi'
import Laporan from '../../pages/Laporan'
import Cuci from '../../pages/Cuci'
import Lebur from '../../pages/Lebur'
import Barang from '../../pages/Barang'
import Service from '../../pages/Service'
import Akun from '../../pages/Akun'

const LAYAR_LEBAR = 1024
const KUNCI_SIDEBAR = 'sidebar'

// Di layar lebar sidebar terbuka (kecuali pernah disembunyikan), di HP dan tablet tertutup.
function sidebarAwal() {
  if (typeof window === 'undefined' || window.innerWidth < LAYAR_LEBAR) return false
  try {
    return localStorage.getItem(KUNCI_SIDEBAR) !== 'tutup'
  } catch {
    return true
  }
}

// Kerangka halaman setelah login: sidebar, bilah atas, dan isi halaman yang dipilih.
export default function AppShell({ onLogout }) {
  const [me, setMe] = useState(null)
  const [tab, setTab] = useState('dashboard')
  const [buka, setBuka] = useState(sidebarAwal)
  const [tema, setTema] = useState(temaAwal)
  const [notif, setNotif] = useState({ menunggu: 0, belum_sampai: 0 })
  const [notifSvc, setNotifSvc] = useState({ badge: 0 })
  const [notifCuci, setNotifCuci] = useState({ badge: 0 })

  const ambilNotif = useCallback(() => {
    api('/barang/notif').then(setNotif).catch(() => {})
    api('/service/notif').then(setNotifSvc).catch(() => {})
    api('/cucian/notif').then(setNotifCuci).catch(() => {})
  }, [])

  useEffect(() => {
    api('/me')
      .then((m) => {
        setMe(m)
        if (m.role === 'pj') setTab('lebur')
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    ambilNotif()
  }, [tab, ambilNotif])

  function pilihMenu(kode) {
    setTab(kode)
    // di HP dan tablet, menu menutup sendiri setelah memilih halaman
    if (window.innerWidth < LAYAR_LEBAR) setBuka(false)
  }

  function ubahSidebar() {
    const baru = !buka
    setBuka(baru)
    if (window.innerWidth >= LAYAR_LEBAR) {
      try {
        localStorage.setItem(KUNCI_SIDEBAR, baru ? 'buka' : 'tutup')
      } catch {
        // abaikan bila penyimpanan tidak tersedia
      }
    }
  }

  function ubahTema() {
    const baru = tema === 'gelap' ? 'terang' : 'gelap'
    simpanTema(baru)
    setTema(baru)
  }

  if (!me) return <div className="content muted">Memuat...</div>

  const bukanPJ = me.role !== 'pj'
  const bukanCabang = me.role !== 'cabang'
  const judul = (menuUntuk(me.role).find(([k]) => k === tab) || [])[1] || ''

  return (
    <div className={buka ? 'app buka' : 'app'}>
      <Sidebar
        grup={grupMenuUntuk(me.role)}
        tab={tab}
        onPilih={pilihMenu}
        badges={{ stok: notif.menunggu, service: notifSvc.badge, cuci: notifCuci.badge }}
        me={me}
      />
      <div className="overlay" onClick={() => setBuka(false)} />

      <div className="main">
        <Topbar judul={judul} me={me} onMenu={ubahSidebar} tema={tema} onTema={ubahTema} onLogout={onLogout} />

        <AreaKonten>
          {tab === 'dashboard' && bukanPJ && <Dashboard />}
          {tab === 'kas' && bukanPJ && <Transaksi jenis="kas" me={me} />}
          {tab === 'stok' && bukanPJ && <Barang me={me} onUbah={ambilNotif} />}
          {tab === 'cuci' && <Cuci me={me} />}
          {tab === 'service' && <Service me={me} onUbah={ambilNotif} />}
          {tab === 'lebur' && bukanCabang && <Lebur me={me} />}
          {tab === 'laporan' && bukanCabang && bukanPJ && <Laporan me={me} />}
          {tab === 'hutang' && bukanCabang && bukanPJ && <Hutang me={me} />}
          {tab === 'labarugi' && bukanCabang && bukanPJ && <LabaRugi />}
          {tab === 'akun' && me.role === 'super_admin' && <Akun me={me} />}
        </AreaKonten>
      </div>
    </div>
  )
}
