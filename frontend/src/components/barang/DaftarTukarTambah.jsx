import { useEffect, useState, useCallback } from 'react'
import { api } from '../../services/api'
import { rupiah } from '../../utils/format'
import Popup from '../common/Popup'
import Pesan from '../common/Pesan'

// Tukar tambah 7 hari terakhir dengan tombol Batal (alasan wajib, tampil konfirmasi dulu).
export default function DaftarTukarTambah({ locId, versi, onUbah }) {
  const [rows, setRows] = useState([])
  const [error, setError] = useState('')
  const [batal, setBatal] = useState(null)
  const [alasan, setAlasan] = useState('')
  const [loading, setLoading] = useState(false)

  const muat = useCallback(() => {
    const p = locId ? `?location_id=${locId}` : ''
    api(`/barang/tukar-tambah${p}`).then(setRows).catch((e) => setError(e.message))
  }, [locId])
  useEffect(muat, [muat, versi])

  async function konfirmasi() {
    if (!alasan.trim()) return setError('Alasan wajib diisi')
    setLoading(true)
    try {
      const res = await api(`/barang/tukar-tambah/${batal.id}/batal`, { method: 'POST', body: JSON.stringify({ reason: alasan.trim() }) })
      setBatal(null)
      setAlasan('')
      setError('')
      onUbah(res.message)
      muat()
    } catch (e) {
      setBatal(null)
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Pesan error={error} />
      {rows.length === 0 && <p className="muted">Belum ada tukar tambah dalam 7 hari terakhir.</p>}
      {rows.map((r) => (
        <div key={r.id} className="panel-aksi" style={{ marginBottom: 10 }}>
          <strong>{r.tanggal} · {r.location_name}</strong>
          <div className="muted">Barang lama: {r.barang_lama || '-'}</div>
          <div className="muted">Barang baru: {r.barang_baru || '-'}</div>
          <div>
            Selisih {r.selisih > 0 ? 'masuk kas' : r.selisih < 0 ? 'keluar kas' : ''} {r.selisih === 0 ? 'nol' : rupiah(Math.abs(r.selisih))}
            {' '}({r.metode === 'qris' ? 'QRIS/Debit' : 'Tunai'}{r.biaya > 0 ? `, potongan ${rupiah(r.biaya)}` : ''})
          </div>
          {r.bisa_batal && (
            <button type="button" className="mini danger-fill" style={{ marginTop: 8 }} onClick={() => { setError(''); setAlasan(''); setBatal(r) }}>
              Batalkan
            </button>
          )}
        </div>
      ))}
      {batal && (
        <Popup
          judul="Batalkan tukar tambah ini?"
          labelKonfirmasi="Ya, batalkan"
          danger
          loading={loading}
          onKonfirmasi={konfirmasi}
          onTutup={() => setBatal(null)}
        >
          <p className="muted">Barang lama keluar dari stok, barang baru kembali tersedia, kas dikembalikan. Tidak bisa jika barang lama sudah dijual, dikirim, atau dicuci.</p>
          <label style={{ display: 'block', marginTop: 12 }}>Alasan membatalkan</label>
          <input value={alasan} onChange={(e) => setAlasan(e.target.value)} placeholder="Contoh: pelanggan tidak jadi" />
        </Popup>
      )}
    </div>
  )
}
