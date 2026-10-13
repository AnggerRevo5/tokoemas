import { useState, useEffect, useCallback } from 'react'
import { api, getToken, setToken, clearToken } from './api'
import Transaksi from './pages/Transaksi'
import Hutang from './pages/Hutang'
import LabaRugi from './pages/LabaRugi'
import Laporan from './pages/Laporan'
import Cuci from './pages/Cuci'
import Lebur from './pages/Lebur'
import Barang from './pages/Barang'
import Service from './pages/Service'

const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)

const namaRole = {
  super_admin: 'Super Admin',
  owner: 'Owner',
  cabang: 'Akun Cabang',
  pj: 'Penanggung Jawab (PJ)',
}

function Login({ onSuccess }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function submit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      const data = await api('/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      })
      setToken(data.token)
      onSuccess()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="login-wrap">
      <form className="card login" onSubmit={submit}>
        <h1>Pembukuan Toko Emas</h1>
        <p className="muted">Bulan Purnama - Kepanjen &amp; Mall Dinoyo</p>

        <label>Username</label>
        <input value={username} onChange={(e) => setUsername(e.target.value)} autoFocus required />

        <label>Password</label>
        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />

        {error && <div className="error">{error}</div>}

        <button disabled={loading}>{loading ? 'Memproses...' : 'Masuk'}</button>
      </form>
    </div>
  )
}

function Dashboard() {
  const [saldo, setSaldo] = useState([])
  const [svc, setSvc] = useState({})
  const [error, setError] = useState('')

  useEffect(() => {
    api('/saldo').then(setSaldo).catch((err) => setError(err.message))
    api('/service/ringkas')
      .then((d) => setSvc(Object.fromEntries(d.map((x) => [x.location_id, x]))))
      .catch(() => {})
  }, [])

  const rincian = (s) =>
    [
      ['Di toko, menunggu diantar', s.di_toko],
      ['Menunggu dikirim ke penyervice', s.menunggu_servis],
      ['Sedang diservis', s.diproses],
      ['Selesai, menunggu dioper', s.selesai],
      ['Siap diambil customer', s.siap_diambil],
    ].filter(([, n]) => n > 0)

  return (
    <div>
      {error && <div className="error">{error}</div>}
      <div className="grid">
        {saldo.map((s) => {
          const sv = svc[s.location_id]
          return (
            <div className="card" key={s.location_id}>
              <h3>{s.name}</h3>
              <div className="label">Saldo kas</div>
              <div className="big">{rupiah(s.saldo_kas)}</div>
              <div className="label">Stok emas</div>
              <div className="big">
                {s.stok_gram} gram <span className="muted">({s.stok_pcs} pcs)</span>
              </div>

              <div className="label">Barang service</div>
              <div className="big">
                {sv ? sv.di_sini : 0} <span className="muted">barang</span>
              </div>
              {sv && rincian(sv).map(([l, n]) => (
                <div className="row-line" key={l}>
                  <span>{l}</span>
                  <span>{n}</span>
                </div>
              ))}
              {sv && sv.masuk > 0 && <div className="muted">{sv.masuk} barang sedang menuju sini</div>}
              {sv && sv.keluar > 0 && <div className="muted">{sv.keluar} barang sedang dikirim keluar</div>}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function Shell({ onLogout }) {
  const [me, setMe] = useState(null)
  const [tab, setTab] = useState('dashboard')
  const [notif, setNotif] = useState({ menunggu: 0, belum_sampai: 0 })
  const [notifSvc, setNotifSvc] = useState({ badge: 0 })

  const ambilNotif = useCallback(() => {
    api('/barang/notif').then(setNotif).catch(() => {})
    api('/service/notif').then(setNotifSvc).catch(() => {})
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

  if (!me) return <div className="page muted">Memuat...</div>

  let menu
  if (me.role === 'pj') {
    menu = [['lebur', 'Lebur'], ['service', 'Service']]
  } else {
    menu = [
      ['dashboard', 'Dashboard'],
      ['kas', 'Kas'],
      ['stok', 'Stok'],
      ['cuci', 'Cuci'],
      ['service', 'Service'],
    ]
    if (me.role !== 'cabang') {
      menu.push(['lebur', 'Lebur'])
    }
    if (me.role !== 'cabang') {
      menu.push(['laporan', 'Laporan'])
    }
    if (me.role !== 'cabang') {
      menu.push(['hutang', 'Hutang'])
      menu.push(['labarugi', 'Laba Rugi'])
    }
  }

  return (
    <div className="page">
      <header>
        <div>
          <h2>Pembukuan Toko Emas</h2>
          <span className="muted">
            {me.username} - {namaRole[me.role] || me.role}
          </span>
        </div>
        <button className="ghost" onClick={onLogout}>Keluar</button>
      </header>

      <nav>
        {menu.map(([k, l]) => (
          <button key={k} className={tab === k ? 'tab active' : 'tab'} onClick={() => setTab(k)}>
            {l}
            {k === 'stok' && notif.menunggu > 0 && <span className="badge">{notif.menunggu}</span>}
            {k === 'service' && notifSvc.badge > 0 && <span className="badge">{notifSvc.badge}</span>}
          </button>
        ))}
      </nav>

      {tab === 'dashboard' && me.role !== 'pj' && <Dashboard />}
      {tab === 'kas' && me.role !== 'pj' && <Transaksi jenis="kas" me={me} />}
      {tab === 'stok' && me.role !== 'pj' && <Barang me={me} onUbah={ambilNotif} />}
      {tab === 'cuci' && me.role !== 'pj' && <Cuci me={me} />}
      {tab === 'service' && <Service me={me} onUbah={ambilNotif} />}
      {tab === 'lebur' && me.role !== 'cabang' && <Lebur me={me} />}
      {tab === 'laporan' && me.role !== 'cabang' && me.role !== 'pj' && <Laporan me={me} />}
      {tab === 'hutang' && me.role !== 'cabang' && me.role !== 'pj' && <Hutang me={me} />}
      {tab === 'labarugi' && me.role !== 'cabang' && me.role !== 'pj' && <LabaRugi />}
    </div>
  )
}

export default function App() {
  const [loggedIn, setLoggedIn] = useState(!!getToken())

  function logout() {
    clearToken()
    setLoggedIn(false)
  }

  return loggedIn ? <Shell onLogout={logout} /> : <Login onSuccess={() => setLoggedIn(true)} />
}