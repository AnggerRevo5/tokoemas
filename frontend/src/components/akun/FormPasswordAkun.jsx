import { useState } from 'react'
import { api } from '../../services/api'
import FormAksi from '../common/FormAksi'

// Form ganti password akun lain (hanya Super Admin).
export default function FormPasswordAkun({ row, onMulai, onSelesai, onError, onTutup }) {
  const [loading, setLoading] = useState(false)
  const [password, setPassword] = useState('')
  const [ulang, setUlang] = useState('')

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    if (password !== ulang) {
      onError('Password dan ulangi password tidak sama')
      return
    }
    setLoading(true)
    try {
      const res = await api(`/akun/${row.id}/password`, {
        method: 'POST',
        body: JSON.stringify({ password }),
      })
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <FormAksi onSubmit={simpan} onTutup={onTutup} loading={loading} danger={false} labelSubmit="Ganti password">
      <div>Ganti password akun <b>{row.username}</b>. Akun itu memakai password baru saat login berikutnya.</div>
      <div className="form-grid">
        <div>
          <label>Password baru (minimal 6 karakter)</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoFocus autoComplete="new-password" />
        </div>
        <div>
          <label>Ulangi password baru</label>
          <input type="password" value={ulang} onChange={(e) => setUlang(e.target.value)} required autoComplete="new-password" />
        </div>
      </div>
    </FormAksi>
  )
}
