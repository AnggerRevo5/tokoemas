import { m } from 'motion/react'

// Kotak konfirmasi (void-box) dengan tombol aksi dan "Tidak jadi".
// children = isi kotak (teks penjelasan dan kolom input).
export default function KotakBatal({
  children,
  onKonfirmasi,
  onTutup,
  labelKonfirmasi = 'Ya, batalkan',
  danger = true,
}) {
  return (
    <m.div
      className="void-box"
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: 'easeOut' }}
    >
      {children}
      <div className="void-actions">
        <button
          type="button"
          className={danger ? 'mini danger-fill' : 'mini primary-fill'}
          onClick={onKonfirmasi}
        >
          {labelKonfirmasi}
        </button>
        <button type="button" className="mini" onClick={onTutup}>Tidak jadi</button>
      </div>
    </m.div>
  )
}
