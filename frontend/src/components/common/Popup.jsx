import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import { m } from 'motion/react'
import Spinner from './Spinner'

// Popup konfirmasi di tengah layar. children = isi (spesifikasi barang, kolom alasan, dll).
// onKonfirmasi dipanggil saat tombol utama ditekan; onTutup saat "Tidak jadi", klik latar, atau Esc.
export default function Popup({
  judul,
  children,
  onKonfirmasi,
  onTutup,
  labelKonfirmasi = 'Ya, lanjutkan',
  danger = false,
  loading = false,
  nonaktif = false,
}) {
  useEffect(() => {
    const esc = (e) => {
      if (e.key === 'Escape' && !loading) onTutup()
    }
    window.addEventListener('keydown', esc)
    return () => window.removeEventListener('keydown', esc)
  }, [onTutup, loading])

  return createPortal(
    <div className="popup-latar" onMouseDown={(e) => e.target === e.currentTarget && !loading && onTutup()}>
      <m.div
        className="popup"
        role="dialog"
        aria-modal="true"
        aria-label={judul}
        initial={{ opacity: 0, scale: 0.97, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.18, ease: 'easeOut' }}
      >
        <h3>{judul}</h3>
        <div className="popup-isi">{children}</div>
        <div className="void-actions">
          <button
            type="button"
            className={danger ? 'mini danger-fill' : 'mini primary-fill'}
            onClick={onKonfirmasi}
            disabled={loading || nonaktif}
            autoFocus
          >
            {loading && <Spinner />}{labelKonfirmasi}
          </button>
          <button type="button" className="mini" onClick={onTutup} disabled={loading}>Tidak jadi</button>
        </div>
      </m.div>
    </div>,
    document.body,
  )
}
