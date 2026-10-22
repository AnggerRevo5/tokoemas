import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import Pesan from '../components/common/Pesan'
import FormAksiCuci from '../components/cuci/FormAksiCuci'
import TabelDaftarCuci from '../components/cuci/TabelDaftarCuci'
import BarangDiPencuci from '../components/cuci/BarangDiPencuci'

export default function Cuci({ me }) {
  const isCabang = me.role === 'cabang'
  const bisaCatat = me.role === 'pj' || me.role === 'super_admin'
  const bisaVoid = me.role === 'super_admin'

  const [filter, setFilter] = useState('aktif')
  const [rows, setRows] = useState([])
  const [items, setItems] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [aksi, setAksi] = useState(null)

  const muat = useCallback(() => {
    api(`/cuci?status=${filter}`).then(setRows).catch((e) => setError(e.message))
    if (!isCabang) {
      api('/cucian/barang').then(setItems).catch(() => {})
    }
  }, [filter, isCabang])

  useEffect(() => {
    muat()
  }, [muat])

  function mulaiAksi(jenis, r) {
    setAksi({ jenis, id: r.id, sisa: r.sisa, qty: r.qty, n: Date.now() })
    setError('')
    setInfo('')
  }

  function resetPesan() {
    setError('')
    setInfo('')
  }

  const itemJob = aksi && aksi.jenis === 'kirim' ? items.filter((i) => i.job_id === aksi.id) : []

  return (
    <div>
      <Pesan error={error} info={info} />

      <div className="card">
        <h3>Daftar Cuci</h3>
        <p className="muted">
          Owner memilih barang di menu Stok lalu klik Cuci. PJ menimbang tiap barang (berat awal) lalu mengirimnya
          ke pencuci. Saat barang kembali, PJ menimbang lagi (berat sesudah) dan mencatat susutnya. Barang yang
          selesai dikembalikan ke back office dengan berat terbaru, lalu owner mengonfirmasi sudah diterima.
          Kalau tidak jadi dicuci, PJ bisa membatalkan dan mengembalikan barang yang belum selesai.
        </p>

        <div className="form-grid">
          <div>
            <label>Tampilkan</label>
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="aktif">Belum selesai</option>
              <option value="menunggu">Menunggu PJ</option>
              <option value="proses">Proses</option>
              <option value="selesai">Selesai</option>
              <option value="batal">Dibatalkan</option>
              <option value="semua">Semua</option>
            </select>
          </div>
        </div>

        {aksi && (
          <FormAksiCuci
            key={aksi.n}
            aksi={aksi}
            itemJob={itemJob}
            onMulai={resetPesan}
            onSelesai={(pesan) => {
              setInfo(pesan)
              setAksi(null)
              muat()
            }}
            onError={setError}
            onTutup={() => setAksi(null)}
          />
        )}

        <TabelDaftarCuci rows={rows} bisaCatat={bisaCatat} bisaVoid={bisaVoid} onAksi={mulaiAksi} />
      </div>

      {!isCabang && (
        <BarangDiPencuci
          items={items}
          bisaCatat={bisaCatat}
          onMulai={resetPesan}
          onSelesai={(pesan) => {
            setInfo(pesan)
            muat()
          }}
          onError={setError}
        />
      )}
    </div>
  )
}
