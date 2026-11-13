import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { gram } from '../utils/format'
import Pesan from '../components/common/Pesan'
import FormAksiCuci from '../components/cuci/FormAksiCuci'
import TabelDaftarCuci from '../components/cuci/TabelDaftarCuci'
import BarangDiPencuci from '../components/cuci/BarangDiPencuci'
import FilterProduksi, { FILTER_PRODUKSI_KOSONG, paramProduksi } from '../components/common/FilterProduksi'

export default function Cuci({ me }) {
  const isCabang = me.role === 'cabang'
  const bisaCatat = me.role === 'pj' || me.role === 'super_admin'
  const bisaVoid = me.role === 'super_admin'

  const [filter, setFilter] = useState('aktif')
  const [fp, setFp] = useState(FILTER_PRODUKSI_KOSONG)
  const [locations, setLocations] = useState([])
  const [rows, setRows] = useState([])
  const [items, setItems] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [aksi, setAksi] = useState(null)

  const muat = useCallback(() => {
    const p = paramProduksi(fp)
    p.set('status', filter)
    api(`/cuci?${p}`).then(setRows).catch((e) => setError(e.message))
    if (!isCabang) {
      api('/cucian/barang').then(setItems).catch(() => {})
    }
  }, [filter, fp, isCabang])

  useEffect(() => {
    muat()
  }, [muat])

  useEffect(() => {
    if (!isCabang) api('/locations').then(setLocations).catch(() => {})
  }, [isCabang])

  function mulaiAksi(jenis, r) {
    setAksi({ jenis, id: r.id, sisa: r.sisa, qty: r.qty, n: Date.now() })
    setError('')
    setInfo('')
  }

  function resetPesan() {
    setError('')
    setInfo('')
  }

  // ringkasan dari daftar yang tampil (mengikuti filter); berat awal = berat sebelum cuci
  const barangCuci = rows.flatMap((r) => r.items)
  const ditimbang = barangCuci.filter((i) => i.berat_sesudah !== null)
  const jumlah = (arr, f) => arr.reduce((s, x) => s + f(x), 0)
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

        <FilterProduksi f={fp} onUbah={setFp} locations={locations} tampilAsal={!isCabang}
          labelNama="Nama atau nomor barang" labelOrang="Pencuci">
          <div>
            <label>Status</label>
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="aktif">Belum selesai</option>
              <option value="menunggu">Menunggu PJ</option>
              <option value="proses">Proses</option>
              <option value="selesai">Selesai</option>
              <option value="batal">Dibatalkan</option>
              <option value="semua">Semua</option>
            </select>
          </div>
        </FilterProduksi>

        <div className="total-bar">
          <span>Total: <b className="utama">{barangCuci.length}</b> barang di {rows.length} pekerjaan, <b>{gram(jumlah(barangCuci, (i) => i.berat_awal))}</b> g (berat awal)</span>
          {ditimbang.length > 0 && (
            <span>Sudah ditimbang ulang: {ditimbang.length} barang, {gram(jumlah(ditimbang, (i) => i.berat_awal))} g jadi {gram(jumlah(ditimbang, (i) => i.berat_sesudah))} g, susut <b>{gram(jumlah(ditimbang, (i) => i.berat_awal - i.berat_sesudah))}</b> g</span>
          )}
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
          fp={fp}
          statusF={filter}
          locations={locations}
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
