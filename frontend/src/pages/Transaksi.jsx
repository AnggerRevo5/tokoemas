import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import Pesan from '../components/common/Pesan'
import KotakBatal from '../components/common/KotakBatal'
import FormTransaksi from '../components/transaksi/FormTransaksi'
import TabelTransaksi from '../components/transaksi/TabelTransaksi'

export default function Transaksi({ jenis, me }) {
  const isKas = jenis === 'kas'
  const bisaVoid = me.role === 'super_admin' || me.role === 'cabang'

  const [locations, setLocations] = useState([])
  const [locId, setLocId] = useState(me.role === 'cabang' ? me.location_id : 1)
  const [rows, setRows] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [voidId, setVoidId] = useState(null)
  const [voidReason, setVoidReason] = useState('')

  const lokasi = locations.find((l) => l.id === Number(locId))

  useEffect(() => {
    api('/locations')
      .then((d) => {
        setLocations(d)
        // pilih lokasi pertama bila lokasi awal tidak ada
        setLocId((lama) => (d.some((l) => l.id === Number(lama)) || d.length === 0 ? lama : d[0].id))
      })
      .catch((e) => setError(e.message))
  }, [])

  const muat = useCallback(() => {
    api(`/${jenis}?location_id=${locId}`)
      .then(setRows)
      .catch((e) => setError(e.message))
  }, [jenis, locId])

  useEffect(() => {
    muat()
  }, [muat])

  function mulaiBatal(id) {
    setVoidId(id)
    setVoidReason('')
    setError('')
    setInfo('')
  }

  async function konfirmasiBatal() {
    if (!voidReason.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    setError('')
    try {
      await api(`/${jenis}/${voidId}/void`, {
        method: 'POST',
        body: JSON.stringify({ reason: voidReason }),
      })
      setInfo('Transaksi dibatalkan')
      setVoidId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      <Pesan error={error} info={info} />

      <FormTransaksi
        jenis={jenis}
        me={me}
        locations={locations}
        locId={locId}
        onLocId={setLocId}
        onMulai={() => {
          setError('')
          setInfo('')
        }}
        onSukses={(res) => {
          setInfo(res.message)
          muat()
        }}
        onError={setError}
      />

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Riwayat{lokasi ? ` - ${lokasi.name}` : ''}</h3>

        {voidId && (
          <KotakBatal onKonfirmasi={konfirmasiBatal} onTutup={() => setVoidId(null)}>
            <div>Batalkan transaksi #{voidId}. Tulis alasannya:</div>
            <input
              value={voidReason}
              onChange={(e) => setVoidReason(e.target.value)}
              placeholder="Contoh: salah input nominal"
              autoFocus
            />
          </KotakBatal>
        )}

        <TabelTransaksi rows={rows} isKas={isKas} bisaVoid={bisaVoid} onBatal={mulaiBatal} />
      </div>
    </div>
  )
}
