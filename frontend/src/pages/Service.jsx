import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { toggleId } from '../utils/format'
import { STATUS_SERVICE, aksiUntuk } from '../constants/service'
import Pesan from '../components/common/Pesan'
import KotakBatal from '../components/common/KotakBatal'
import FormTerimaService from '../components/service/FormTerimaService'
import BarAksiService from '../components/service/BarAksiService'
import TabelService from '../components/service/TabelService'

export default function Service({ me, onUbah }) {
  const isCabang = me.role === 'cabang'
  const isPJ = me.role === 'pj'
  const bisaVoid = me.role === 'super_admin'

  const [locations, setLocations] = useState([])
  const [rows, setRows] = useState([])
  const [status, setStatus] = useState('aktif')
  const [cari, setCari] = useState('')
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [pilih, setPilih] = useState({})
  const [aksi, setAksi] = useState(null)

  const [voidId, setVoidId] = useState(null)
  const [alasan, setAlasan] = useState('')

  const cabangList = locations.filter((l) => l.type === 'cabang')

  useEffect(() => {
    api('/locations').then(setLocations).catch((e) => setError(e.message))
  }, [])

  const muat = useCallback(() => {
    const p = new URLSearchParams({ status })
    if (cari.trim()) p.set('q', cari.trim())
    api(`/service?${p.toString()}`)
      .then((d) => {
        setRows(d)
        setPilih((lama) => {
          const baru = {}
          d.forEach((r) => {
            if (lama[r.id]) baru[r.id] = true
          })
          return baru
        })
      })
      .catch((e) => setError(e.message))
    if (onUbah) onUbah()
  }, [status, cari])

  useEffect(() => {
    muat()
  }, [muat])

  const terpilih = rows.filter((r) => pilih[r.id])
  const statusSet = new Set(terpilih.map((r) => r.status))
  const statusPilih = statusSet.size === 1 ? [...statusSet][0] : null
  const aksiTersedia = statusPilih ? aksiUntuk(statusPilih, me.role) : []

  useEffect(() => {
    if (aksi && !aksiTersedia.some(([k]) => k === aksi)) setAksi(null)
  }, [statusPilih])

  function resetPesan() {
    setError('')
    setInfo('')
  }

  function mulaiBatal(id) {
    setVoidId(id)
    setAlasan('')
    resetPesan()
  }

  async function konfirmasiBatal() {
    if (!alasan.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    setError('')
    try {
      const res = await api(`/service/void/${voidId}`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasan }),
      })
      setInfo(res.message)
      setVoidId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      <Pesan error={error} info={info} />

      {!isPJ && (
        <FormTerimaService
          isCabang={isCabang}
          me={me}
          cabangList={cabangList}
          onMulai={resetPesan}
          onSukses={(res) => {
            setInfo(res.message)
            setStatus('aktif')
            muat()
          }}
          onError={setError}
        />
      )}

      <div className="card">
        <div className="form-grid">
          <div>
            <label>Status</label>
            <select value={status} onChange={(e) => { setStatus(e.target.value); setPilih({}) }}>
              <option value="aktif">Belum diambil</option>
              {Object.entries(STATUS_SERVICE).map(([k, [l]]) => (
                <option key={k} value={k}>{l}</option>
              ))}
              <option value="semua">Semua</option>
            </select>
          </div>
          <div>
            <label>Cari customer, barang, nota, atau penyervice</label>
            <input value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Rina, SRV-00001, Anto..." />
          </div>
        </div>
        <p className="muted" style={{ marginBottom: 0 }}>{rows.length} barang service</p>
      </div>

      {terpilih.length > 0 && (
        <BarAksiService
          terpilih={terpilih}
          statusPilih={statusPilih}
          aksiTersedia={aksiTersedia}
          aksi={aksi}
          onAksi={setAksi}
          cabangList={cabangList}
          onMulai={resetPesan}
          onSelesai={(pesan) => {
            setInfo(pesan)
            setAksi(null)
            setPilih({})
            muat()
          }}
          onError={setError}
          onBatalPilih={() => {
            setPilih({})
            setAksi(null)
          }}
        />
      )}

      <div className="card" style={{ marginTop: 16 }}>
        {voidId && (
          <KotakBatal onKonfirmasi={konfirmasiBatal} onTutup={() => setVoidId(null)}>
            <div>Batalkan barang service #{voidId} (salah input). Tulis alasannya:</div>
            <input value={alasan} onChange={(e) => setAlasan(e.target.value)} autoFocus />
          </KotakBatal>
        )}

        <TabelService
          rows={rows}
          role={me.role}
          bisaVoid={bisaVoid}
          pilih={pilih}
          onToggle={(id) => setPilih((p) => toggleId(p, id))}
          onBatal={mulaiBatal}
        />
      </div>
    </div>
  )
}
