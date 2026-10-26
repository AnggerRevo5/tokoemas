import { useEffect } from 'react'
import { animate, m, useMotionValue, useReducedMotion, useTransform } from 'motion/react'

// Angka yang bergulir singkat (±600 ms) ke nilai baru. Hanya tampilan: angka akhirnya selalu format(nilai).
// format sebaiknya fungsi tetap (di luar komponen). dariNol: bergulir dari 0 saat pertama tampil.
export default function AngkaBerjalan({ nilai, format = String, durasi = 0.6, dariNol = false, className }) {
  const kurangiGerak = useReducedMotion()
  const angka = useMotionValue(dariNol && !kurangiGerak ? 0 : nilai)
  const teks = useTransform(angka, (v) => format(v))

  useEffect(() => {
    if (kurangiGerak || !Number.isFinite(Number(nilai))) {
      angka.set(nilai)
      return undefined
    }
    const kontrol = animate(angka, Number(nilai), { duration: durasi, ease: 'easeOut' })
    return () => kontrol.stop()
  }, [nilai, durasi, kurangiGerak, angka])

  return <m.span className={className}>{teks}</m.span>
}
