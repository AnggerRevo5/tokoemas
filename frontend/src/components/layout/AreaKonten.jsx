import { useRef } from 'react'
import { m } from 'motion/react'
import useLabelTabel from '../../hooks/useLabelTabel'

// Area isi halaman. Juga memberi label kolom pada tabel agar tampil sebagai kartu di layar kecil.
// halaman = kode halaman aktif; setiap ganti halaman isinya muncul dengan fade + geser singkat.
export default function AreaKonten({ halaman, children }) {
  const ref = useRef(null)
  useLabelTabel(ref)
  return (
    <main className="content" ref={ref}>
      <m.div
        key={halaman}
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.22, ease: 'easeOut' }}
      >
        {children}
      </m.div>
    </main>
  )
}
