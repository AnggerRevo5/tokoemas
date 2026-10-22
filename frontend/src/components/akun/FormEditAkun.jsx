import { useState } from 'react'
import { api } from '../../services/api'
import { NAMA_ROLE, URUTAN_ROLE } from '../../constants/roles'
import FormAksi from '../common/FormAksi'
import PilihanLokasi from '../common/PilihanLokasi'

// Form ubah role, cabang, dan status aktif satu akun.
export default function FormEditAkun({ row, cabangList, onMulai, onSelesai, onError, onTutup }) {
  const [loading, setLoading] = useState(false)
  const [role, setRole] = useState(row.role)
  const [lokasi, setLokasi] = useState(row.location_id ? String(row.location_id) : '')
  const [aktif, setAktif] = useState(row.is_active)

  const lokasiAktif = lokasi || (cabangList.length > 0 ? String(cabangList[0].id) : '')

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      const res = await api(`/akun/${row.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          role,
          location_id: role === 'cabang' ? Number(lokasiAktif) : 0,
          is_active: aktif,
        }),
      })
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <FormAksi onSubmit={simpan} onTutup={onTutup} loading={loading} danger={false} labelSubmit="Simpan">
      <div>Ubah akun <b>{row.username}</b>.</div>
      <div className="form-grid">
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
        <div>
          <label>Status</label>
          <select value={aktif ? '1' : '0'} onChange={(e) => setAktif(e.target.value === '1')}>
            <option value="1">Aktif</option>
            <option value="0">Nonaktif (tidak bisa login)</option>
          </select>
        </div>
      </div>
    </FormAksi>
  )
}
