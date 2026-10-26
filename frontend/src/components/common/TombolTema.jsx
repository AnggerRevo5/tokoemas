import { AnimatePresence, m } from 'motion/react'
import { Moon, Sun } from 'lucide-react'

// Tombol ganti tema: matahari (emas) di mode terang, bulan di mode gelap, berganti dengan transisi halus.
export default function TombolTema({ tema, onUbah, className = '' }) {
  const terang = tema !== 'gelap'
  const label = terang ? 'Ganti ke mode gelap' : 'Ganti ke mode terang'

  return (
    <button
      type="button"
      className={`ghost tema-ikon ${className}`.trim()}
      onClick={onUbah}
      aria-label={label}
      title={label}
    >
      <AnimatePresence initial={false}>
        <m.span
          key={terang ? 'terang' : 'gelap'}
          className={terang ? 'ikon-tema matahari' : 'ikon-tema'}
          initial={{ opacity: 0, rotate: -45, scale: 0.7 }}
          animate={{ opacity: 1, rotate: 0, scale: 1 }}
          exit={{ opacity: 0, rotate: 45, scale: 0.7 }}
          transition={{ duration: 0.25, ease: 'easeOut' }}
        >
          {terang ? <Sun size={19} strokeWidth={1.9} aria-hidden="true" /> : <Moon size={18} strokeWidth={1.9} aria-hidden="true" />}
        </m.span>
      </AnimatePresence>
    </button>
  )
}
