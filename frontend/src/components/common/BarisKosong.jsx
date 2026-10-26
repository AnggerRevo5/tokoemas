import { Inbox } from 'lucide-react'

// Baris tabel saat daftar kosong: ikon dan teks di tengah. kolom = jumlah kolom tabel.
export default function BarisKosong({ kolom, children }) {
  return (
    <tr className="baris-kosong">
      <td colSpan={kolom}>
        <div className="kosong">
          <Inbox size={26} strokeWidth={1.5} aria-hidden="true" />
          <span>{children}</span>
        </div>
      </td>
    </tr>
  )
}
