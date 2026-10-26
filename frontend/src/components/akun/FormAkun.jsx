import { useState } from 'react'
import { api } from '../../services/api'
import { NAMA_ROLE, URUTAN_ROLE } from '../../constants/roles'
import PilihanLokasi from '../common/PilihanLokasi'
import Spinner from '../common/Spinner'

// Form tambah akun baru. Role cabang wajib memilih cabang.
// onMulai: bersihkan pesan; onSukses(message); onError(msg).
export default function FormAkun({ cabangList, onMulai, onSukses, onError }) {
  const [loading, setLoading] = useState(false)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState('cabang')
  const [lokasi, setLokasi] = useState('')

  // cabang pertama sebagai pilihan awal
  const lokasiAktif = lokasi || (cabangList.length > 0 ? String(cabangList[0].id) : '')

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      const res = await api('/akun', {
        method: 'POST',
        body: JSON.stringify({
          username,
          password,
          role,
          location_id: role === 'cabang' ? Number(lokasiAktif) : 0,
        }),
      })
      setUsername('')
      setPassword('')
      onSukses(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="card" style={{ marginTop: 16 }} onSubmit={simpan}>
      <h3>Tambah Akun</h3>
      <div className="form-grid">
        <div>
          <label>Username</label>
          <input value={username} onChange={(e) => setUsername(e.target.value)} required autoComplete="off" />
        </div>
        <div>
          <label>Password (minimal 6 karakter)</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="new-password" />
        </div>
        <div>
          <label>Role</label>
          <select value={role} onChange={(e) => setRole(e.target.value)}>
            {URUTAN_ROLE.map((r) => (
              <option key={r} value={r}>{NAMA_ROLE[r]}</option>
            ))}
          </select>
        </div>
        {role === 'cabang' && (
          <div>
            <label>Cabang</label>
            <PilihanLokasi locations={cabangList} value={lokasiAktif} onChange={(e) => setLokasi(e.target.value)} />
          </div>
        )}
      </div>
      <button disabled={loading}>{loading ? <><Spinner />Menyimpan...</> : 'Tambah akun'}</button>
    </form>
  )
}
