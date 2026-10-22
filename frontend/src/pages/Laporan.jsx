import { useState, useEffect } from 'react'
import { api, unduh } from '../services/api'
import { awalBulan, hariIni } from '../utils/format'
import { LAPORAN } from '../constants/laporan'
import FormLaporan from '../components/laporan/FormLaporan'
import PratinjauLaporan from '../components/laporan/PratinjauLaporan'

export default function Laporan({ me }) {
  const [dari, setDari] = useState(awalBulan())
  const [sampai, setSampai] = useState(hariIni())
  const [locId, setLocId] = useState('0')
  const [locations, setLocations] = useState([])
  const [pilih, setPilih] = useState(() => Object.fromEntries(LAPORAN.map(([k]) => [k, true])))
  const [hasil, setHasil] = useState(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api('/locations').then(setLocations).catch((e) => setError(e.message))
  }, [])

  const jenis = LAPORAN.filter(([k]) => pilih[k]).map(([k]) => k).join(',')
  const query = `jenis=${jenis}&from=${dari}&to=${sampai}&location_id=${locId}`

  function toggle(k) {
    setPilih((p) => ({ ...p, [k]: !p[k] }))
    setHasil(null)
  }

  function semua(nilai) {
    setPilih(Object.fromEntries(LAPORAN.map(([k]) => [k, nilai])))
    setHasil(null)
  }

  function cekPilihan() {
    if (!jenis) {
      setError('Pilih minimal satu laporan')
      return false
    }
    return true
  }

  async function ambil() {
    return api(`/laporan/preview?${query}`)
  }

  async function pratinjau() {
    setError('')
    if (!cekPilihan()) return
    setLoading(true)
    try {
      setHasil(await ambil())
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function excel() {
    setError('')
    if (!cekPilihan()) return
    setLoading(true)
    try {
      await unduh(`/laporan/excel?${query}`, `laporan_${dari}_sd_${sampai}.xlsx`)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function cetak() {
    setError('')
    if (!cekPilihan()) return
    setLoading(true)
    try {
      setHasil(await ambil())
      setTimeout(() => window.print(), 500)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <FormLaporan
        dari={dari}
        onDari={(v) => { setDari(v); setHasil(null) }}
        sampai={sampai}
        onSampai={(v) => { setSampai(v); setHasil(null) }}
        locId={locId}
        onLocId={(v) => { setLocId(v); setHasil(null) }}
        locations={locations}
        pilih={pilih}
        onToggle={toggle}
        onSemua={semua}
        error={error}
        loading={loading}
        onPratinjau={pratinjau}
        onExcel={excel}
        onCetak={cetak}
      />

      {hasil && <PratinjauLaporan hasil={hasil} username={me.username} />}
    </div>
  )
}
