import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { m } from 'motion/react'
import { X } from 'lucide-react'
import { api } from '../../services/api'
import { hariIni, rupiah } from '../../utils/format'
import { BIAYA_KOSONG, nominalBiaya } from '../../utils/biaya'
import Popup from '../common/Popup'
import Pesan from '../common/Pesan'
import PilihMetodeBayar from '../common/PilihMetodeBayar'
import Spinner from '../common/Spinner'
import SpesifikasiBarang from './SpesifikasiBarang'

const BATAS_HARI = 7
const LABEL_METODE = { qris: 'QRIS/Debit', transfer: 'Transfer' }
const labelMetode = (m) => LABEL_METODE[m] || 'Tunai'

// true bila tanggal (YYYY-MM-DD) masih dalam batas hari yang boleh diubah akun cabang
function dalamBatas(tgl) {
  if (!tgl) return false
  const batas = new Date()
  batas.setDate(batas.getDate() - BATAS_HARI)
  return tgl >= batas.toLocaleDateString('en-CA')
}

// Card penjualan satu barang yang sudah terjual dari Stok: batalkan atau koreksi (harga, tanggal, metode, potongan).
// Akun cabang hanya untuk lokasinya sendiri dan penjualan paling lama 7 hari; Super Admin dan Owner tanpa batas.
// Setiap langkah menampilkan card konfirmasi sebelum disimpan. onUbah(pesan): data berubah; onTutup().
export default function CardPenjualan({ item, me, onUbah, onTutup }) {
  const tglLama = item.tanggal_jual ? item.tanggal_jual.slice(0, 10) : ''
  const metodeLama = item.metode || 'tunai'
  const [aksi, setAksi] = useState(null) // 'koreksi' | 'batal'
  const [harga, setHarga] = useState(String(item.harga_jual || ''))
  const [tgl, setTgl] = useState(tglLama || hariIni())
  const [metode, setMetode] = useState(metodeLama)
  const [biaya, setBiaya] = useState(BIAYA_KOSONG)
  const [alasan, setAlasan] = useState('')
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const hargaBaru = Number(harga) || 0
  const potongan = metode === 'qris' ? nominalBiaya(hargaBaru, biaya) : 0

  // alasan barang ini tidak bisa diubah oleh akun ini (kosong = boleh)
  let terkunci = ''
  if (!item.metode) {
    terkunci = 'Penjualan ini dicatat tanpa catatan kas, jadi tidak bisa dibatalkan otomatis.'
  } else if (me.role === 'cabang' && (item.location_id !== me.location_id || !dalamBatas(tglLama))) {
    terkunci = `Penjualan lebih dari ${BATAS_HARI} hari lalu hanya bisa diubah Super Admin atau Owner.`
  } else if (me.role !== 'cabang' && me.role !== 'super_admin' && me.role !== 'owner') {
    terkunci = 'Akun ini tidak bisa mengubah penjualan.'
  }

  useEffect(() => {
    const esc = (e) => e.key === 'Escape' && !konfirmasi && !loading && onTutup()
    window.addEventListener('keydown', esc)
    return () => window.removeEventListener('keydown', esc)
  }, [onTutup, konfirmasi, loading])

  function periksa(e) {
    e.preventDefault()
    setError('')
    if (!alasan.trim()) return setError('Alasan wajib diisi')
    if (aksi === 'koreksi') {
      if (hargaBaru <= 0) return setError('Harga jual wajib diisi')
      if (metode === 'qris' && potongan > hargaBaru) return setError('Potongan tidak boleh melebihi harga jual')
    }
    setKonfirmasi(true)
  }

  async function simpan() {
    setLoading(true)
    try {
      const res = aksi === 'batal'
        ? await api(`/jual-toko/${item.id}/batal`, { method: 'POST', body: JSON.stringify({ reason: alasan.trim() }) })
        : await api(`/jual-toko/${item.id}/koreksi`, {
          method: 'POST',
          body: JSON.stringify({ harga: hargaBaru, date: tgl, metode, biaya: potongan, reason: alasan.trim() }),
        })
      onUbah(res.message)
      onTutup()
    } catch (err) {
      setKonfirmasi(false)
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return createPortal(
    <div
      className="popup-latar latar-lebar"
      onMouseDown={(e) => e.target === e.currentTarget && !konfirmasi && !loading && onTutup()}
    >
      <m.div
        className="popup popup-lebar"
        role="dialog"
        aria-modal="true"
        aria-label={`Penjualan ${item.item_no}`}
        initial={{ opacity: 0, scale: 0.97, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.18, ease: 'easeOut' }}
      >
        <div className="popup-kepala">
          <h3>Penjualan {item.item_no} - {item.item_name} {item.karat}</h3>
          <button type="button" className="mini popup-tutup" onClick={onTutup} disabled={loading} aria-label="Tutup">
            <X size={16} aria-hidden="true" />
          </button>
        </div>
        <Pesan error={error} />

        <table className="tabel-spek" style={{ marginTop: 12 }}>
          <tbody>
            <tr><th>Harga jual</th><td>{rupiah(item.harga_jual || 0)}</td></tr>
            <tr><th>Tanggal jual</th><td>{tglLama || '-'}</td></tr>
            <tr><th>Metode</th><td>{item.metode ? labelMetode(item.metode) : '-'}</td></tr>
            <tr><th>Harga beli</th><td>{item.harga_beli > 0 ? rupiah(item.harga_beli) : '-'}</td></tr>
            <tr><th>Lokasi</th><td>{item.lokasi}</td></tr>
          </tbody>
        </table>
        {terkunci && <p className="muted">{terkunci}</p>}

        {!terkunci && !aksi && (
          <div className="aksi-tombol" style={{ marginTop: 14 }}>
            <button type="button" className="mini primary-fill" onClick={() => { setError(''); setAksi('koreksi') }}>
              Edit penjualan
            </button>
            <button type="button" className="mini danger-fill" onClick={() => { setError(''); setAksi('batal') }}>
              Batalkan penjualan
            </button>
          </div>
        )}

        {!terkunci && aksi && (
          <form className="panel-aksi" onSubmit={periksa} style={{ marginTop: 14 }}>
            <h4 style={{ margin: '0 0 8px' }}>{aksi === 'batal' ? 'Batalkan penjualan' : 'Edit penjualan'}</h4>
            {aksi === 'koreksi' && (
              <>
                <div className="form-grid">
                  <div>
                    <label>Harga jual (Rp)</label>
                    <input type="number" min="1" value={harga} onChange={(e) => setHarga(e.target.value)} required />
                  </div>
                  <div>
                    <label>Tanggal jual</label>
                    <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
                  </div>
                </div>
                <PilihMetodeBayar
                  metode={metode}
                  onMetode={setMetode}
                  total={hargaBaru}
                  biaya={biaya}
                  onBiaya={setBiaya}
                  tujuanTunai={`kas ${item.lokasi}`}
                />
              </>
            )}
            <div style={{ marginTop: 10 }}>
              <label>Alasan</label>
              <input value={alasan} onChange={(e) => setAlasan(e.target.value)} placeholder="Contoh: salah input harga" maxLength={250} required />
            </div>
            <p className="muted">
              {aksi === 'batal'
                ? 'Barang kembali Tersedia, riwayat stok keluarnya dibatalkan, dan kas penjualan dikurangi sebesar harga barang ini.'
                : 'Penjualan lama dibatalkan lalu dicatat ulang dengan isian baru. Kas dikoreksi otomatis.'}
            </p>
            <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
              <button disabled={loading}>{loading ? <><Spinner />Memproses...</> : 'Lanjut'}</button>
              <button type="button" className="ghost" onClick={() => { setAksi(null); setError('') }}>Batal</button>
            </div>
          </form>
        )}
      </m.div>

      {konfirmasi && (
        <Popup
          judul={aksi === 'batal' ? `Batalkan penjualan ${item.item_no}?` : `Simpan koreksi penjualan ${item.item_no}?`}
          labelKonfirmasi={aksi === 'batal' ? 'Ya, batalkan penjualan' : 'Ya, simpan koreksi'}
          danger={aksi === 'batal'}
          loading={loading}
          onKonfirmasi={simpan}
          onTutup={() => setKonfirmasi(false)}
        >
          <p className="muted">Periksa barang dan rinciannya sebelum disimpan:</p>
          <SpesifikasiBarang item={item} />
          {aksi === 'batal' ? (
            <table className="tabel-spek" style={{ marginTop: 12 }}>
              <tbody>
                <tr><th>Penjualan</th><td>{rupiah(item.harga_jual || 0)}, {tglLama}, {labelMetode(metodeLama)}</td></tr>
                <tr><th>Alasan</th><td>{alasan.trim()}</td></tr>
                <tr><th>Akibat</th><td>Barang kembali Tersedia dan kas dikurangi {rupiah(item.harga_jual || 0)}</td></tr>
              </tbody>
            </table>
          ) : (
            <table className="tabel-spek" style={{ marginTop: 12 }}>
              <thead><tr><th></th><th>Sebelum</th><th>Sesudah</th></tr></thead>
              <tbody>
                <tr className={hargaBaru !== item.harga_jual ? 'berubah' : undefined}>
                  <th>Harga jual</th><td>{rupiah(item.harga_jual || 0)}</td><td>{rupiah(hargaBaru)}</td>
                </tr>
                <tr className={tgl !== tglLama ? 'berubah' : undefined}>
                  <th>Tanggal</th><td>{tglLama}</td><td>{tgl}</td>
                </tr>
                <tr className={metode !== metodeLama ? 'berubah' : undefined}>
                  <th>Metode</th><td>{labelMetode(metodeLama)}</td><td>{labelMetode(metode)}</td>
                </tr>
                <tr><th>Potongan</th><td>-</td><td>{potongan > 0 ? rupiah(potongan) : '-'}</td></tr>
                <tr><th>Alasan</th><td colSpan="2">{alasan.trim()}</td></tr>
              </tbody>
            </table>
          )}
        </Popup>
      )}
    </div>,
    document.body,
  )
}
