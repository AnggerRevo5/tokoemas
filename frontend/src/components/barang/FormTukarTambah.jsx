import { useState, useEffect } from 'react'
import { api } from '../../services/api'
import { gram, hariIni, rupiah } from '../../utils/format'
import { KARAT } from '../../constants/barang'
import KartuAksi from './KartuAksi'
import Popup from '../common/Popup'
import DaftarTukarTambah from './DaftarTukarTambah'

const LAMA_KOSONG = { jenis: 'perhiasan', item_name: '', karat: '8', weight_gram: '', brand: '', nilai: '', kondisi: 'kotor' }

// Tukar tambah: banyak barang lama dari pelanggan (nilai tukar diisi manual) dan banyak barang baru dari stok toko.
// Kas toko hanya mencatat selisih; tunai atau QRIS/Debit sama-sama masuk kas toko.
export default function FormTukarTambah({ me, locations, onTutup, onSelesai }) {
  const cabang = locations.filter((l) => l.type === 'cabang')
  const [tab, setTab] = useState('baru')
  const [versi, setVersi] = useState(0)
  const [lokasi, setLokasi] = useState(me.role === 'cabang' ? String(me.location_id) : String(cabang[0]?.id || ''))
  const [lama, setLama] = useState([{ ...LAMA_KOSONG }])
  const [baru, setBaru] = useState([]) // { id, nama, harga }
  const [cari, setCari] = useState('')
  const [hasil, setHasil] = useState([])
  const [metode, setMetode] = useState('tunai')
  const [biaya, setBiaya] = useState('')
  const [tgl, setTgl] = useState(hariIni())
  const [note, setNote] = useState('')
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const totalLama = lama.reduce((s, x) => s + (Number(x.nilai) || 0), 0)
  const totalBaru = baru.reduce((s, x) => s + (Number(x.harga) || 0), 0)
  const selisih = totalBaru - totalLama
  const potongan = metode === 'qris' && selisih > 0 ? Number(biaya) || 0 : 0

  useEffect(() => {
    if (!lokasi || !cari.trim()) {
      setHasil([])
      return undefined
    }
    const t = setTimeout(() => {
      api(`/barang?location_id=${lokasi}&status=tersedia&limit=20&q=${encodeURIComponent(cari.trim())}`)
        .then((d) => setHasil(d.rows))
        .catch(() => setHasil([]))
    }, 250)
    return () => clearTimeout(t)
  }, [lokasi, cari])

  const ubahLama = (i, k, v) => setLama((l) => l.map((x, n) => (n === i ? { ...x, [k]: v } : x)))
  const ubahBaru = (i, v) => setBaru((l) => l.map((x, n) => (n === i ? { ...x, harga: v } : x)))
  const tambahBaru = (r) => {
    if (baru.some((x) => x.id === r.id)) return
    setBaru((l) => [...l, { id: r.id, nama: `${r.item_no} ${r.item_name} ${r.karat || ''} ${gram(r.weight_gram)}g`, harga: '', gratis: r.jenis === 'tempat_perhiasan' || r.jenis === 'kotak_perhiasan' }])
  }

  function periksa(e) {
    e.preventDefault()
    setError('')
    if (!lokasi) return setError('Pilih toko')
    if (baru.length === 0) return setError('Pilih minimal satu barang baru')
    for (const [i, x] of lama.entries()) {
      if (x.jenis === 'logam_mulia' ? !x.brand.trim() : !x.item_name.trim()) return setError(`Barang lama ${i + 1}: nama atau merek wajib diisi`)
      if (!(Number(x.weight_gram) > 0)) return setError(`Barang lama ${i + 1}: berat wajib diisi`)
      if (!(Number(x.nilai) > 0)) return setError(`Barang lama ${i + 1}: nilai tukar wajib diisi`)
    }
    for (const x of baru) {
      if (!(Number(x.harga) > 0) && !x.gratis) return setError('Harga semua barang baru wajib diisi')
    }
    if (metode === 'qris' && selisih < 0) return setError('Selisih yang dibayar toko ke pelanggan hanya bisa tunai')
    if (potongan > selisih) return setError('Potongan tidak boleh melebihi selisih')
    setKonfirmasi(true)
  }

  async function simpan() {
    setLoading(true)
    try {
      const res = await api('/barang/tukar-tambah', {
        method: 'POST',
        body: JSON.stringify({
          location_id: Number(lokasi), date: tgl, note: note.trim(), metode, biaya: potongan,
          lama: lama.map((x) => ({
            jenis: x.jenis, item_name: x.item_name, brand: x.brand, karat: `${x.karat}K`, weight_gram: Number(x.weight_gram),
            nilai: Number(x.nilai), kondisi: x.kondisi,
          })),
          baru: baru.map((x) => ({ item_id: x.id, harga: Number(x.harga) || 0 })),
        }),
      })
      onSelesai(res.message)
      setLama([{ ...LAMA_KOSONG }])
      setBaru([])
      setBiaya('')
      setNote('')
      setCari('')
      setKonfirmasi(false)
      setVersi((v) => v + 1)
      setTab('daftar')
    } catch (err) {
      setKonfirmasi(false)
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const teksSelisih = selisih > 0 ? `Pelanggan membayar ${rupiah(selisih)} (kas toko masuk)`
    : selisih < 0 ? `Toko membayar ${rupiah(-selisih)} ke pelanggan (kas toko keluar)` : 'Tidak ada selisih kas'

  return (
    <KartuAksi judul="Tukar tambah" galat={error} onTutup={onTutup}>
      <div className="aksi-tombol" style={{ marginBottom: 12 }}>
        <button type="button" className={`mini ${tab === 'baru' ? 'primary-fill' : ''}`} onClick={() => setTab('baru')}>Tukar tambah baru</button>
        <button type="button" className={`mini ${tab === 'daftar' ? 'primary-fill' : ''}`} onClick={() => setTab('daftar')}>7 hari terakhir</button>
      </div>

      {tab === 'daftar' && <DaftarTukarTambah locId={me.role === 'cabang' ? '' : lokasi} versi={versi} onUbah={onSelesai} />}

      {tab === 'baru' && (
        <form onSubmit={periksa}>
          {me.role !== 'cabang' && (
            <div className="form-grid">
              <div>
                <label>Toko</label>
                <select value={lokasi} onChange={(e) => { setLokasi(e.target.value); setBaru([]) }}>
                  {cabang.map((l) => <option key={l.id} value={l.id}>{l.name}</option>)}
                </select>
              </div>
            </div>
          )}

          <h4>Barang lama dari pelanggan</h4>
          {lama.map((x, i) => (
            <div key={i} className="panel-aksi" style={{ marginBottom: 8 }}>
              <div className="form-grid">
                <div>
                  <label>Jenis</label>
                  <select value={x.jenis} onChange={(e) => ubahLama(i, 'jenis', e.target.value)}>
                    <option value="perhiasan">Perhiasan emas</option>
                    <option value="logam_mulia">Logam mulia emas</option>
                  </select>
                </div>
                {x.jenis === 'logam_mulia'
                  ? <div><label>Merek</label><input value={x.brand} onChange={(e) => ubahLama(i, 'brand', e.target.value)} /></div>
                  : <div><label>Nama</label><input value={x.item_name} onChange={(e) => ubahLama(i, 'item_name', e.target.value)} placeholder="Cincin..." /></div>}
                {x.jenis === 'perhiasan' && (
                  <div>
                    <label>Kadar</label>
                    <select value={x.karat} onChange={(e) => ubahLama(i, 'karat', e.target.value)}>
                      {KARAT.map((n) => <option key={n} value={n}>{n}K</option>)}
                    </select>
                  </div>
                )}
                <div><label>Berat (g)</label><input type="number" step="0.001" min="0" value={x.weight_gram} onChange={(e) => ubahLama(i, 'weight_gram', e.target.value)} /></div>
                <div><label>Nilai tukar (Rp)</label><input type="number" min="0" step="1" value={x.nilai} onChange={(e) => ubahLama(i, 'nilai', e.target.value)} /></div>
                <div>
                  <label>Kondisi</label>
                  <select value={x.kondisi} onChange={(e) => ubahLama(i, 'kondisi', e.target.value)}>
                    <option value="kotor">Kotor</option>
                    <option value="bersih">Bersih</option>
                  </select>
                </div>
              </div>
              {lama.length > 1 && <button type="button" className="mini" onClick={() => setLama((l) => l.filter((_, n) => n !== i))}>Hapus barang lama</button>}
            </div>
          ))}
          <button type="button" className="mini" onClick={() => setLama((l) => [...l, { ...LAMA_KOSONG }])} disabled={lama.length >= 100}>+ Tambah barang lama</button>
          <p className="muted">Total nilai tukar: {rupiah(totalLama)}</p>

          <h4>Barang baru yang diambil</h4>
          <input value={cari} onChange={(e) => setCari(e.target.value)} placeholder="Cari nama atau kode barang di stok toko..." />
          {hasil.length > 0 && (
            <div className="panel-aksi" style={{ marginTop: 6 }}>
              {hasil.map((r) => (
                <div key={r.id} style={{ display: 'flex', justifyContent: 'space-between', gap: 8, padding: '4px 0' }}>
                  <span>{r.item_no} {r.item_name} {r.karat} {gram(r.weight_gram)}g</span>
                  <button type="button" className="mini" onClick={() => tambahBaru(r)} disabled={baru.some((x) => x.id === r.id)}>Tambah</button>
                </div>
              ))}
            </div>
          )}
          {baru.map((x, i) => (
            <div key={x.id} className="form-grid" style={{ marginTop: 8, alignItems: 'end' }}>
              <div><label>Barang</label><div>{x.nama}</div></div>
              <div><label>Harga (Rp)</label><input type="number" min="0" step="1" value={x.harga} onChange={(e) => ubahBaru(i, e.target.value)} /></div>
              <div><button type="button" className="mini" onClick={() => setBaru((l) => l.filter((_, n) => n !== i))}>Hapus</button></div>
            </div>
          ))}
          <p className="muted">Total harga barang baru: {rupiah(totalBaru)}</p>

          <div className="form-grid" style={{ marginTop: 8 }}>
            <div>
              <label>Metode pembayaran selisih</label>
              <select value={metode} onChange={(e) => setMetode(e.target.value)}>
                <option value="tunai">Tunai</option>
                <option value="qris">QRIS/Debit</option>
              </select>
            </div>
            {metode === 'qris' && selisih > 0 && (
              <div><label>Potongan QRIS/Debit (Rp)</label><input type="number" min="0" step="1" value={biaya} onChange={(e) => setBiaya(e.target.value)} placeholder="0" /></div>
            )}
            <div><label>Tanggal</label><input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} /></div>
            <div><label>Catatan</label><input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} /></div>
          </div>
          <p><strong>{teksSelisih}</strong>. Tunai dan QRIS/Debit masuk kas toko ini, bukan back office.</p>
          <button type="submit" className="mini primary-fill">Periksa dan simpan</button>
        </form>
      )}

      {konfirmasi && (
        <Popup judul="Simpan tukar tambah ini?" labelKonfirmasi="Ya, simpan" loading={loading} onKonfirmasi={simpan} onTutup={() => setKonfirmasi(false)}>
          <table className="tabel-spek">
            <tbody>
              <tr><th>Barang lama</th><td>{lama.length} barang, nilai tukar {rupiah(totalLama)}</td></tr>
              <tr><th>Barang baru</th><td>{baru.length} barang, harga {rupiah(totalBaru)}</td></tr>
              <tr><th>Selisih</th><td>{teksSelisih}</td></tr>
              <tr><th>Metode</th><td>{metode === 'qris' ? 'QRIS/Debit' : 'Tunai'}{potongan > 0 ? `, potongan ${rupiah(potongan)}` : ''}</td></tr>
              <tr><th>Tanggal</th><td>{tgl}</td></tr>
            </tbody>
          </table>
        </Popup>
      )}
    </KartuAksi>
  )
}
