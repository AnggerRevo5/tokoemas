import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { m } from 'motion/react'
import { X } from 'lucide-react'
import { api } from '../../services/api'
import { NAMA_UMUM, KARAT_UMUM, KARAT_LAIN } from '../../constants/barang'
import Spinner from '../common/Spinner'
import Popup from '../common/Popup'
import Pesan from '../common/Pesan'
import SpesifikasiBarang from './SpesifikasiBarang'

// Card edit satu barang di tengah layar (nama/merek, kadar, berat, harga beli, keterangan, baki).
// baki: daftar baki di lokasi barang. onSelesai(pesan): setelah tersimpan; onTutup().
// onHapus(item): buka konfirmasi hapus barang (hanya dikirim bila akun boleh menghapus).
// Ditutup dengan tombol silang, klik latar, atau Esc.
export default function FormEditBarang({ item, baki, kapasitas, onSelesai, onTutup, onHapus }) {
  const lm = item.jenis === 'logam_mulia'
  const [f, setF] = useState({
    item_name: item.item_name,
    brand: item.brand,
    karat: item.karat,
    weight_gram: String(item.weight_gram),
    harga_beli: String(item.harga_beli || ''),
    note: item.note,
    tray_id: String(item.baki_id || ''),
  })
  const [loading, setLoading] = useState(false)
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [error, setError] = useState('')
  const ubah = (k) => (e) => setF((x) => ({ ...x, [k]: e.target.value }))
  // harga beli dari customer sudah masuk kas, jadi tidak bisa diubah di sini
  const hargaTerkunci = item.dari_customer

  // gambaran barang setelah diedit, untuk dibandingkan di popup konfirmasi
  const bakiBaru = baki.find((b) => String(b.id) === f.tray_id)
  const sesudah = {
    ...item,
    item_name: lm ? `Logam mulia ${f.brand.trim()}` : f.item_name.trim(),
    brand: lm ? f.brand.trim() : item.brand,
    karat: lm ? '24K' : f.karat,
    weight_gram: Number(f.weight_gram),
    harga_beli: Number(f.harga_beli || 0),
    note: f.note.trim(),
    baki_no: lm ? 0 : bakiBaru ? bakiBaru.no : 0,
  }

  const adaPerubahan = ['item_name', 'karat', 'weight_gram', 'harga_beli', 'note', 'baki_no']
    .some((k) => String(sesudah[k] ?? '') !== String(item[k] ?? ''))

  useEffect(() => {
    const esc = (e) => e.key === 'Escape' && !konfirmasi && !loading && onTutup()
    window.addEventListener('keydown', esc)
    return () => window.removeEventListener('keydown', esc)
  }, [onTutup, konfirmasi, loading])

  function periksa(e) {
    e.preventDefault()
    setError('')
    setKonfirmasi(true)
  }

  async function simpan() {
    setLoading(true)
    try {
      const res = await api(`/barang/${item.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          item_name: f.item_name,
          brand: f.brand,
          karat: f.karat,
          weight_gram: Number(f.weight_gram),
          harga_beli: Number(f.harga_beli || 0),
          note: f.note,
          tray_id: Number(f.tray_id || 0),
        }),
      })
      onSelesai(res.message)
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
      <m.form
        className="popup popup-lebar"
        role="dialog"
        aria-modal="true"
        aria-label={`Edit barang ${item.item_no}`}
        onSubmit={periksa}
        initial={{ opacity: 0, scale: 0.97, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.18, ease: 'easeOut' }}
      >
      <div className="popup-kepala">
        <h3>Edit barang {item.item_no} - {item.item_name} {item.karat}</h3>
        <button type="button" className="mini popup-tutup" onClick={onTutup} disabled={loading} aria-label="Tutup">
          <X size={16} aria-hidden="true" />
        </button>
      </div>
      <Pesan error={error} />
      <div className="form-grid" style={{ marginTop: 12 }}>
        {lm ? (
          <div>
            <label>Merek</label>
            <input value={f.brand} onChange={ubah('brand')} required />
          </div>
        ) : (
          <div>
            <label>Nama perhiasan</label>
            <input list="nama-edit" value={f.item_name} onChange={ubah('item_name')} required />
            <datalist id="nama-edit">
              {NAMA_UMUM.map((n) => <option key={n} value={n} />)}
            </datalist>
          </div>
        )}
        <div>
          <label>Kadar</label>
          {lm ? (
            <input value="24K" disabled />
          ) : (
            <select value={f.karat} onChange={ubah('karat')} required>
              <option value="">Kadar</option>
              <optgroup label="Umum">
                {KARAT_UMUM.map((n) => <option key={n} value={`${n}K`}>{n}K</option>)}
              </optgroup>
              <optgroup label="Lainnya">
                {KARAT_LAIN.map((n) => <option key={n} value={`${n}K`}>{n}K</option>)}
              </optgroup>
            </select>
          )}
        </div>
        <div>
          <label>Berat (gram)</label>
          <input type="number" step="0.001" min="0.001" value={f.weight_gram} onChange={ubah('weight_gram')} required />
        </div>
        <div>
          <label>Harga beli (Rp)</label>
          <input type="number" min="0" value={f.harga_beli} onChange={ubah('harga_beli')} disabled={hargaTerkunci} />
        </div>
        <div>
          <label>Keterangan</label>
          <input value={f.note} onChange={ubah('note')} />
        </div>
        {!lm && (
          <div>
            <label>Baki</label>
            <select value={f.tray_id} onChange={ubah('tray_id')}>
              <option value="">Tanpa baki</option>
              {baki.map((b) => {
                const penuh = b.isi >= kapasitas && b.id !== item.baki_id
                return (
                  <option key={b.id} value={b.id} disabled={penuh}>
                    Baki {b.no} ({b.isi}/{kapasitas}{penuh ? ', penuh' : ''})
                  </option>
                )
              })}
            </select>
          </div>
        )}
      </div>
      {hargaTerkunci && (
        <p className="muted">Harga beli barang dari customer sudah tercatat di kas, jadi tidak bisa diubah di sini.</p>
      )}
      <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
        <button disabled={loading}>{loading ? <><Spinner />Menyimpan...</> : 'Simpan perubahan'}</button>
        <button type="button" className="ghost" onClick={onTutup}>Batal</button>
        {onHapus && (
          <button
            type="button"
            className="mini danger-fill"
            style={{ marginLeft: 'auto' }}
            disabled={loading}
            onClick={() => {
              onTutup()
              onHapus(item)
            }}
          >
            Hapus barang
          </button>
        )}
      </div>
      </m.form>
      {konfirmasi && (
        <Popup
          judul={`Simpan perubahan ${item.item_no}?`}
          labelKonfirmasi="Ya, simpan perubahan"
          loading={loading}
          nonaktif={!adaPerubahan}
          onKonfirmasi={simpan}
          onTutup={() => setKonfirmasi(false)}
        >
          <p className="muted">Periksa lagi spesifikasi barang. Baris yang berubah ditandai.</p>
          <SpesifikasiBarang item={item} sesudah={sesudah} />
          {!adaPerubahan && <p className="muted" style={{ marginTop: 10 }}>Tidak ada yang diubah.</p>}
        </Popup>
      )}
    </div>,
    document.body,
  )
}
