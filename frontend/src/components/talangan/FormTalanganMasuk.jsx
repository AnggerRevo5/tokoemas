import { useState, useEffect } from 'react'
import { api } from '../../services/api'
import { rupiah, hariIni, gram } from '../../utils/format'
import { KARAT_UMUM, KARAT_LAIN } from '../../constants/barang'
import { JENIS_TALANGAN, BAHAN_TALANGAN } from '../../constants/talangan'
import Popup from '../common/Popup'

const kosong = () => ({ nama_barang: '', jenis: 'perhiasan', bahan: 'emas', karat: '18K', berat: '', harga_beli: '' })

// Form input barang kategori "Barang cabang" (talangan MASUK): barang milik BP lain yang dibeli dari customer.
// Satu baris = satu barang; tiap simpan menampilkan kartu konfirmasi. Barang tidak masuk stok toko.
// kategoriSelect: <select> asal barang dari form induk supaya pengguna bisa kembali ke kategori lain.
export default function FormTalanganMasuk({ locationId, kategoriSelect, onMulai, onSukses, onError }) {
  const [bpList, setBpList] = useState([])
  const [bpId, setBpId] = useState('')
  const [tgl, setTgl] = useState(hariIni())
  const [noSurat, setNoSurat] = useState('')
  const [catatan, setCatatan] = useState('')
  const [baris, setBaris] = useState([kosong()])
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    api('/talangan/bp')
      .then((l) => {
        setBpList(l)
        setBpId((lama) => lama || (l[0] ? String(l[0].id) : ''))
      })
      .catch((e) => onError(e.message))
    // hanya saat form dibuka; onError dari induk tidak perlu memicu muat ulang
    // eslint-disable-next-line
  }, [])

  const ubah = (i, k, v) => setBaris((b) => b.map((x, idx) => (idx === i ? { ...x, [k]: v } : x)))
  const totalBeli = baris.reduce((s, b) => s + Number(b.harga_beli || 0), 0)
  const namaBP = (bpList.find((b) => String(b.id) === bpId) || {}).nama || ''
  const kadar = (b) => (b.jenis === 'logam_mulia' ? (b.bahan === 'perak' ? '999' : '24K') : b.bahan === 'perak' ? '925' : b.karat)

  function periksa(e) {
    e.preventDefault()
    onMulai()
    if (!bpId) return onError('Pilih BP pemilik barang')
    if (baris.some((b) => !(Number(b.harga_beli) > 0) || !(Number(b.berat) > 0))) return onError('Berat dan harga beli wajib diisi di setiap baris')
    setKonfirmasi(true)
  }

  async function simpan() {
    setLoading(true)
    let tersimpan = 0
    try {
      // backend menyimpan satu barang per permintaan; berhenti di kesalahan pertama
      for (const b of baris) {
        await api('/talangan', {
          method: 'POST',
          body: JSON.stringify({
            location_id: Number(locationId),
            bp_id: Number(bpId),
            nama_barang: b.nama_barang,
            jenis: b.jenis,
            bahan: b.bahan,
            karat: b.karat,
            berat: Number(b.berat),
            harga_beli: Math.round(Number(b.harga_beli)),
            tanggal: tgl,
            no_surat: noSurat,
            catatan,
          }),
        })
        tersimpan++
      }
      setKonfirmasi(false)
      setBaris([kosong()])
      setNoSurat('')
      setCatatan('')
      onSukses(`${tersimpan} barang talangan ${namaBP} tercatat, kas toko berkurang ${rupiah(totalBeli)}. Barang tidak masuk stok toko.`)
    } catch (err) {
      setKonfirmasi(false)
      if (tersimpan > 0) setBaris((b) => b.slice(tersimpan))
      onError(`${err.message}${tersimpan > 0 ? ` (${tersimpan} barang sebelumnya sudah tersimpan)` : ''}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="card" style={{ marginBottom: 16 }} onSubmit={periksa}>
      <h3>Barang cabang BP (talangan)</h3>
      <p className="muted">
        Barang ini milik cabang BP lain yang dijual customer ke toko. Toko membayar customer lebih dulu; barang tidak masuk stok toko dan
        diurus lewat menu Talangan sampai ditukar dengan BP.
      </p>
      <div className="form-grid">
        <div>
          <label>Asal barang</label>
          {kategoriSelect}
        </div>
        <div>
          <label>BP pemilik barang (sesuai surat customer)</label>
          <select value={bpId} onChange={(e) => setBpId(e.target.value)} required>
            {bpList.map((b) => <option key={b.id} value={b.id}>{b.nama}</option>)}
          </select>
        </div>
        <div>
          <label>Tanggal</label>
          <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
        </div>
        <div>
          <label>Nomor surat (opsional)</label>
          <input value={noSurat} onChange={(e) => setNoSurat(e.target.value)} maxLength={60} />
        </div>
        <div className="span2">
          <label>Catatan untuk semua barang</label>
          <input value={catatan} onChange={(e) => setCatatan(e.target.value)} maxLength={200} />
        </div>
      </div>

      <div style={{ marginTop: 12 }}>
        {baris.map((x, i) => (
          <div className="form-grid" key={i} style={{ marginBottom: 8 }}>
            <div>
              <label>Nama barang</label>
              <input value={x.nama_barang} onChange={(e) => ubah(i, 'nama_barang', e.target.value)} placeholder="Cincin" required maxLength={100} />
            </div>
            <div>
              <label>Jenis</label>
              <select value={x.jenis} onChange={(e) => ubah(i, 'jenis', e.target.value)}>
                {JENIS_TALANGAN.map(([k, l]) => <option key={k} value={k}>{l}</option>)}
              </select>
            </div>
            <div>
              <label>Bahan</label>
              <select value={x.bahan} onChange={(e) => ubah(i, 'bahan', e.target.value)}>
                {BAHAN_TALANGAN.map(([k, l]) => <option key={k} value={k}>{l}</option>)}
              </select>
            </div>
            {x.jenis === 'perhiasan' && x.bahan === 'emas' && (
              <div>
                <label>Kadar</label>
                <select value={x.karat} onChange={(e) => ubah(i, 'karat', e.target.value)}>
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
              <input type="number" step="0.001" min="0" value={x.berat} onChange={(e) => ubah(i, 'berat', e.target.value)} required />
            </div>
            <div>
              <label>Harga beli (Rp)</label>
              <input type="number" min="0" step="1" value={x.harga_beli} onChange={(e) => ubah(i, 'harga_beli', e.target.value)} required />
            </div>
            {baris.length > 1 && (
              <div style={{ alignSelf: 'end' }}>
                <button type="button" className="mini" onClick={() => setBaris((b) => b.filter((_, idx) => idx !== i))}>Hapus baris</button>
              </div>
            )}
          </div>
        ))}
        <button type="button" className="ghost" onClick={() => setBaris((b) => [...b, kosong()])}>+ Tambah barang</button>
      </div>

      <div className="form-actions" style={{ marginTop: 12 }}>
        <button type="submit" className="primary">Simpan talangan ({rupiah(totalBeli)})</button>
      </div>

      {konfirmasi && (
        <Popup judul="Simpan talangan?" labelKonfirmasi="Ya, simpan" onKonfirmasi={simpan} onTutup={() => setKonfirmasi(false)} loading={loading}>
          <p>
            Barang milik <b>{namaBP}</b>, dibeli dari customer. Kas toko berkurang <b>{rupiah(totalBeli)}</b>. Barang tidak masuk stok toko
            dan harga beli tidak bisa diubah setelah disimpan (koreksi hanya oleh owner).
          </p>
          <table className="tabel-spek">
            <tbody>
              {baris.map((b, i) => (
                <tr key={i}>
                  <td>{b.nama_barang}</td>
                  <td>{kadar(b)}</td>
                  <td className="num">{gram(b.berat)} g</td>
                  <td className="num">{rupiah(Number(b.harga_beli))}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Popup>
      )}
    </form>
  )
}
