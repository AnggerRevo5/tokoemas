import { useEffect, useRef } from 'react'

// di atas jumlah ini baris tidak dianimasikan posisinya agar tabel panjang tetap ringan
const BATAS_LAYOUT = 120
// lebih dari ini baris hilang sekaligus dianggap ganti filter, bukan konfirmasi
const BATAS_KELUAR = 8

// Menentukan apakah perubahan daftar dianimasikan. Animasi keluar hanya untuk perubahan kecil
// (misalnya baris yang baru dikonfirmasi); saat seluruh isi daftar berganti, baris lama langsung hilang.
// Tidak mengubah data: hanya membandingkan id baris dengan render sebelumnya.
export default function useAnimasiDaftar(ids) {
  const lalu = useRef(ids)
  const ada = new Set(ids)
  const hilang = lalu.current.filter((id) => !ada.has(id)).length
  const instan = hilang > BATAS_KELUAR || (hilang > 0 && hilang === lalu.current.length && ids.length > 0)

  useEffect(() => {
    lalu.current = ids
  })

  return { instan, layout: ids.length <= BATAS_LAYOUT }
}
