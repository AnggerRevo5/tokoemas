import { useState } from 'react'
import { api } from '../../services/api'
import { gram, hariIni, rupiah } from '../../utils/format'
import { KETERANGAN_AKSI } from '../../constants/barang'
import PilihanLokasi from '../common/PilihanLokasi'
import { m } from 'motion/react'
import AngkaBerjalan from '../common/AngkaBerjalan'
import Spinner from '../common/Spinner'

const formatGram = (v) => String(gram(v))
const formatRupiah = (v) => rupiah(Math.round(v))

// Bar aksi untuk barang yang dicentang: tombol langkah dan form konfirmasinya.
// aksi/onAksi dipegang halaman. onMulai: bersihkan pesan; onSelesai(message); onError(msg); onBatalPilih().
export default function BarAksiBarang({
  terpilih, beratTerpilih, mode, isCabang, adaLM, aksiTersedia, aksi, onAksi, cabangList,
  onMulai, onSelesai, onError, onBatalPilih,
}) {
  const [loading, setLoading] = useState(false)
  const [tgl, setTgl] = useState(hariIni())
  const [target, setTarget] = useState('')
  const [catatan, setCatatan] = useState('')
  const [harga, setHarga] = useState({}) // harga jual per barang { id: 'nominal' }

  // tujuan oper: cabang pertama bila belum dipilih
  const targetAktif = target || (cabangList.length > 0 ? String(cabangList[0].id) : '')

  async function jalankan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    const ids = terpilih.map((r) => r.id)
    try {
      if (aksi === 'jual' && terpilih.some((r) => !(Number(harga[r.id]) > 0))) {
        throw new Error('Harga jual setiap barang wajib diisi')
      }
      let res
      if (aksi === 'kirim') {
        res = await api('/barang/oper', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, target_location_id: Number(targetAktif), date: tgl }),
        })
      } else if (aksi === 'terima_toko' || aksi === 'belum_toko') {
        res = await api('/barang/terima-toko', {
          method: 'POST',
          body: JSON.stringify({
            item_ids: ids,
            result: aksi === 'terima_toko' ? 'sampai' : 'belum',
            date: tgl,
            note: catatan,
          }),
        })
      } else if (aksi === 'antar') {
        res = await api('/barang/kirim-bo', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, date: tgl }),
        })
      } else if (aksi === 'sampai') {
        res = await api('/barang/terima', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, result: 'sampai', date: tgl }),
        })
      } else if (aksi === 'belum') {
        res = await api('/barang/terima', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, result: 'belum', date: tgl, note: catatan }),
        })
      } else if (aksi === 'terima_kembali' || aksi === 'belum_kembali') {
        res = await api('/barang/terima-kembali', {
          method: 'POST',
          body: JSON.stringify({
            item_ids: ids,
            result: aksi === 'terima_kembali' ? 'sampai' : 'belum',
            date: tgl,
            note: catatan,
          }),
        })
      } else if (aksi === 'cuci') {
        res = await api('/cuci', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, start_date: tgl }),
        })
      } else if (aksi === 'lebur') {
        res = await api('/lebur', {
          method: 'POST',
          body: JSON.stringify({ item_ids: ids, start_date: tgl, note: catatan }),
        })
      } else {
        res = await api('/barang/jual', {
          method: 'POST',
          body: JSON.stringify({
            items: terpilih.map((r) => ({ item_id: r.id, harga: Number(harga[r.id]) })),
            date: tgl,
          }),
        })
      }
      setCatatan('')
      setHarga({})
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <m.div
      className="card action-bar"
      style={{ marginTop: 16 }}
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: 'easeOut' }}
    >
      <div>
        <b>{terpilih.length}</b> barang dipilih (<AngkaBerjalan nilai={beratTerpilih} format={formatGram} durasi={0.4} /> gram)
      </div>
      {mode === 'campur' && (
        <div className="muted">Pilih barang dengan status, lokasi, dan arah kiriman yang sama.</div>
      )}
      {mode === 'transit_toko' && !isCabang && (
        <div className="muted">Barang sedang menuju toko. Konfirmasi diterima hanya oleh akun toko tujuan.</div>
      )}
      {mode === 'transit' && isCabang && (
        <div className="muted">Menunggu konfirmasi dari back office.</div>
      )}
      {adaLM && mode === 'back_office' && (
        <div className="muted">
          Logam mulia hanya bisa dikirim ke toko. Untuk dijual ke penadah, PJ menjualnya lewat menu Lebur.
        </div>
      )}

      <div className="aksi-tombol">
        {aksiTersedia.map(([k, l]) => (
          <button
            key={k}
            type="button"
            className={aksi === k ? 'mini primary-fill' : 'mini primary'}
            onClick={() => {
              onAksi(k)
              onMulai()
            }}
          >
            {l}
          </button>
        ))}
        <button type="button" className="mini" onClick={onBatalPilih}>
          Batal pilih
        </button>
      </div>

      {aksi && (
        <form className="panel-aksi" onSubmit={jalankan}>
          <div className="form-grid">
            <div>
              <label>{aksi === 'selesai_cuci' ? 'Tanggal selesai' : 'Tanggal'}</label>
              <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
            </div>

            {aksi === 'kirim' && (
              <div>
                <label>Oper ke toko</label>
                <PilihanLokasi locations={cabangList} value={targetAktif} onChange={(e) => setTarget(e.target.value)} />
              </div>
            )}

            {(aksi === 'belum' || aksi === 'belum_toko' || aksi === 'belum_kembali' || aksi === 'lebur') && (
              <div className="span2">
                <label>Catatan (opsional)</label>
                <input
                  value={catatan}
                  onChange={(e) => setCatatan(e.target.value)}
                  placeholder={
                    aksi === 'lebur'
                      ? 'Contoh: ada permata, dilepas dulu'
                      : aksi === 'belum_toko'
                        ? 'Contoh: belum ada kurir yang datang'
                        : 'Contoh: masih di kurir'
                  }
                />
              </div>
            )}
          </div>

          {aksi === 'jual' && (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Barang</th>
                    <th className="num">Berat</th>
                    <th className="num">Harga beli</th>
                    <th>Harga jual (Rp)</th>
                  </tr>
                </thead>
                <tbody>
                  {terpilih.map((r) => (
                    <tr key={r.id}>
                      <td>{r.item_no} {r.item_name} {r.karat}</td>
                      <td className="num">{gram(r.weight_gram)} g</td>
                      <td className="num">{r.harga_beli > 0 ? rupiah(r.harga_beli) : '-'}</td>
                      <td>
                        <input
                          type="number"
                          min="1"
                          value={harga[r.id] ?? ''}
                          onChange={(e) => setHarga((p) => ({ ...p, [r.id]: e.target.value }))}
                          required
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <div className="total-bar">
                <span>
                  Total harga jual (masuk ke kas toko):{' '}
                  <b className="utama">
                    <AngkaBerjalan nilai={terpilih.reduce((s, r) => s + (Number(harga[r.id]) || 0), 0)} format={formatRupiah} />
                  </b>
                </span>
              </div>
            </div>
          )}

          {KETERANGAN_AKSI[aksi] && <p className="muted">{KETERANGAN_AKSI[aksi]}</p>}

          <button disabled={loading}>{loading ? <><Spinner />Memproses...</> : 'Konfirmasi'}</button>
        </form>
      )}
    </m.div>
  )
}
