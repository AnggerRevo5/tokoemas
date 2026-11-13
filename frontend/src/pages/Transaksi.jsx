import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import Pesan from '../components/common/Pesan'
import Popup from '../components/common/Popup'
import FormTransaksi from '../components/transaksi/FormTransaksi'
import TabelTransaksi from '../components/transaksi/TabelTransaksi'
import PenjualanDariKas from '../components/transaksi/PenjualanDariKas'
import CardGadai from '../components/barang/CardGadai'
import FilterKas, { FILTER_KAS_KOSONG } from '../components/transaksi/FilterKas'

export default function Transaksi({ jenis, me }) {
  const isKas = jenis === 'kas'
  const bisaVoid = me.role === 'super_admin' || me.role === 'cabang'

  const [locations, setLocations] = useState([])
  const [locId, setLocId] = useState(me.role === 'cabang' ? me.location_id : 1)
  const [rows, setRows] = useState([])
  const [filter, setFilter] = useState(FILTER_KAS_KOSONG) // hanya untuk riwayat kas
  const [limit, setLimit] = useState(200)
  const [ringkas, setRingkas] = useState({ masuk: 0, keluar: 0 })
  const [adaLagi, setAdaLagi] = useState(false)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [voidId, setVoidId] = useState(null)
  const [voidReason, setVoidReason] = useState('')
  const [kasJual, setKasJual] = useState(null)
  const [gadaiItem, setGadaiItem] = useState(null)

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

  const ubahFilter = useCallback((bagian) => {
    setFilter((f) => ({ ...f, ...bagian }))
    setLimit(200)
  }, [])

  const muat = useCallback(() => {
    if (!isKas) {
      api(`/${jenis}?location_id=${locId}`)
        .then(setRows)
        .catch((e) => setError(e.message))
      return
    }
    const p = new URLSearchParams({ location_id: locId, limit })
    if (filter.dari) p.set('from', filter.dari)
    if (filter.sampai) p.set('to', filter.sampai)
    if (filter.arah) p.set('arah', filter.arah)
    if (filter.kelompok) p.set('kelompok', filter.kelompok)
    if (filter.metode) p.set('metode', filter.metode)
    if (filter.q) p.set('q', filter.q)
    if (filter.batal) p.set('batal', '1')
    api(`/kas?${p}`)
      .then((d) => {
        setRows(d.rows)
        setRingkas({ masuk: d.total_masuk, keluar: d.total_keluar })
        setAdaLagi(d.ada_lagi)
      })
      .catch((e) => setError(e.message))
  }, [jenis, isKas, locId, filter, limit])

  useEffect(() => {
    muat()
  }, [muat])

  function mulaiBatal(id) {
    setVoidId(id)
    setVoidReason('')
    setError('')
    setInfo('')
  }

  // kas dari gadai diubah lewat card gadainya: ambil gadai lalu barangnya
  async function bukaGadai(r) {
    setError('')
    setInfo('')
    try {
      const g = await api(`/gadai/${r.gadai_id}`)
      const d = await api(`/barang?item_id=${g.item_id}&status=semua`)
      if (d.rows.length === 0) throw new Error('Barang gadai tidak ditemukan')
      setGadaiItem(d.rows[0])
    } catch (err) {
      setError(err.message)
    }
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

        {isKas && (
          <FilterKas
            filter={filter}
            onUbah={ubahFilter}
            onReset={() => { setFilter(FILTER_KAS_KOSONG); setLimit(200) }}
            ringkas={ringkas}
          />
        )}

        {voidId && (
          <Popup
            judul={`Batalkan transaksi #${voidId}?`}
            labelKonfirmasi="Ya, batalkan"
            danger
            nonaktif={!voidReason.trim()}
            onKonfirmasi={konfirmasiBatal}
            onTutup={() => setVoidId(null)}
          >
            <p className="muted">Transaksi ini akan dibatalkan dan tidak dihitung lagi. Tulis alasannya:</p>
            <input
              value={voidReason}
              onChange={(e) => setVoidReason(e.target.value)}
              placeholder="Contoh: salah input nominal"
              autoFocus
            />
          </Popup>
        )}

        {gadaiItem && (
          <CardGadai
            key={gadaiItem.id}
            item={gadaiItem}
            me={me}
            onUbah={(pesan) => {
              setInfo(pesan)
              muat()
            }}
            onTutup={() => setGadaiItem(null)}
          />
        )}

        {kasJual && (
          <PenjualanDariKas
            kas={kasJual}
            me={me}
            onUbah={(pesan) => {
              setInfo(pesan)
              muat()
            }}
            onTutup={() => setKasJual(null)}
          />
        )}

        <TabelTransaksi
          rows={rows}
          isKas={isKas}
          bisaVoid={bisaVoid}
          onBatal={mulaiBatal}
          onJual={isKas ? (r) => { setError(''); setInfo(''); setKasJual(r) } : undefined}
          onGadai={isKas ? bukaGadai : undefined}
        />
        {isKas && adaLagi && (
          <button type="button" className="ghost" style={{ marginTop: 12 }} onClick={() => setLimit((l) => l + 200)}>
            Muat lebih banyak
          </button>
        )}
      </div>
    </div>
  )
}
