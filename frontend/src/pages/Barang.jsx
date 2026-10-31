import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { toggleId } from '../utils/format'
import { bisaPilih, tentukanMode, aksiUntukMode } from '../utils/barang'
import Pesan from '../components/common/Pesan'
import Popup from '../components/common/Popup'
import NotifBanner from '../components/barang/NotifBanner'
import FormBarangMasuk from '../components/barang/FormBarangMasuk'
import FormEditBarang from '../components/barang/FormEditBarang'
import PanelBaki from '../components/barang/PanelBaki'
import PindahBaki from '../components/barang/PindahBaki'
import PindahSatu from '../components/barang/PindahSatu'
import SpesifikasiBarang from '../components/barang/SpesifikasiBarang'
import FilterBarang from '../components/barang/FilterBarang'
import BarAksiBarang from '../components/barang/BarAksiBarang'
import TabelBarang from '../components/barang/TabelBarang'

export default function Barang({ me, onUbah }) {
  const isCabang = me.role === 'cabang'
  const bisaVoid = me.role === 'super_admin'

  const [locations, setLocations] = useState([])
  const [locId, setLocId] = useState(isCabang ? String(me.location_id) : '')
  const [status, setStatus] = useState('tersedia')
  const [jenisF, setJenisF] = useState('semua')
  const [cari, setCari] = useState('')
  const [data, setData] = useState({ rows: [], jumlah: 0, total_gram: 0, total_beli: 0, total_jual: 0 })
  const [notif, setNotif] = useState({ menunggu: 0, belum_sampai: 0, keluar: 0 })
  const [pilih, setPilih] = useState({})
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [aksi, setAksi] = useState(null)
  const [baki, setBaki] = useState({ rows: [], kapasitas: 100 })
  const [bakiF, setBakiF] = useState('')
  const [editItem, setEditItem] = useState(null)
  const [pindahItem, setPindahItem] = useState(null)

  const [voidItem, setVoidItem] = useState(null)
  const [alasan, setAlasan] = useState('')
  const [loadingBatal, setLoadingBatal] = useState(false)

  const tipeLok = (id) => (locations.find((l) => l.id === Number(id)) || {}).type
  const cabangList = locations.filter((l) => l.type === 'cabang')

  useEffect(() => {
    api('/locations')
      .then((d) => {
        setLocations(d)
        const b = d.find((l) => l.type === 'back_office')
        if (!isCabang && b) setLocId(String(b.id))
      })
      .catch((e) => setError(e.message))
  }, [])

  const muat = useCallback(() => {
    if (locId === '') return
    const p = new URLSearchParams({ location_id: locId, status })
    if (jenisF !== 'semua') p.set('jenis', jenisF)
    if (bakiF) p.set('baki_id', bakiF)
    if (cari.trim()) p.set('q', cari.trim())
    api(`/barang?${p.toString()}`)
      .then((d) => {
        setData(d)
        setPilih((lama) => {
          const baru = {}
          d.rows.forEach((r) => {
            if (lama[r.id] && bisaPilih(r)) baru[r.id] = true
          })
          return baru
        })
      })
      .catch((e) => setError(e.message))
    api(`/baki?location_id=${locId}`).then(setBaki).catch(() => {})
    api('/barang/notif').then(setNotif).catch(() => {})
  }, [locId, status, jenisF, cari, bakiF])

  useEffect(() => {
    muat()
  }, [muat])

  const bisaDipilih = data.rows.filter(bisaPilih)
  const terpilih = data.rows.filter((r) => pilih[r.id])
  const beratTerpilih = terpilih.reduce((s, r) => s + r.weight_gram, 0)
  const adaLM = terpilih.some((r) => r.jenis === 'logam_mulia')
  const semuaDipilih = bisaDipilih.length > 0 && bisaDipilih.every((r) => pilih[r.id])

  const mode = tentukanMode(terpilih, tipeLok)
  // pindah baki: semua yang dicentang perhiasan tersedia di satu lokasi yang punya baki
  const lokasiPindah = terpilih.length > 0 && terpilih.every(
    (r) => r.jenis === 'perhiasan' && r.status === 'tersedia' && r.location_id === terpilih[0].location_id,
  ) ? terpilih[0].location_id : null
  const bakiPindah = lokasiPindah ? baki.rows.filter((b) => b.location_id === lokasiPindah) : []
  const aksiTersedia = aksiUntukMode(mode, { isCabang, adaLM })

  useEffect(() => {
    if (aksi && !aksiTersedia.some(([k]) => k === aksi)) setAksi(null)
  }, [mode, adaLM])

  function toggleSemua() {
    if (semuaDipilih) {
      setPilih({})
    } else {
      const n = {}
      bisaDipilih.forEach((r) => (n[r.id] = true))
      setPilih(n)
    }
  }

  function resetPesan() {
    setError('')
    setInfo('')
  }

  // dipanggil setelah ada perubahan data: muat ulang daftar dan angka merah di menu
  function selesai() {
    muat()
    if (onUbah) onUbah()
  }

  function mulaiBatal(r) {
    setVoidItem(r)
    setAlasan('')
    resetPesan()
  }

  async function konfirmasiBatal() {
    if (!alasan.trim()) {
      setError('Alasan menghapus wajib diisi')
      return
    }
    setError('')
    setLoadingBatal(true)
    try {
      const res = await api(`/barang/void/${voidItem.id}`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasan }),
      })
      setInfo(res.message)
      if (editItem && editItem.id === voidItem.id) setEditItem(null)
      setVoidItem(null)
      selesai()
    } catch (err) {
      setVoidItem(null)
      setError(err.message)
    } finally {
      setLoadingBatal(false)
    }
  }

  return (
    <div>
      <Pesan error={error} info={info} />

      <NotifBanner
        notif={notif}
        isCabang={isCabang}
        onLihat={() => {
          if (!isCabang) setLocId('0')
          setStatus('dikirim')
        }}
      />

      <FormBarangMasuk
        me={me}
        isCabang={isCabang}
        locations={locations}
        onMulai={resetPesan}
        onSukses={({ message, lokasiId, status: statusBaru }) => {
          setBakiF('')
          setInfo(message)
          if (!isCabang) setLocId(lokasiId)
          setStatus(statusBaru)
          selesai()
        }}
        onError={setError}
      />

      <FilterBarang
        isCabang={isCabang}
        locations={locations}
        locId={locId}
        onLocId={(v) => {
          setLocId(v)
          setBakiF('')
          setPilih({})
        }}
        status={status}
        onStatus={setStatus}
        jenisF={jenisF}
        onJenis={setJenisF}
        cari={cari}
        onCari={setCari}
        data={data}
      />

      <PanelBaki
        baki={baki.rows}
        kapasitas={baki.kapasitas}
        bakiF={bakiF}
        onPilih={setBakiF}
        lokasiId={locId}
        bolehTambah={locId !== '' && locId !== '0'}
        onUbah={(pesan) => {
          resetPesan()
          setInfo(pesan)
          muat()
        }}
        onError={setError}
      />

      {editItem && (
        <FormEditBarang
          key={editItem.id}
          item={editItem}
          baki={baki.rows.filter((b) => b.location_id === editItem.location_id)}
          kapasitas={baki.kapasitas}
          onSelesai={(pesan) => {
            setInfo(pesan)
            setEditItem(null)
            selesai()
          }}
          onTutup={() => setEditItem(null)}
          onHapus={bisaVoid ? mulaiBatal : undefined}
        />
      )}

      {terpilih.length > 0 && (
        <BarAksiBarang
          terpilih={terpilih}
          beratTerpilih={beratTerpilih}
          mode={mode}
          isCabang={isCabang}
          adaLM={adaLM}
          aksiTersedia={aksiTersedia}
          aksi={aksi}
          onAksi={setAksi}
          cabangList={cabangList}
          onMulai={resetPesan}
          onSelesai={(pesan) => {
            setInfo(pesan)
            setAksi(null)
            setPilih({})
            selesai()
          }}
          onError={setError}
          onBatalPilih={() => {
            setPilih({})
            setAksi(null)
          }}
        />
      )}

      {bakiPindah.length > 0 && (
        <PindahBaki
          terpilih={terpilih}
          baki={bakiPindah}
          kapasitas={baki.kapasitas}
          onSelesai={(pesan) => {
            resetPesan()
            setInfo(pesan)
            setPilih({})
            selesai()
          }}
          onError={setError}
        />
      )}

      {pindahItem && (
        <PindahSatu
          item={pindahItem}
          baki={baki.rows.filter((b) => b.location_id === pindahItem.location_id)}
          kapasitas={baki.kapasitas}
          onSelesai={(pesan) => {
            resetPesan()
            setInfo(pesan)
            setPindahItem(null)
            selesai()
          }}
          onTutup={() => setPindahItem(null)}
          onError={setError}
        />
      )}

      {voidItem && (
        <Popup
          judul={`Hapus barang ${voidItem.item_no}?`}
          labelKonfirmasi="Ya, hapus barang"
          danger
          loading={loadingBatal}
          onKonfirmasi={konfirmasiBatal}
          onTutup={() => setVoidItem(null)}
        >
          <p className="muted">Barang dihapus dari stok (salah input). Periksa lagi spesifikasinya:</p>
          <SpesifikasiBarang item={voidItem} />
          <label style={{ display: 'block', marginTop: 12 }}>Alasan menghapus</label>
          <input
            value={alasan}
            onChange={(e) => setAlasan(e.target.value)}
            placeholder="Contoh: salah input berat"
          />
        </Popup>
      )}

      <div className="card" style={{ marginTop: 16 }}>

        <TabelBarang
          rows={data.rows}
          pilih={pilih}
          onToggle={(id) => setPilih((p) => toggleId(p, id))}
          semuaDipilih={semuaDipilih}
          onToggleSemua={toggleSemua}
          bisaVoid={bisaVoid}
          onBatal={mulaiBatal}
          onPindah={(r) => {
            resetPesan()
            setPindahItem(r)
          }}
          onEdit={(r) => {
            resetPesan()
            setEditItem(r)
          }}
        />
      </div>
    </div>
  )
}
