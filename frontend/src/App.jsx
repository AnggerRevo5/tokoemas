import { useState, useEffect } from 'react'
import { api, getToken, setToken, clearToken } from './api'
import Transaksi from './pages/Transaksi'
import Hutang from './pages/Hutang'
import LabaRugi from './pages/LabaRugi'
import ExportExcel from './pages/ExportExcel'

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
  const [error, setError] = useState('')

  useEffect(() => {
    api('/saldo').then(setSaldo).catch((err) => setError(err.message))
  }, [])

  return (
    <div>
      {error && <div className="error">{error}</div>}
      <div className="grid">
        {saldo.map((s) => (
          <div className="card" key={s.location_id}>
            <h3>{s.name}</h3>
            <div className="label">Saldo kas</div>
            <div className="big">{rupiah(s.saldo_kas)}</div>
            <div className="label">Stok emas</div>
            <div className="big">
              {s.stok_gram} gram <span className="muted">({s.stok_pcs} pcs)</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function Shell({ onLogout }) {
  const [me, setMe] = useState(null)
  const [tab, setTab] = useState('dashboard')

  useEffect(() => {
    api('/me').then(setMe).catch(() => {})
  }, [])

  if (!me) return <div className="page muted">Memuat...</div>

  const menu = [
    ['dashboard', 'Dashboard'],
    ['kas', 'Kas'],
    ['stok', 'Stok'],
    ['export', 'Export Excel'],
  ]
  if (me.role !== 'cabang') {
    menu.push(['hutang', 'Hutang'])
    menu.push(['labarugi', 'Laba Rugi'])
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
          </button>
        ))}
      </nav>

      {tab === 'dashboard' && <Dashboard />}
      {tab === 'kas' && <Transaksi jenis="kas" me={me} />}
      {tab === 'stok' && <Transaksi jenis="stok" me={me} />}
      {tab === 'export' && <ExportExcel me={me} />}
      {tab === 'hutang' && me.role !== 'cabang' && <Hutang />}
      {tab === 'labarugi' && me.role !== 'cabang' && <LabaRugi />}
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