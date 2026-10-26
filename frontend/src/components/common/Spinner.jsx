import { LoaderCircle } from 'lucide-react'

// Ikon memproses yang berputar halus, dipakai di tombol saat loading.
export default function Spinner({ size = 16 }) {
  return <LoaderCircle className="spinner" size={size} strokeWidth={2.2} aria-hidden="true" />
}
