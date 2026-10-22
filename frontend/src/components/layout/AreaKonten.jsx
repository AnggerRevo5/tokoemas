import { useRef } from 'react'
import useLabelTabel from '../../hooks/useLabelTabel'

// Area isi halaman. Juga memberi label kolom pada tabel agar tampil sebagai kartu di layar kecil.
export default function AreaKonten({ children }) {
  const ref = useRef(null)
  useLabelTabel(ref)
  return <main className="content" ref={ref}>{children}</main>
}
