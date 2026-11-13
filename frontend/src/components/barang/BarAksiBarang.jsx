import { useState } from 'react'
import { api } from '../../services/api'
import { gram, hariIni, rupiah } from '../../utils/format'
import { KETERANGAN_AKSI, BP_LIST } from '../../constants/barang'
import PilihMetodeBayar from '../common/PilihMetodeBayar'
import { BIAYA_KOSONG, nominalBiaya } from '../../utils/biaya'
import { bolehGratis, tanpaGram } from '../../utils/barang'
import { m } from 'motion/react'
import AngkaBerjalan from '../common/AngkaBerjalan'
import Spinner from '../common/Spinner'
import FormGadai from './FormGadai'
import Popup from '../common/Popup'
import KartuAksi from './KartuAksi'

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
  const [galat, setGalat] = useState('') // pesan galat ditampilkan di dalam card, karena halaman tertutup latar
  const mulai = () => {
    setGalat('')
    onMulai()
  }
  const gagal = (msg) => {
    setGalat(msg)
    onError(msg)
  }
  const judulAksi = (aksiTersedia.find(([k]) => k === aksi) || [])[1] || ''
  const totalJual = terpilih.reduce((s, r) => s + (Number(harga[r.id]) || 0), 0)
  // harga jual wajib > 0, kecuali tempat/kotak perhiasan yang dicentang Gratis (harga '0')
  const hargaKosong = terpilih.some((r) => !(Number(harga[r.id]) > 0) && !(bolehGratis(r.jenis) && harga[r.id] === '0'))
  const beratTeks = (r) => (tanpaGram(r.jenis) && !(r.weight_gram > 0) ? '-' : `${gram(r.weight_gram)} g`)

  // tujuan oper: cabang pertama bila belum dipilih
  const targetAktif = target || (cabangList.length > 0 ? String(cabangList[0].id) : '')

  // penjualan: tampilkan card konfirmasi (barang, harga, metode) sebelum benar-benar tersimpan
  function periksaJual(e) {
    e.preventDefault()
    mulai()
    if (hargaKosong) return gagal('Harga jual setiap barang wajib diisi')
    if (metode === 'qris' && nominalBiaya(totalJual, biaya) > totalJual) {
      return gagal('Potongan tidak boleh melebihi total penjualan')
    }
    setKonfirmasiJual(true)
  }

  async function jalankan(e) {
    e.preventDefault()
    mulai()
    setLoading(true)
    const ids = terpilih.map((r) => r.id)
    try {
      if (aksi === 'jual' && hargaKosong) {
        throw new Error('Harga jual setiap barang wajib diisi')
      }
      if (aksi === 'jual' && metode === 'qris' && nominalBiaya(totalJual, biaya) > totalJual) {
        throw new Error('Potongan tidak boleh melebihi total penjualan')
      }
      let res
      if (aksi === 'kirim') {
        const keBP = targetAktif.startsWith('bp:')
        res = await api('/barang/oper', {
          method: 'POST',
          body: JSON.stringify({
            item_ids: ids,
            date: tgl,
            ...(keBP ? { bp_nama: targetAktif.slice(3) } : { target_location_id: Number(targetAktif) }),
          }),
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
      } else if (aksi === 'tarik_bp') {
        res = await api('/barang/tarik-bp', { method: 'POST', body: JSON.stringify({ item_ids: ids, date: tgl }) })
      } else if (aksi === 'serah' || aksi === 'batal_kirim') {
        res = await api(aksi === 'serah' ? '/barang/serah-kurir' : '/barang/batal-kirim', {
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
      gagal(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
    <m.div
      className="card action-bar melayang"
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
      {mode === 'di_bp' && isCabang && (
        <div className="muted">Barang ini sedang berada di BP. Penarikan kembali dilakukan owner.</div>
      )}
      {mode === 'siap_toko' && isCabang && (
        <div className="muted">Back office masih menyiapkan kiriman ini. Konfirmasi bisa setelah diserahkan ke kurir.</div>
      )}
      {mode === 'transit' && isCabang && (
        <div className="muted">Menunggu konfirmasi dari back office.</div>
      )}
      {adaLM && mode === 'back_office' && (
        <div className="muted">
          Logam mulia hanya bisa dioper ke toko atau BP. Untuk dijual ke penadah, PJ mencatatnya lewat menu Lebur, lalu owner mengonfirmasi di sana.
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
        <KartuAksi
          judul={judulAksi}
          galat={galat}
          onTutup={() => {
            setGalat('')
            onAksi(null)
            onMulai()
          }}
        >
      {aksi === 'gadai' && (
        <FormGadai items={terpilih} onMulai={mulai} onSelesai={onSelesai} onError={gagal} />
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
                <label>Oper ke</label>
                <select value={targetAktif} onChange={(e) => setTarget(e.target.value)}>
                  <optgroup label="Toko">
                    {cabangList.map((l) => (
                      <option key={l.id} value={String(l.id)}>{l.name}</option>
                    ))}
                  </optgroup>
                  <optgroup label="BP (tanpa akun)">
                    {BP_LIST.map((b) => (
                      <option key={b} value={`bp:${b}`}>{b}</option>
                    ))}
                  </optgroup>
                </select>
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
                      <td className="num">{beratTeks(r)}</td>
                      <td className="num">{r.harga_beli > 0 ? rupiah(r.harga_beli) : '-'}</td>
                      <td>
                        <input
                          type="number"
                          min="1"
                          value={harga[r.id] === '0' ? '' : harga[r.id] ?? ''}
                          onChange={(e) => setHarga((p) => ({ ...p, [r.id]: e.target.value }))}
                          disabled={harga[r.id] === '0'}
                          placeholder={harga[r.id] === '0' ? 'Gratis' : undefined}
                          aria-label={`Harga jual ${r.item_no}`}
                          required={harga[r.id] !== '0'}
                        />
                        {bolehGratis(r.jenis) && (
                          <label className="muted" style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 4 }}>
                            <input
                              type="checkbox"
                              checked={harga[r.id] === '0'}
                              onChange={(e) => setHarga((p) => ({ ...p, [r.id]: e.target.checked ? '0' : '' }))}
                            />
                            Gratis
                          </label>
                        )}
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
        </KartuAksi>
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
                    <td className="num">{beratTeks(r)}</td>
                    <td className="num">{harga[r.id] === '0' ? 'Gratis' : rupiah(Number(harga[r.id]) || 0)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <table className="tabel-spek" style={{ marginTop: 12 }}>
            <tbody>
              <tr><th>Tanggal</th><td>{tgl}</td></tr>
              <tr><th>Metode</th><td>{metode === 'qris' ? 'QRIS/Debit (masuk kas back office)' : metode === 'transfer' ? 'Transfer (masuk kas toko)' : 'Tunai (masuk kas toko)'}</td></tr>
              <tr><th>Total harga jual</th><td>{rupiah(totalJual)}</td></tr>
              {metode === 'qris' && <tr><th>Potongan</th><td>{rupiah(nominalBiaya(totalJual, biaya))}</td></tr>}
            </tbody>
          </table>
        </Popup>
      )}
    </m.div>
    </>
  )
}
