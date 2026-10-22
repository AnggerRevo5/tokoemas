import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { awalBulan, hariIni } from '../utils/format'
import Pesan from '../components/common/Pesan'
import FormAksiLebur from '../components/lebur/FormAksiLebur'
import TabelLebur from '../components/lebur/TabelLebur'
import PanelSiapJual from '../components/lebur/PanelSiapJual'
import RiwayatPenjualan from '../components/lebur/RiwayatPenjualan'

export default function Lebur({ me }) {
  const bisaCatat = me.role === 'pj' || me.role === 'super_admin'
  const bisaUndo = me.role === 'super_admin'

  const [filter, setFilter] = useState('berjalan')
  const [batches, setBatches] = useState([])
  const [siap, setSiap] = useState([])
  const [penj, setPenj] = useState({ rows: [], total_modal: 0, total_jual: 0, total_untung: 0 })
  const [dari, setDari] = useState(awalBulan())
  const [sampai, setSampai] = useState(hariIni())
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [aksi, setAksi] = useState(null)

  const muat = useCallback(() => {
    api(`/lebur?status=${filter}`).then(setBatches).catch((e) => setError(e.message))
    api('/penjualan/siap').then(setSiap).catch((e) => setError(e.message))
    api(`/penjualan?from=${dari}&to=${sampai}`).then(setPenj).catch((e) => setError(e.message))
  }, [filter, dari, sampai])

  useEffect(() => {
    muat()
  }, [muat])

  const batchAksi = aksi ? batches.find((b) => b.id === aksi.id) : null

  function bukaAksi(jenis, b) {
    setAksi({ jenis, id: b.id, n: Date.now() })
    setError('')
    setInfo('')
  }

  function resetPesan() {
    setError('')
    setInfo('')
  }

  return (
    <div>
      <Pesan error={error} info={info} />

      <div className="card">
        <h3>Proses Lebur</h3>
        <p className="muted">
          Owner memulai lebur dari menu Stok (centang perhiasan di back office, klik Lebur). PJ mencatat:
          berat setelah dipotong dan leburan kotor, lalu cukim. Cukim dijual ke penadah di bagian bawah.
          Angka berat di tabel: awal → setelah dipotong → leburan kotor → cukim.
        </p>

        <div className="form-grid">
          <div>
            <label>Tampilkan</label>
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="berjalan">Sedang berjalan</option>
              <option value="terjual">Sudah terjual</option>
              <option value="batal">Dibatalkan</option>
              <option value="semua">Semua</option>
            </select>
          </div>
        </div>

        {aksi && batchAksi && (
          <FormAksiLebur
            key={aksi.n}
            aksi={aksi}
            batch={batchAksi}
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

        <TabelLebur batches={batches} bisaCatat={bisaCatat} bisaUndo={bisaUndo} onAksi={bukaAksi} />
      </div>

      <PanelSiapJual
        siap={siap}
        bisaCatat={bisaCatat}
        onMulai={resetPesan}
        onSukses={(pesan) => {
          setInfo(pesan)
          muat()
        }}
        onError={setError}
      />

      <RiwayatPenjualan
        penj={penj}
        dari={dari}
        onDari={setDari}
        sampai={sampai}
        onSampai={setSampai}
        bisaCatat={bisaCatat}
        onMulai={resetPesan}
        onSelesai={(pesan) => {
          setInfo(pesan)
          muat()
        }}
        onError={setError}
      />
    </div>
  )
}
