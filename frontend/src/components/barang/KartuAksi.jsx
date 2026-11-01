import { createPortal } from 'react-dom'
import { m } from 'motion/react'

// Card di tengah layar untuk mengisi aksi barang (jual, gadai, kirim, dll). Ditutup lewat tombol Tutup atau klik latar.
// galat: pesan kesalahan dari isian, ditampilkan di dalam card.
export default function KartuAksi({ judul, galat, onTutup, children }) {
  return createPortal(
    <div className="popup-latar latar-lebar" onMouseDown={(e) => e.target === e.currentTarget && onTutup()}>
      <m.div
        className="popup popup-lebar"
        role="dialog"
        aria-modal="true"
        aria-label={judul}
        initial={{ opacity: 0, scale: 0.97, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.18, ease: 'easeOut' }}
      >
        <div className="popup-kepala">
          <h3>{judul}</h3>
          <button type="button" className="mini popup-tutup" onClick={onTutup}>Tutup</button>
        </div>
        {galat && <div className="error" role="alert" style={{ marginTop: 10 }}>{galat}</div>}
        <div className="popup-isi">{children}</div>
      </m.div>
    </div>,
    document.body,
  )
}
