import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { AnimatePresence, m } from 'motion/react'
import { Check, CircleAlert, X } from 'lucide-react'

const LAMA_TOAST = 6000

// Toast di pojok layar untuk pesan sukses; menutup sendiri atau lewat tombol silang.
function Toast({ pesan }) {
  const [lalu, setLalu] = useState(pesan)
  const [tutup, setTutup] = useState(false)

  // pesan baru selalu tampil lagi meskipun yang sebelumnya sudah ditutup
  if (pesan !== lalu) {
    setLalu(pesan)
    setTutup(false)
  }

  useEffect(() => {
    if (!pesan || tutup) return undefined
    const t = setTimeout(() => setTutup(true), LAMA_TOAST)
    return () => clearTimeout(t)
  }, [pesan, tutup])

  return createPortal(
    <div className="toast-wadah" role="status" aria-live="polite">
      <AnimatePresence>
        {pesan && !tutup && (
          <m.div
            key={pesan}
            className="toast"
            initial={{ opacity: 0, y: 12 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 8 }}
            transition={{ duration: 0.22, ease: 'easeOut' }}
          >
            <span className="toast-ikon">
              <Check size={13} strokeWidth={3} aria-hidden="true" />
            </span>
            <span className="toast-teks">{pesan}</span>
            <button type="button" className="toast-tutup" onClick={() => setTutup(true)} aria-label="Tutup pesan">
              <X size={16} aria-hidden="true" />
            </button>
          </m.div>
        )}
      </AnimatePresence>
    </div>,
    document.body,
  )
}

// Pesan halaman: error (merah bata) dan peringatan (banner) di atas halaman, info sukses sebagai toast.
export default function Pesan({ error, info, warn }) {
  return (
    <>
      <AnimatePresence initial={false}>
        {error && (
          <m.div
            key="error"
            className="error"
            role="alert"
            style={{ marginBottom: 16 }}
            initial={{ opacity: 0, y: -8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.2, ease: 'easeOut' }}
          >
            <CircleAlert size={18} aria-hidden="true" />
            <span>{error}</span>
          </m.div>
        )}
      </AnimatePresence>
      {warn && <div className="banner" style={{ marginBottom: 16 }}>{warn}</div>}
      <Toast pesan={info} />
    </>
  )
}
