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
  const [tampilkanSeluruhBatch, setTampilkanSeluruhBatch] = useState(false)
  const [locations, setLocations] = useState([])
  const [rows, setRows] = useState([])
  const [items, setItems] = useState([])
  const [pilihKirim, setPilihKirim] = useState({})
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [aksi, setAksi] = useState(null)

  const muat = useCallback(() => {
    const p = paramProduksi(fp)
    p.set('status', filter)
    if (tampilkanSeluruhBatch) p.set('tampilkan_semua', 'true')
    api(`/cuci?${p}`).then(setRows).catch((e) => setError(e.message))
    if (!isCabang) {
      api('/cucian/barang').then(setItems).catch(() => {})
    }
  }, [filter, fp, tampilkanSeluruhBatch, isCabang])

  useEffect(() => {
    muat()
  }, [muat])

  useEffect(() => {
    if (!isCabang) api('/locations').then(setLocations).catch(() => {})
  }, [isCabang])

  function mulaiAksi(jenis, r) {
    setAksi({ jenis, id: r.id, sisa: r.sisa, qty: r.qty, n: Date.now() })
    setPilihKirim({})
    setError('')
    setInfo('')
  }

  const barangBelumKirim = rows.flatMap((r) =>
    r.items
      .filter((i) => i.di_pencuci && !i.sudah_dikirim)
      .map((i) => ({ ...i, job_id: r.id, weight_gram: i.berat_awal })),
  )
  const terpilihKirim = Object.values(pilihKirim)
  const semuaDipilih = barangBelumKirim.length > 0 && barangBelumKirim.every((i) => pilihKirim[i.id])

  function toggleKirim(jobId, item) {
    if (!pilihKirim[item.id] && terpilihKirim.length >= 300) {
      setError('Maksimal 300 barang sekali kirim')
      return
    }
    setPilihKirim((p) => {
      if (p[item.id]) {
        const n = { ...p }
        delete n[item.id]
        return n
      }
      return { ...p, [item.id]: { ...item, job_id: jobId, weight_gram: item.berat_awal } }
    })
  }

  function toggleSemuaKirim() {
    const belumTerpilih = barangBelumKirim.filter((i) => !pilihKirim[i.id])
    if (!semuaDipilih && terpilihKirim.length + belumTerpilih.length > 300) {
      setError('Maksimal 300 barang sekali kirim; dipilih hingga batas tersebut')
    }
    setPilihKirim((p) => {
      const n = { ...p }
      let sisa = 300 - Object.keys(p).length
      barangBelumKirim.forEach((i) => {
        if (semuaDipilih) delete n[i.id]
        else if (!p[i.id] && sisa > 0) {
          n[i.id] = i
          sisa--
        }
      })
      return n
    })
  }

  function mulaiKirim() {
    setAksi({ jenis: 'kirim', n: Date.now() })
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

  return (
    <div>
      <Pesan error={error} info={info} />

      <div className="card">
        <h3>Daftar Cuci</h3>
        <p className="muted">
          Owner memilih barang di menu Stok lalu klik Cuci. PJ memilih barang yang akan dikirim, menimbang tiap
          barang (berat awal), lalu mengirimnya ke pencuci. Saat barang kembali, PJ menimbang lagi (berat sesudah) dan mencatat susutnya. Barang yang
          selesai dikembalikan ke back office dengan berat terbaru, lalu owner mengonfirmasi sudah diterima.
          Kalau tidak jadi dicuci, PJ bisa membatalkan dan mengembalikan barang yang belum selesai.
        </p>

        <FilterProduksi f={fp} onUbah={setFp} locations={locations} tampilAsal={!isCabang}
          labelNama="Nama atau nomor barang" labelOrang="Pencuci"
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

        {bisaCatat && barangBelumKirim.length > 0 && (
          <label style={{ display: 'flex', alignItems: 'center', gap: 8, margin: '12px 0' }}>
            <input type="checkbox" checked={semuaDipilih} onChange={toggleSemuaKirim} />
            Pilih semua barang yang menunggu dikirim
          </label>
        )}
        {bisaCatat && barangBelumKirim.length > 0 && (
          <p className="muted">Maksimal 300 barang dalam satu kali pengiriman.</p>
        )}

        {bisaCatat && terpilihKirim.length > 0 && (
          <div className="card action-bar" style={{ margin: '12px 0' }}>
            <span><b>{terpilihKirim.length}</b> barang dipilih</span>
            <div className="aksi-tombol">
              <button type="button" className="mini primary" onClick={mulaiKirim}>
                Kirim ke pencuci
              </button>
              <button type="button" className="mini" onClick={() => setPilihKirim({})}>
                Batal pilih
              </button>
            </div>
          </div>
        )}

        {aksi && (
          <FormAksiCuci
            key={aksi.n}
            aksi={aksi}
            itemJob={aksi.jenis === 'kirim' ? terpilihKirim : []}
            onMulai={resetPesan}
            onSelesai={(pesan) => {
              setInfo(pesan)
              setAksi(null)
              setPilihKirim({})
              muat()
            }}
            onError={setError}
            onTutup={() => setAksi(null)}
          />
        )}

        <TabelDaftarCuci
          rows={rows}
          bisaCatat={bisaCatat}
          bisaVoid={bisaVoid}
          pilihKirim={pilihKirim}
          batasPilihTercapai={terpilihKirim.length >= 300}
          onToggleKirim={toggleKirim}
          onAksi={mulaiAksi}
        />
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
