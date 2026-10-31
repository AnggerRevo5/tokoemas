import { Fragment } from 'react'
import { Check } from 'lucide-react'

// Progres status sebagai urutan langkah. langkah = label sesuai urutan status yang sudah ada di constants;
// aktif = indeks status sekarang (-1 bila tidak ada di urutan, misalnya dibatalkan).
// ringkas = titik-titik kecil untuk di dalam tabel; tanpa ringkas = langkah berlabel (vertikal di HP).
export default function Stepper({ langkah, aktif, ringkas = false, batal = false }) {
  const akhir = aktif === langkah.length - 1
  const keadaan = (i) => (i < aktif || (akhir && i === aktif) ? 'selesai' : i === aktif ? 'aktif' : '')
  const judul = batal
    ? 'Dibatalkan'
    : aktif >= 0
      ? `Tahap ${aktif + 1} dari ${langkah.length}: ${langkah[aktif]}`
      : ''

  if (ringkas) {
    return (
      <div className={batal ? 'stepper-ringkas batal' : 'stepper-ringkas'} role="img" aria-label={judul} title={judul}>
        {langkah.map((l, i) => (
          <Fragment key={i}>
            {i > 0 && <i className={i <= aktif ? 'selesai' : ''} />}
            <span className={keadaan(i)} />
          </Fragment>
        ))}
      </div>
    )
  }

  return (
    <ol className={batal ? 'stepper batal' : 'stepper'} aria-label={judul}>
      {langkah.map((l, i) => (
        <li key={i} className={keadaan(i)} aria-current={i === aktif ? 'step' : undefined}>
          <span className="step-titik">
            {keadaan(i) === 'selesai' ? <Check size={13} strokeWidth={3} aria-hidden="true" /> : i + 1}
          </span>
          <span className="step-label">{l}</span>
        </li>
      ))}
    </ol>
  )
}
