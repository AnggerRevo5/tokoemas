import { useState } from 'react'
import { api } from '../../services/api'
import { simpanTema, temaAwal } from '../../utils/tema'
import TombolTema from '../common/TombolTema'

export default function Login({ onSuccess }) {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [tema, setTema] = useState(temaAwal)

  async function submit(e) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      // backend memasang cookie sesi; token tidak lagi disimpan di browser oleh JavaScript
      await api('/login', {
        method: 'POST',
        body: JSON.stringify({ username, password }),
      })
      onSuccess()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  function ubahTema() {
    const baru = tema === 'gelap' ? 'terang' : 'gelap'
    simpanTema(baru)
    setTema(baru)
  }

  return (
    <div className="login-page">
      <TombolTema tema={tema} onUbah={ubahTema} className="login-tema" />

      <form className="card login" onSubmit={submit}>
        <h1>Pembukuan Toko Emas</h1>
        <p className="muted">Bulan Purnama</p>

        <label>Username</label>
        <input value={username} onChange={(e) => setUsername(e.target.value)} autoFocus required autoComplete="username" />

        <label>Password</label>
        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="current-password" />

        {error && <div className="error">{error}</div>}

        <button disabled={loading}>{loading ? 'Memproses...' : 'Masuk'}</button>
      </form>

      <small className="login-foot">Gunakan akun yang diberikan oleh Super Admin.</small>
    </div>
  )
}
