import { useState } from 'react'
import { api } from '../../services/api'
import { rupiah, hariIni, gram } from '../../utils/format'
import { KARAT_UMUM, KARAT_LAIN } from '../../constants/barang'
import { JENIS_TALANGAN, BAHAN_TALANGAN } from '../../constants/talangan'
import Popup from '../common/Popup'

const kosong = () => ({ nama_barang: '', jenis: 'perhiasan', bahan: 'emas', karat: '18K', berat: '', harga_beli: '', no_referensi: '' })

// Form PJ untuk barang kita yang dibeli BP dari customer (talangan KELUAR). Barang masuk stok BO sebagai stok kita biasa.
// onSukses(idBaru): dipanggil setelah tersimpan (layar Penukaran memakainya untuk langsung memilih barang itu).
export default function FormKeluar({ bpId, namaBP, onSukses, onError, onMulai }) {
  const [f, setF] = useState(kosong)
  const [tgl, setTgl] = useState(hariIni())
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)
  const ubah = (k) => (e) => setF((x) => ({ ...x, [k]: e.target.value }))

  function periksa(e) {
    e.preventDefault()
    onMulai?.()
    if (!bpId) return onError('Pilih cabang BP dulu')
    setKonfirmasi(true)
  }

  async function simpan() {
    setLoading(true)
    try {
      const res = await api('/talangan/keluar', {
        method: 'POST',
        body: JSON.stringify({
          bp_id: Number(bpId),
          nama_barang: f.nama_barang,
          jenis: f.jenis,
          bahan: f.bahan,
          karat: f.karat,
          berat: Number(f.berat),
          harga_beli: Math.round(Number(f.harga_beli)),
          tanggal: tgl,
          no_referensi: f.no_referensi,
        }),
      })
      setKonfirmasi(false)
      setF(kosong())
      onSukses(res.id, res.message)
    } catch (err) {
      setKonfirmasi(false)
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="panel-aksi" onSubmit={periksa} style={{ marginTop: 12 }}>
      <h4>Catat barang kita yang diambil dari {namaBP || 'BP'}</h4>
      <div className="form-grid">
        <div>
          <label>Nama barang</label>
          <input value={f.nama_barang} onChange={ubah('nama_barang')} required maxLength={100} />
        </div>
        <div>
          <label>Jenis</label>
          <select value={f.jenis} onChange={ubah('jenis')}>
            {JENIS_TALANGAN.map(([k, l]) => <option key={k} value={k}>{l}</option>)}
          </select>
        </div>
        <div>
          <label>Bahan</label>
          <select value={f.bahan} onChange={ubah('bahan')}>
            {BAHAN_TALANGAN.map(([k, l]) => <option key={k} value={k}>{l}</option>)}
          </select>
        </div>
        {f.jenis === 'perhiasan' && f.bahan === 'emas' && (
          <div>
            <label>Kadar</label>
            <select value={f.karat} onChange={ubah('karat')}>
              <optgroup label="Umum">
                {KARAT_UMUM.map((n) => <option key={n} value={`${n}K`}>{n}K</option>)}
              </optgroup>
              <optgroup label="Lainnya">
                {KARAT_LAIN.map((n) => <option key={n} value={`${n}K`}>{n}K</option>)}
              </optgroup>
            </select>
          </div>
        )}
        <div>
          <label>Berat (gram)</label>
          <input type="number" step="0.001" min="0" value={f.berat} onChange={ubah('berat')} required />
        </div>
        <div>
          <label>Harga yang BP bayar ke customer (Rp)</label>
          <input type="number" step="1" min="0" value={f.harga_beli} onChange={ubah('harga_beli')} required />
        </div>
        <div>
          <label>Tanggal</label>
          <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
        </div>
        <div>
          <label>Nomor nota BP (opsional)</label>
          <input value={f.no_referensi} onChange={ubah('no_referensi')} maxLength={60} />
        </div>
      </div>
      <div className="form-actions" style={{ marginTop: 10 }}>
        <button type="submit" className="primary">Simpan barang keluar</button>
      </div>

      {konfirmasi && (
        <Popup judul="Simpan barang keluar?" labelKonfirmasi="Ya, simpan" onKonfirmasi={simpan} onTutup={() => setKonfirmasi(false)} loading={loading}>
          <p>
            <b>{f.nama_barang}</b> {f.jenis === 'perhiasan' && f.bahan === 'emas' ? f.karat : ''} {gram(f.berat)} g, harga {rupiah(Number(f.harga_beli))}.
            Barang masuk stok back office sebagai stok kita dan dihitung dalam penukaran dengan {namaBP}. Harga tidak bisa diubah setelah disimpan.
          </p>
        </Popup>
      )}
    </form>
  )
}
