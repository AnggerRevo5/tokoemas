import { useState } from 'react'
import { api } from '../../services/api'
import { hariIni, gram, toggleId } from '../../utils/format'
import { STATUS_CUCI } from '../../constants/cuci'
import { AnimatePresence } from 'motion/react'
import useAnimasiDaftar from '../../hooks/useAnimasiDaftar'
import AngkaBerjalan from '../common/AngkaBerjalan'
import BarisAnimasi from '../common/BarisAnimasi'
import BarisKosong from '../common/BarisKosong'
import Spinner from '../common/Spinner'

const formatGram = (v) => String(gram(v))

// Kartu "Barang di Pencuci": daftar barang, pilih yang sudah kembali, isi berat sesudah, lalu selesai cuci.
// onMulai: bersihkan pesan; onSelesai(message): setelah tersimpan; onError(msg).
export default function BarangDiPencuci({ items, bisaCatat, onMulai, onSelesai, onError }) {
  const [loading, setLoading] = useState(false)
  const [pilih, setPilih] = useState({})
  const [tglItem, setTglItem] = useState(hariIni())
  const [bSesudah, setBSesudah] = useState({})
  const [cari, setCari] = useState('')

  const tampilItems = items.filter((r) => {
    const s = cari.trim().toLowerCase()
    if (!s) return true
    return (
      r.item_name.toLowerCase().includes(s) ||
      (r.washer_name || '').toLowerCase().includes(s) ||
      (r.item_no || '').toLowerCase().includes(s)
    )
  })
  const bisaDipilih = tampilItems.filter((r) => r.job_status === 'proses')
  const terpilih = items.filter((r) => pilih[r.id] && r.job_status === 'proses')
  const nilaiSesudah = (r) => (bSesudah[r.id] !== undefined ? bSesudah[r.id] : String(gram(r.berat_awal)))
  const totalAwal = terpilih.reduce((s, r) => s + r.berat_awal, 0)
  const totalSesudah = terpilih.reduce((s, r) => s + (Number(nilaiSesudah(r)) || 0), 0)
  const semuaDipilih = bisaDipilih.length > 0 && bisaDipilih.every((r) => pilih[r.id])
  const animasi = useAnimasiDaftar(tampilItems.map((r) => r.id))

  function toggleSemua() {
    if (semuaDipilih) {
      setPilih({})
    } else {
      const n = {}
      bisaDipilih.forEach((r) => (n[r.id] = true))
      setPilih(n)
    }
  }

  async function selesaiBarang(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      const res = await api('/cucian/selesai', {
        method: 'POST',
        body: JSON.stringify({
          finish_date: tglItem,
          items: terpilih.map((r) => ({ item_id: r.id, weight_after: Number(nilaiSesudah(r)) })),
        }),
      })
      setPilih({})
      setBSesudah({})
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card" style={{ marginTop: 16 }}>
      <h3>Barang di Pencuci</h3>
      <p className="muted">
        {bisaCatat
          ? 'Centang barang yang sudah dikembalikan pencuci, isi berat sesudah cuci (susut tercatat otomatis), lalu klik Selesai cuci. Barang yang belum dikirim ke pencuci belum bisa dipilih.'
          : 'Daftar barang yang sedang dalam proses cuci.'}
      </p>

      <div className="form-grid">
        <div>
          <label>Cari barang atau pencuci</label>
          <input value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Cincin, Adip..." />
        </div>
      </div>

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {bisaCatat && (
                <th>
                  <input type="checkbox" checked={semuaDipilih} onChange={toggleSemua} />
                </th>
              )}
              <th>No</th>
              <th>Barang</th>
              <th className="num">Berat awal</th>
              <th>Asal</th>
              <th>Pencuci</th>
              <th>Cuci</th>
            </tr>
          </thead>
          <tbody>
            {tampilItems.length === 0 && <BarisKosong kolom="7">Tidak ada barang yang sedang dicuci</BarisKosong>}
            <AnimatePresence initial={false} custom={animasi.instan}>
            {tampilItems.map((r) => (
              <BarisAnimasi key={r.id} layout={animasi.layout}>
                {bisaCatat && (
                  <td>
                    <input
                      type="checkbox"
                      checked={!!pilih[r.id]}
                      disabled={r.job_status !== 'proses'}
                      onChange={() => setPilih((p) => toggleId(p, r.id))}
                    />
                  </td>
                )}
                <td>{r.item_no}</td>
                <td>{r.item_name} {r.karat}</td>
                <td className="num">{r.job_status === 'proses' ? `${gram(r.berat_awal)} g` : <span className="muted">belum ditimbang</span>}</td>
                <td>{r.asal}</td>
                <td>{r.washer_name || <span className="muted">Belum dikirim</span>}</td>
                <td>
                  <span className={STATUS_CUCI[r.job_status][1]}>{STATUS_CUCI[r.job_status][0]}</span>
                  <div className="muted">#{r.job_id}</div>
                </td>
              </BarisAnimasi>
            ))}
            </AnimatePresence>
          </tbody>
        </table>
      </div>

      {bisaCatat && terpilih.length > 0 && (
        <form className="panel-aksi" onSubmit={selesaiBarang}>
          <div>
            <b>{terpilih.length}</b> barang dipilih. Timbang tiap barang dan isi berat sesudah cuci:
          </div>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Barang</th>
                  <th className="num">Berat awal</th>
                  <th>Berat sesudah (g)</th>
                  <th className="num">Susut</th>
                </tr>
              </thead>
              <tbody>
                {terpilih.map((r) => {
                  const susut = r.berat_awal - (Number(nilaiSesudah(r)) || 0)
                  return (
                    <tr key={r.id}>
                      <td>{r.item_no} {r.item_name} {r.karat}</td>
                      <td className="num">{gram(r.berat_awal)} g</td>
                      <td>
                        <input
                          type="number"
                          step="0.001"
                          min="0.001"
                          max={r.berat_awal}
                          value={nilaiSesudah(r)}
                          onChange={(e) => setBSesudah((p) => ({ ...p, [r.id]: e.target.value }))}
                          required
                        />
                      </td>
                      <td className={susut < 0 ? 'num neg' : 'num'}>{gram(susut)} g</td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
          <div className="total-bar">
            <span>
              Total: <b><AngkaBerjalan nilai={totalAwal} format={formatGram} /></b> g jadi{' '}
              <b className="utama"><AngkaBerjalan nilai={totalSesudah} format={formatGram} /></b> g, susut{' '}
              <b><AngkaBerjalan nilai={totalAwal - totalSesudah} format={formatGram} /></b> g.
            </span>
          </div>
          <p className="muted" style={{ marginTop: 8 }}>
            Barang dikembalikan ke back office dengan berat terbaru dan menunggu konfirmasi owner.
          </p>
          <div className="form-grid">
            <div>
              <label>Tanggal selesai</label>
              <input type="date" value={tglItem} onChange={(e) => setTglItem(e.target.value)} required />
            </div>
          </div>
          <button disabled={loading}>{loading ? <><Spinner />Memproses...</> : 'Selesai cuci'}</button>
        </form>
      )}
    </div>
  )
}
