import { useCallback, useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { m } from 'motion/react'
import { X } from 'lucide-react'
import { api } from '../../services/api'
import { gram, rupiah } from '../../utils/format'
import Pesan from '../common/Pesan'
import CardPenjualan from '../barang/CardPenjualan'

// Daftar barang yang membentuk satu catatan kas penjualan. Dari sini tiap penjualan bisa diedit atau dibatalkan
// lewat card penjualan yang sama dengan di menu Stok (kas, stok, dan laporan ikut dikoreksi).
// onUbah(pesan): ada perubahan (riwayat kas dimuat ulang); onTutup().
export default function PenjualanDariKas({ kas, me, onUbah, onTutup }) {
  const [rows, setRows] = useState(null)
  const [error, setError] = useState('')
  const [pilih, setPilih] = useState(null)

  const muat = useCallback(() => {
    api(`/barang?kas_id=${kas.id}&status=terjual`)
      .then(setRows)
      .catch((e) => setError(e.message))
  }, [kas.id])

  useEffect(() => {
    muat()
  }, [muat])

  useEffect(() => {
    const esc = (e) => e.key === 'Escape' && !pilih && onTutup()
    window.addEventListener('keydown', esc)
    return () => window.removeEventListener('keydown', esc)
  }, [onTutup, pilih])

  const daftar = rows ? rows.rows : []

  return createPortal(
    <div className="popup-latar latar-lebar" onMouseDown={(e) => e.target === e.currentTarget && !pilih && onTutup()}>
      <m.div
        className="popup popup-lebar"
        role="dialog"
        aria-modal="true"
        aria-label="Penjualan pada catatan kas ini"
        initial={{ opacity: 0, scale: 0.97, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.18, ease: 'easeOut' }}
      >
        <div className="popup-kepala">
          <h3>Penjualan pada kas {kas.transaction_date.slice(0, 10)} ({rupiah(kas.amount)})</h3>
          <button type="button" className="mini popup-tutup" onClick={onTutup} aria-label="Tutup">
            <X size={16} aria-hidden="true" />
          </button>
        </div>
        <Pesan error={error} />
        {!rows && !error && <p className="muted">Memuat...</p>}
        {rows && daftar.length === 0 && (
          <p className="muted">
            Tidak ada penjualan dari menu Stok yang masih terhubung ke catatan kas ini. Mungkin semuanya sudah dibatalkan.
          </p>
        )}
        {daftar.length > 0 && (
          <div className="table-wrap" style={{ marginTop: 12 }}>
            <table>
              <thead>
                <tr>
                  <th>No</th><th>Barang</th><th className="num">Berat</th><th className="num">Harga jual</th>
                  <th>Tanggal</th><th>Metode</th><th></th>
                </tr>
              </thead>
              <tbody>
                {daftar.map((r) => (
                  <tr key={r.id}>
                    <td>{r.item_no}</td>
                    <td>{r.item_name} {r.karat}</td>
                    <td className="num">{gram(r.weight_gram)} g</td>
                    <td className="num">{rupiah(r.harga_jual || 0)}</td>
                    <td>{r.tanggal_jual ? r.tanggal_jual.slice(0, 10) : '-'}</td>
                    <td>{r.metode === 'qris' ? 'QRIS/Debit' : r.metode === 'transfer' ? 'Transfer' : 'Tunai'}</td>
                    <td>
                      <button type="button" className="mini primary" onClick={() => setPilih(r)}>Ubah / batalkan</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </m.div>

      {pilih && (
        <CardPenjualan
          key={pilih.id}
          item={pilih}
          me={me}
          onUbah={(pesan) => {
            onUbah(pesan)
            muat()
          }}
          onTutup={() => setPilih(null)}
        />
      )}
    </div>,
    document.body,
  )
}
