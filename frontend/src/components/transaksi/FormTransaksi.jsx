import { useState, useEffect } from 'react'
import { api } from '../../services/api'
import { hariIni } from '../../utils/format'
import { KATEGORI } from '../../constants/kategori'
import PilihanLokasi from '../common/PilihanLokasi'

// Form input kas atau stok emas. Lokasi (locId) dipegang halaman karena dipakai juga oleh riwayat.
// onMulai: bersihkan pesan; onSukses(res): setelah tersimpan; onError(msg).
export default function FormTransaksi({
  jenis, me, locations, locId, onLocId, onMulai, onSukses, onError,
}) {
  const isKas = jenis === 'kas'

  const [direction, setDirection] = useState('in')
  const [category, setCategory] = useState('')
  const [target, setTarget] = useState('')
  const [tanggal, setTanggal] = useState(hariIni())
  const [amount, setAmount] = useState('')
  const [itemName, setItemName] = useState('')
  const [berat, setBerat] = useState('')
  const [karat, setKarat] = useState('')
  const [qty, setQty] = useState(1)
  const [note, setNote] = useState('')
  const [loading, setLoading] = useState(false)

  const lokasi = locations.find((l) => l.id === Number(locId))
  const tipe = lokasi ? lokasi.type : 'cabang'
  const opsi = KATEGORI[jenis][tipe][direction]
  const cabangList = locations.filter((l) => l.type === 'cabang')

  useEffect(() => {
    const c = locations.find((l) => l.type === 'cabang')
    if (c) setTarget(String(c.id))
  }, [locations])

  useEffect(() => {
    setCategory(opsi[0][0])
  }, [tipe, direction, jenis])

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    const body = isKas
      ? {
          location_id: Number(locId),
          direction,
          category,
          amount: Number(amount),
          note,
          transaction_date: tanggal,
        }
      : {
          location_id: Number(locId),
          target_location_id: category === 'ke_toko' ? Number(target) : 0,
          direction,
          category,
          item_name: itemName,
          weight_gram: Number(berat),
          karat,
          qty: Number(qty),
          note,
          transaction_date: tanggal,
        }
    try {
      const res = await api(`/${jenis}`, { method: 'POST', body: JSON.stringify(body) })
      setAmount('')
      setItemName('')
      setBerat('')
      setKarat('')
      setQty(1)
      setNote('')
      onSukses(res)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="card" onSubmit={simpan}>
      <h3>{isKas ? 'Input Kas' : 'Input Stok Emas'}</h3>

      <div className="form-grid">
        {me.role !== 'cabang' && (
          <div>
            <label>Lokasi</label>
            <PilihanLokasi locations={locations} value={locId} onChange={(e) => onLocId(e.target.value)} />
          </div>
        )}

        <div>
          <label>Arah</label>
          <select value={direction} onChange={(e) => setDirection(e.target.value)}>
            <option value="in">{isKas ? 'Uang masuk' : 'Barang masuk'}</option>
            <option value="out">{isKas ? 'Uang keluar' : 'Barang keluar'}</option>
          </select>
        </div>

        <div>
          <label>Kategori</label>
          <select value={category} onChange={(e) => setCategory(e.target.value)}>
            {opsi.map(([k, l]) => (
              <option key={k} value={k}>{l}</option>
            ))}
          </select>
        </div>

        {!isKas && category === 'ke_toko' && (
          <div>
            <label>Kirim ke toko</label>
            <PilihanLokasi locations={cabangList} value={target} onChange={(e) => setTarget(e.target.value)} />
          </div>
        )}

        <div>
          <label>Tanggal</label>
          <input type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} required />
        </div>

        {isKas ? (
          <div>
            <label>Nominal (Rp)</label>
            <input type="number" min="1" value={amount} onChange={(e) => setAmount(e.target.value)} required />
          </div>
        ) : (
          <>
            <div>
              <label>Nama barang</label>
              <input value={itemName} onChange={(e) => setItemName(e.target.value)} placeholder="Cincin, kalung, gelang..." required />
            </div>
            <div>
              <label>Berat total (gram)</label>
              <input type="number" step="0.001" min="0.001" value={berat} onChange={(e) => setBerat(e.target.value)} required />
            </div>
            <div>
              <label>Kadar</label>
              <input value={karat} onChange={(e) => setKarat(e.target.value)} placeholder="22K" />
            </div>
            <div>
              <label>Jumlah (pcs)</label>
              <input type="number" min="1" value={qty} onChange={(e) => setQty(e.target.value)} required />
            </div>
          </>
        )}

        <div className="span2">
          <label>Catatan</label>
          <input value={note} onChange={(e) => setNote(e.target.value)} />
        </div>
      </div>

      {isKas && direction === 'in' && category === 'omzet' && (
        <p className="muted">
          Penjualan barang lewat menu Stok sudah otomatis masuk ke kas. Pakai isian ini hanya untuk
          pemasukan penjualan di luar sistem, supaya tidak tercatat dua kali.
        </p>
      )}
      {!isKas && category === 'ke_toko' && (
        <p className="muted">Otomatis tercatat sebagai barang masuk di toko tujuan.</p>
      )}
      {!isKas && category === 'kembali_bo' && (
        <p className="muted">Otomatis tercatat sebagai barang masuk di Back Office.</p>
      )}

      <button disabled={loading}>{loading ? 'Menyimpan...' : 'Simpan'}</button>
    </form>
  )
}
