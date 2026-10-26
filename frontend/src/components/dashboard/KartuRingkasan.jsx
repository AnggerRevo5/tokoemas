import { m } from 'motion/react'
import AngkaBerjalan from '../common/AngkaBerjalan'

// Kartu ringkasan di atas dashboard: angka besar serif emas yang bergulir saat tampil,
// dengan kilau tipis sekali lewat (lalu tiap ~8 detik). urutan = posisi kartu, untuk jeda kilau.
export default function KartuRingkasan({ ikon: Ikon, label, nilai, format, satuan, sub, urutan = 0 }) {
  return (
    <m.div
      className="card stat interaktif gold-shimmer"
      style={{ '--tunda': `${0.35 + urutan * 0.18}s` }}
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3, ease: 'easeOut', delay: urutan * 0.05 }}
    >
      <div className="stat-kepala">
        <span className="stat-label">{label}</span>
        {Ikon && (
          <span className="stat-ikon">
            <Ikon size={16} strokeWidth={1.8} aria-hidden="true" />
          </span>
        )}
      </div>
      <span className="angka-besar text-gold">
        <AngkaBerjalan nilai={nilai} format={format} dariNol />
        {satuan && <small>{satuan}</small>}
      </span>
      {sub && <div className="stat-sub">{sub}</div>}
    </m.div>
  )
}
