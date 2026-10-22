import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import Pesan from '../components/common/Pesan'
import RingkasanAkses from '../components/akun/RingkasanAkses'
import FormAkun from '../components/akun/FormAkun'
import FormCabang from '../components/akun/FormCabang'
import FormEditAkun from '../components/akun/FormEditAkun'
import FormPasswordAkun from '../components/akun/FormPasswordAkun'
import TabelAkun from '../components/akun/TabelAkun'

// Halaman Super Admin: daftar akun, tambah akun, ubah role/cabang/status, ganti password, tambah cabang.
export default function Akun({ me }) {
  const [rows, setRows] = useState([])
  const [ringkasan, setRingkasan] = useState([])
  const [locations, setLocations] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [aksi, setAksi] = useState(null) // { jenis: 'ubah' | 'password', row, n }

  const cabangList = locations.filter((l) => l.type === 'cabang')

  const muat = useCallback(() => {
    api('/akun')
      .then((d) => {
        setRows(d.rows)
        setRingkasan(d.ringkasan)
      })
      .catch((e) => setError(e.message))
    api('/locations').then(setLocations).catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    muat()
  }, [muat])

  function resetPesan() {
    setError('')
    setInfo('')
  }

  function bukaAksi(jenis, row) {
    setAksi({ jenis, row, n: Date.now() })
    resetPesan()
  }

  async function toggleAktif(row) {
    const nonaktifkan = row.is_active
    if (nonaktifkan && !window.confirm(`Nonaktifkan akun ${row.username}? Akun ini tidak bisa login lagi.`)) return
    resetPesan()
    try {
      const res = await api(`/akun/${row.id}`, {
        method: 'PUT',
        body: JSON.stringify({ role: row.role, location_id: row.location_id, is_active: !row.is_active }),
      })
      setInfo(res.message)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  const sukses = (pesan) => {
    setInfo(pesan)
    muat()
  }

  return (
    <div>
      <Pesan error={error} info={info} />

      <RingkasanAkses ringkasan={ringkasan} />

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Daftar Akun</h3>

        {aksi && aksi.jenis === 'ubah' && (
          <FormEditAkun
            key={aksi.n}
            row={aksi.row}
            cabangList={cabangList}
            onMulai={resetPesan}
            onSelesai={(pesan) => {
              setAksi(null)
              sukses(pesan)
            }}
            onError={setError}
            onTutup={() => setAksi(null)}
          />
        )}
        {aksi && aksi.jenis === 'password' && (
          <FormPasswordAkun
            key={aksi.n}
            row={aksi.row}
            onMulai={resetPesan}
            onSelesai={(pesan) => {
              setAksi(null)
              sukses(pesan)
            }}
            onError={setError}
            onTutup={() => setAksi(null)}
          />
        )}

        <TabelAkun
          rows={rows}
          meId={me.user_id}
          onUbah={(r) => bukaAksi('ubah', r)}
          onPassword={(r) => bukaAksi('password', r)}
          onToggleAktif={toggleAktif}
        />
      </div>

      <FormAkun cabangList={cabangList} onMulai={resetPesan} onSukses={sukses} onError={setError} />

      <FormCabang cabangList={cabangList} onMulai={resetPesan} onSukses={sukses} onError={setError} />
    </div>
  )
}
