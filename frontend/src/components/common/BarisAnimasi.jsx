import { m } from 'motion/react'

const varian = {
  keluar: (instan) =>
    instan
      ? { opacity: 0, transition: { duration: 0 } }
      : { opacity: 0, x: -16, transition: { duration: 0.22, ease: 'easeOut' } },
}

// Baris tabel yang keluar dengan geser + pudar; baris lain bergeser halus mengisi tempatnya.
// Dipakai di dalam <AnimatePresence custom={instan}> bersama hook useAnimasiDaftar.
export default function BarisAnimasi({ layout, children, ...props }) {
  return (
    <m.tr
      layout={layout ? 'position' : false}
      variants={varian}
      exit="keluar"
      transition={{ layout: { duration: 0.25, ease: 'easeOut' } }}
      {...props}
    >
      {children}
    </m.tr>
  )
}
