import { useState } from 'react'
import { api } from '../../services/api'
import { gram, hariIni, rupiah } from '../../utils/format'
import { KETERANGAN_AKSI } from '../../constants/barang'
import PilihanLokasi from '../common/PilihanLokasi'
import PilihMetodeBayar from '../common/PilihMetodeBayar'
import { BIAYA_KOSONG, nominalBiaya } from '../../utils/biaya'
import { m } from 'motion/react'
import AngkaBerjalan from '../common/AngkaBerjalan'
import Spinner from '../common/Spinner'
import FormGadai from './FormGadai'
import Popup from '../common/Popup'

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
  const [metode, setMetode] = useState('tunai')
  const [biaya, setBiaya] = useState(BIAYA_KOSONG)
  const [konfirmasiJual, setKonfirmasiJual] = useState(false)
  const totalJual = terpilih.reduce((s, r) => s + (Number(harga[r.id]) || 0), 0)

  // tujuan oper: cabang pertama bila belum dipilih
  const targetAktif = target || (cabangList.length > 0 ? String(cabangList[0].id) : '')

  // penjualan: tampilkan card konfirmasi (barang, harga, metode) sebelum benar-benar tersimpan
  function periksaJual(e) {
    e.preventDefault()
    onMulai()
    if (terpilih.some((r) => !(Number(harga[r.id]) > 0))) return onError('Harga jual setiap barang wajib diisi')
    if (metode === 'qris' && nominalBiaya(totalJual, biaya) > totalJual) {
      return onError('Potongan tidak boleh melebihi total penjualan')
    }
    setKonfirmasiJual(true)
  }

  async function jalankan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    const ids = terpilih.map((r) => r.id)
    try {
      if (aksi === 'jual' && terpilih.some((r) => !(Number(harga[r.id]) > 0))) {
        throw new Error('Harga jual setiap barang wajib diisi')
      }
      if (aksi === 'jual' && metode === 'qris' && nominalBiaya(totalJual, biaya) > totalJual) {
        throw new Error('Potongan tidak boleh melebihi total penjualan')
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
            metode,
            biaya: metode === 'qris' ? nominalBiaya(totalJual, biaya) : 0,
          }),
        })
      }
      setCatatan('')
      setHarga({})
      setMetode('tunai')
      setBiaya(BIAYA_KOSONG)
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

      {aksi === 'gadai' && (
        <FormGadai item={terpilih[0]} onMulai={onMulai} onSelesai={onSelesai} onError={onError} />
      )}

      {aksi && aksi !== 'gadai' && (
        <form className="panel-aksi" onSubmit={aksi === 'jual' ? periksaJual : jalankan}>
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
                  Total harga jual ({metode === 'qris' ? 'masuk ke kas back office' : 'masuk ke kas toko'}):{' '}
                  <b className="utama">
                    <AngkaBerjalan nilai={totalJual} format={formatRupiah} />
                  </b>
                </span>
              </div>
            </div>
          )}
          {aksi === 'jual' && (
            <PilihMetodeBayar
              metode={metode}
              onMetode={setMetode}
              total={totalJual}
              biaya={biaya}
              onBiaya={setBiaya}
            />
          )}

          {KETERANGAN_AKSI[aksi] && <p className="muted">{KETERANGAN_AKSI[aksi]}</p>}

          <button disabled={loading}>{loading ? <><Spinner />Memproses...</> : aksi === 'jual' ? 'Lanjut' : 'Konfirmasi'}</button>
        </form>
      )}

      {konfirmasiJual && (
        <Popup
          judul={`Jual ${terpilih.length} barang?`}
          labelKonfirmasi="Ya, catat penjualan"
          loading={loading}
          onKonfirmasi={async (e) => {
            await jalankan(e)
            setKonfirmasiJual(false)
          }}
          onTutup={() => setKonfirmasiJual(false)}
        >
          <p className="muted">Periksa barang dan harga sebelum disimpan:</p>
          <div className="table-wrap" style={{ maxHeight: 260, overflow: 'auto' }}>
            <table className="tabel-spek">
              <thead>
                <tr><th>Barang</th><th>Kadar</th><th className="num">Berat</th><th className="num">Harga jual</th></tr>
              </thead>
              <tbody>
                {terpilih.map((r) => (
                  <tr key={r.id}>
                    <td>{r.item_no} {r.item_name}</td>
                    <td>{r.karat}</td>
                    <td className="num">{gram(r.weight_gram)} g</td>
                    <td className="num">{rupiah(Number(harga[r.id]) || 0)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <table className="tabel-spek" style={{ marginTop: 12 }}>
            <tbody>
              <tr><th>Tanggal</th><td>{tgl}</td></tr>
              <tr><th>Metode</th><td>{metode === 'qris' ? 'QRIS/Debit (masuk kas back office)' : 'Tunai (masuk kas toko)'}</td></tr>
              <tr><th>Total harga jual</th><td>{rupiah(totalJual)}</td></tr>
              {metode === 'qris' && <tr><th>Potongan</th><td>{rupiah(nominalBiaya(totalJual, biaya))}</td></tr>}
            </tbody>
          </table>
        </Popup>
      )}
    </m.div>
  )
}
