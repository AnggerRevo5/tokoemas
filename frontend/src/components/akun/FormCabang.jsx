import { useState } from 'react'
import { api } from '../../services/api'

// Form tambah cabang baru. Setelah cabang dibuat, tambahkan akun dengan role Akun Cabang untuk cabang itu.
export default function FormCabang({ cabangList, onMulai, onSukses, onError }) {
  const [loading, setLoading] = useState(false)
  const [nama, setNama] = useState('')

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      const res = await api('/lokasi', { method: 'POST', body: JSON.stringify({ name: nama }) })
      setNama('')
      onSukses(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="card" style={{ marginTop: 16 }} onSubmit={simpan}>
      <h3>Tambah Cabang Baru</h3>
      <p className="muted">
        Cabang saat ini: {cabangList.length > 0 ? cabangList.map((l) => l.name).join(', ') : '-'}.
        Cabang baru langsung punya fitur yang sama dengan cabang lain. Setelah dibuat, buat akun dengan role
        Akun Cabang untuk cabang tersebut.
      </p>
      <div className="form-grid">
        <div>
          <label>Nama cabang</label>
          <input value={nama} onChange={(e) => setNama(e.target.value)} placeholder="Contoh: Cabang Malang Kota" required />
        </div>
      </div>
      <button disabled={loading}>{loading ? 'Menyimpan...' : 'Tambah cabang'}</button>
    </form>
  )
}
