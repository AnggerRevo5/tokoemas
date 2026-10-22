import { useEffect } from 'react'

// Mengisi atribut data-label pada sel tabel dari teks header kolomnya.
// Dipakai CSS untuk menampilkan tabel sebagai kartu di layar HP. Tidak mengubah isi atau logika halaman.
function beriLabel(root) {
  root.querySelectorAll('table:not(.tetap)').forEach((tabel) => {
    const judul = [...tabel.querySelectorAll('thead th')].map((th) => th.textContent.trim())
    tabel.querySelectorAll('tbody tr').forEach((tr) => {
      ;[...tr.children].forEach((sel, i) => {
        if (sel.tagName !== 'TD' || sel.hasAttribute('colspan')) return
        const label = judul[i] || ''
        if (sel.getAttribute('data-label') !== label) sel.setAttribute('data-label', label)
      })
    })
  })
}

export default function useLabelTabel(ref) {
  useEffect(() => {
    const root = ref.current
    if (!root) return undefined
    let jadwal = 0
    const jalan = () => {
      cancelAnimationFrame(jadwal)
      jadwal = requestAnimationFrame(() => beriLabel(root))
    }
    jalan()
    const pengamat = new MutationObserver(jalan)
    pengamat.observe(root, { childList: true, subtree: true })
    return () => {
      pengamat.disconnect()
      cancelAnimationFrame(jadwal)
    }
  }, [ref])
}
