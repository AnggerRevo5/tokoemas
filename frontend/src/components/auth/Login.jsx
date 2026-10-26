import { useState } from 'react'
import { api } from '../../services/api'
import { simpanTema, temaAwal } from '../../utils/tema'
import { m } from 'motion/react'
import { CircleAlert } from 'lucide-react'
import TombolTema from '../common/TombolTema'
import Spinner from '../common/Spinner'

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
      <div className="login-bulan" aria-hidden="true" />

      <TombolTema tema={tema} onUbah={ubahTema} className="login-tema" />

      <m.form
        className="card login"
        onSubmit={submit}
        initial={{ opacity: 0, y: 12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3, ease: 'easeOut' }}
      >
        <img className="brand-mark" src="/logo.png" alt="" aria-hidden="true" />
        <h1>Pembukuan Toko Emas</h1>
        <p className="login-sub">Bulan Purnama</p>

        <label>Username</label>
        <input value={username} onChange={(e) => setUsername(e.target.value)} autoFocus required autoComplete="username" />

        <label>Password</label>
        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="current-password" />

        {error && (
          <div className="error" role="alert">
            <CircleAlert size={18} aria-hidden="true" />
            <span>{error}</span>
          </div>
        )}

        <button disabled={loading}>{loading ? <><Spinner />Memproses...</> : 'Masuk'}</button>
      </m.form>

      <small className="login-foot">Gunakan akun yang diberikan oleh Super Admin.</small>
    </div>
  )
}
