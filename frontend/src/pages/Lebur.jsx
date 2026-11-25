import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { awalBulan, hariIni, gram } from '../utils/format'
import Pesan from '../components/common/Pesan'
import FormAksiLebur from '../components/lebur/FormAksiLebur'
import TabelLebur from '../components/lebur/TabelLebur'
import AntrianLebur from '../components/lebur/AntrianLebur'
import PanelSiapJual from '../components/lebur/PanelSiapJual'
import RiwayatPenjualan from '../components/lebur/RiwayatPenjualan'
import FilterProduksi, { FILTER_PRODUKSI_KOSONG, paramProduksi } from '../components/common/FilterProduksi'

export default function Lebur({ me }) {
  const bisaCatat = me.role === 'pj' || me.role === 'super_admin'
  const bisaUndo = me.role === 'super_admin'
  const bisaKonfirmasi = me.role === 'owner' || me.role === 'super_admin'

  const [filter, setFilter] = useState('berjalan')
  const [fp, setFp] = useState(FILTER_PRODUKSI_KOSONG)
  const [tampilkanSeluruhBatch, setTampilkanSeluruhBatch] = useState(false)
  const [locations, setLocations] = useState([])
  const [batches, setBatches] = useState([])
  const [antrian, setAntrian] = useState([])
  const [siap, setSiap] = useState([])
  const [penj, setPenj] = useState({ rows: [], total_modal: 0, total_jual: 0, total_untung: 0 })
  const [dari, setDari] = useState(awalBulan())
  const [sampai, setSampai] = useState(hariIni())
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [aksi, setAksi] = useState(null)

  const muat = useCallback(() => {
    const p = paramProduksi(fp)
    p.set('status', filter)
    if (tampilkanSeluruhBatch) p.set('tampilkan_semua', 'true')
    api(`/lebur?${p}`).then(setBatches).catch((e) => setError(e.message))
    if (['pj', 'owner', 'super_admin'].includes(me.role)) {
      api('/lebur/antrian').then(setAntrian).catch((e) => setError(e.message))
    }
    api('/penjualan/siap').then(setSiap).catch((e) => setError(e.message))
    api(`/penjualan?from=${dari}&to=${sampai}`).then(setPenj).catch((e) => setError(e.message))
  }, [filter, fp, tampilkanSeluruhBatch, dari, sampai, me.role])

  useEffect(() => {
    muat()
  }, [muat])

  useEffect(() => {
    api('/locations').then(setLocations).catch(() => {})
  }, [])

  // ringkasan dari daftar yang tampil (mengikuti filter)
  const jumlah = (arr, f) => arr.reduce((s, x) => s + f(x), 0)
  const filterBarangAktif = Boolean(
    fp.q.trim() || fp.karat || Number(fp.berat_min) > 0 || Number(fp.berat_max) > 0 || fp.asal,
  )
  const totalBeratTampil = !tampilkanSeluruhBatch && filterBarangAktif
    ? jumlah(batches, (b) => jumlah(b.items, (it) => it.weight_gram))
    : jumlah(batches, (b) => b.source_weight)
  const sudahCukim = batches.filter((b) => b.cukim_weight !== null && b.cukim_weight !== undefined)
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

    {['pj', 'owner', 'super_admin'].includes(me.role) && (
      <AntrianLebur
        rows={antrian}
        locations={locations}
        role={me.role}
        onSelesai={(pesan) => {
          setInfo(pesan)
          setError('')
          muat()
        }}
        onError={setError}
      />
    )}

      <details className="card" style={{ marginTop: 16 }} open>
        <summary style={{ cursor: 'pointer', fontWeight: 700 }}>
          Proses Lebur ({batches.length} batch)
        </summary>
        <p className="muted">
          Owner memilih barang di Stok untuk dimasukkan ke antrean tanpa mengelompokkannya. PJ lalu memilih barang
          yang dijadikan satu batch, mencatat berat setelah dipotong dan leburan kotor, lalu cukim. Cukim dijual ke
          penadah di bagian bawah.
          Angka berat di tabel: awal → setelah dipotong → leburan kotor → cukim.
        </p>

        <FilterProduksi f={fp} onUbah={setFp} locations={locations} multipleKarat
          labelNama="Nama atau nomor barang" labelOrang="Dicatat oleh (leburan atau cukim)"
          aksiTambahan={
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, margin: 0 }}>
              <input
                type="checkbox"
                checked={tampilkanSeluruhBatch}
                onChange={(e) => setTampilkanSeluruhBatch(e.target.checked)}
              />
              Tampilkan seluruh batch
            </label>
          }>
          <div>
            <label>Status</label>
            <select value={filter} onChange={(e) => setFilter(e.target.value)}>
              <option value="berjalan">Sedang berjalan</option>
              <option value="terjual">Sudah terjual</option>
              <option value="batal">Dibatalkan</option>
              <option value="semua">Semua</option>
            </select>
          </div>
        </FilterProduksi>

        <div className="total-bar">
          <span>Total: <b className="utama">{batches.length}</b> batch, <b>{jumlah(batches, (b) => b.items.length)}</b> barang, <b>{gram(totalBeratTampil)}</b> g (berat awal)</span>
          {sudahCukim.length > 0 && (
            <span>Sudah cukim: {sudahCukim.length} batch, {gram(jumlah(sudahCukim, (b) => b.source_weight))} g jadi {gram(jumlah(sudahCukim, (b) => b.cukim_weight))} g, susut <b>{gram(jumlah(sudahCukim, (b) => b.source_weight - b.cukim_weight))}</b> g</span>
          )}
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
      </details>

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
        bisaKonfirmasi={bisaKonfirmasi}
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
