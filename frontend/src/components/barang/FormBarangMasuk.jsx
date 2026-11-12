import { useState, useEffect, useRef } from 'react'
import { api } from '../../services/api'
import { gram, hariIni, rupiah } from '../../utils/format'
import { KATEGORI } from '../../constants/kategori'
import { NAMA_UMUM, MEREK_LM, KARAT_UMUM, KARAT_LAIN } from '../../constants/barang'
import { barisBaru, tanpaGram, bolehBaki, JENIS_LABEL } from '../../utils/barang'
import PilihanLokasi from '../common/PilihanLokasi'
import { m } from 'motion/react'
import { Plus, X } from 'lucide-react'
import AngkaBerjalan from '../common/AngkaBerjalan'
import Spinner from '../common/Spinner'
import Popup from '../common/Popup'

const formatGram = (v) => String(gram(v))
const formatRupiah = (v) => rupiah(Math.round(v))

// Tombol dan form "Tambah barang masuk" (banyak baris sekaligus).
// onMulai: bersihkan pesan; onSukses({ message, lokasiId, status }): setelah tersimpan; onError(msg).
export default function FormBarangMasuk({ me, isCabang, locations, onMulai, onSukses, onError }) {
  const [formBuka, setFormBuka] = useState(false)
  const [loading, setLoading] = useState(false)
  const [fLoc, setFLoc] = useState(isCabang ? String(me.location_id) : '')
  const [fKat, setFKat] = useState('')
  const [fTgl, setFTgl] = useState(hariIni())
  const [fNote, setFNote] = useState('')
  const [fPosisi, setFPosisi] = useState('toko')
  const [fBaki, setFBaki] = useState('')
  const [bakiList, setBakiList] = useState([])
  const [baris, setBaris] = useState([barisBaru()])
  const [fokus, setFokus] = useState(false)
  const [konfirmasi, setKonfirmasi] = useState(false)
  const wadah = useRef(null)

  const tipeLok = (id) => (locations.find((l) => l.id === Number(id)) || {}).type
  const fTipe = tipeLok(fLoc) || 'cabang'
  const fOpsi = KATEGORI.stok[fTipe].in
  const wajibHarga = fKat === 'beli_customer'

  // akun owner/admin: lokasi awal = back office
  useEffect(() => {
    const b = locations.find((l) => l.type === 'back_office')
    if (!isCabang && b) setFLoc(String(b.id))
  }, [locations, isCabang])

  useEffect(() => {
    if (fOpsi.length) setFKat(fOpsi[0][0])
  }, [fTipe])

  // daftar baki di lokasi tujuan; otomatis memilih baki pertama
  useEffect(() => {
    if (!formBuka || !fLoc) return
    api(`/baki?location_id=${fLoc}`)
      .then((d) => {
        const list = d.rows.filter((b) => b.location_id === Number(fLoc))
        setBakiList(list)
        setFBaki((lama) => {
          if (lama && list.some((b) => String(b.id) === lama)) return lama
          return list.length > 0 ? String(list[0].id) : ''
        })
      })
      .catch(() => {})
  }, [formBuka, fLoc])

  useEffect(() => {
    if (fokus && wadah.current) {
      const el = wadah.current.querySelectorAll('.in-berat')
      if (el.length) el[el.length - 1].focus()
      setFokus(false)
    }
  }, [baris, fokus])

  function ubahBaris(i, k, v) {
    setBaris((b) => b.map((x, idx) => (idx === i ? { ...x, [k]: v } : x)))
  }
  function tambahBaris() {
    setBaris((b) => [...b, barisBaru(b[b.length - 1])])
  }
  function hapusBaris(i) {
    setBaris((b) => (b.length === 1 ? b : b.filter((_, idx) => idx !== i)))
  }
  function enterBaris(e, dariHarga) {
    if (e.key !== 'Enter') return
    e.preventDefault()
    if (!dariHarga && wajibHarga) {
      const h = e.currentTarget.parentElement.querySelector('.in-harga')
      if (h) h.focus()
      return
    }
    tambahBaris()
    setFokus(true)
  }

  // susuk, tempat, dan kotak perhiasan: satu baris bisa berisi banyak barang (kolom jumlah)
  const jumlahBaris = (b) => (tanpaGram(b.jenis) ? Math.max(1, Math.floor(Number(b.jumlah) || 1)) : 1)
  const totalBarang = baris.reduce((s, b) => s + jumlahBaris(b), 0)
  const totalGram = baris.reduce((s, b) => s + (tanpaGram(b.jenis) ? 0 : Number(b.weight_gram || 0)), 0)
  const totalBeli = baris.reduce((s, b) => s + Number(b.harga_beli || 0) * jumlahBaris(b), 0)
  // di form: 'perak' = logam mulia perak, 'perhiasan_perak' = perhiasan perak (kadar 925)
  const jenisServer = (j) => (j === 'perak' ? 'logam_mulia' : j === 'perhiasan_perak' ? 'perhiasan' : j)
  const labelJenis = (j) =>
    j === 'perak' ? 'Logam mulia perak' : j === 'perhiasan_perak' ? 'Perhiasan perak' : j === 'logam_mulia' ? 'Logam mulia emas' : JENIS_LABEL[j]
  const namaBaris = (b) =>
    b.jenis === 'logam_mulia' || b.jenis === 'perak' ? b.brand : b.item_name.trim() || JENIS_LABEL[b.jenis]
  const kadarBaris = (b) =>
    b.jenis === 'perak' ? '999' : b.jenis === 'perhiasan_perak' ? '925' : b.jenis === 'logam_mulia' ? '24K' : b.karat || '-'
  // baki sama dengan server: tempat dan kotak perhiasan tidak pernah masuk baki
  const masukBaki = (b) => bolehBaki(jenisServer(b.jenis))
  const namaBaki = (bakiList.find((x) => String(x.id) === fBaki) || {}).name
  const teksBaki = !namaBaki || !baris.some(masukBaki)
    ? 'Tanpa baki'
    : baris.every(masukBaki) ? namaBaki : `${namaBaki} (tempat dan kotak perhiasan tanpa baki)`

  // tampilkan card konfirmasi (barang, jumlah, harga) sebelum benar-benar tersimpan
  function periksaMasuk(e) {
    e.preventDefault()
    onMulai()
    if (totalBarang > 300) return onError('Maksimal 300 barang sekali input')
    setKonfirmasi(true)
  }

  async function simpanMasuk() {
    onMulai()
    setLoading(true)
    try {
      const res = await api('/barang', {
        method: 'POST',
        body: JSON.stringify({
          location_id: Number(fLoc),
          category: fKat,
          entry_date: fTgl,
          note: fNote,
          posisi: wajibHarga ? fPosisi : '',
          items: baris.map((b) => ({
            jenis: jenisServer(b.jenis),
            bahan: b.jenis === 'perak' || b.jenis === 'perhiasan_perak' ? 'perak' : 'emas',
            item_name: b.item_name,
            brand: b.brand,
            karat: b.karat,
            weight_gram: Number(b.weight_gram || 0),
            harga_beli: Number(b.harga_beli || 0),
            note: b.note,
            tray_id: Number(fBaki || 0),
            jumlah: jumlahBaris(b),
          })),
        }),
      })
      const message = res.message + (res.kas_keluar > 0 ? ` - kas keluar ${rupiah(res.kas_keluar)}` : '')
      const status = fPosisi === 'bo' && wajibHarga ? 'dikirim' : 'tersedia'
      setBaris([barisBaru()])
      setFNote('')
      setFPosisi('toko')
      setKonfirmasi(false)
      setFormBuka(false)
      onSukses({ message, lokasiId: fLoc, status })
    } catch (err) {
      setKonfirmasi(false)
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <button type="button" className="ghost" onClick={() => setFormBuka(!formBuka)}>
          {formBuka ? <><X size={16} aria-hidden="true" />Tutup form</> : <><Plus size={16} aria-hidden="true" />Tambah barang masuk</>}
        </button>
      </div>

      {formBuka && (
        <m.form
          className="card"
          style={{ marginBottom: 16 }}
          onSubmit={periksaMasuk}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && e.target.tagName === 'INPUT') e.preventDefault()
          }}
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.22, ease: 'easeOut' }}
        >
          <h3>Tambah Barang Masuk</h3>
          <p className="muted">
            Satu baris = satu barang (susuk, tempat, dan kotak perhiasan bisa diisi jumlahnya). Tekan Enter di kolom terakhir untuk menambah baris baru
            (jenis, nama, dan kadar terisi otomatis).
          </p>

          <div className="form-section">
          <div className="section-title">Informasi masuk</div>
          <div className="form-grid">
            {!isCabang && (
              <div>
                <label>Lokasi</label>
                <PilihanLokasi locations={locations} value={fLoc} onChange={(e) => setFLoc(e.target.value)} />
              </div>
            )}
            <div>
              <label>Asal barang</label>
              <select value={fKat} onChange={(e) => setFKat(e.target.value)}>
                {fOpsi.map(([k, l]) => (
                  <option key={k} value={k}>{l}</option>
                ))}
              </select>
            </div>
            {wajibHarga && (
              <div>
                <label>Posisi barang</label>
                <select value={fPosisi} onChange={(e) => setFPosisi(e.target.value)}>
                  <option value="toko">Tetap di toko</option>
                  <option value="bo">Antar ke back office</option>
                </select>
              </div>
            )}
            <div>
              <label>Masukkan barang ke baki</label>
              <select value={fBaki} onChange={(e) => setFBaki(e.target.value)}>
                <option value="">Tanpa baki</option>
                {bakiList.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name} ({b.isi} barang)
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label>Tanggal</label>
              <input type="date" value={fTgl} onChange={(e) => setFTgl(e.target.value)} required />
            </div>
            <div>
              <label>Catatan untuk semua barang</label>
              <input value={fNote} onChange={(e) => setFNote(e.target.value)} />
            </div>
          </div>

          {wajibHarga && (
            <p className="muted" style={{ marginTop: 10 }}>
              Harga beli wajib diisi. Kas toko otomatis keluar sebesar total harga beli.
              {fPosisi === 'bo' && ' Barang berstatus dalam perjalanan sampai back office mengonfirmasi.'}
            </p>
          )}
          </div>

          <datalist id="nama-umum">
            {NAMA_UMUM.map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>
          <datalist id="merek-lm">
            {MEREK_LM.map((n) => (
              <option key={n} value={n} />
            ))}
          </datalist>

          <div className="form-section">
          <div className="section-title">Daftar barang</div>
          <div ref={wadah}>
            <div className="baris-grid baris-head">
              <span>Jenis</span>
              <span>Nama / merek</span>
              <span>Kadar</span>
              <span>Berat (gram)</span>
              <span>Jumlah</span>
              <span>Harga beli (Rp)</span>
              <span>Keterangan</span>
              <span></span>
            </div>
            {baris.map((b, i) => (
              <div className="baris-grid" key={i}>
                <select value={b.jenis} onChange={(e) => ubahBaris(i, 'jenis', e.target.value)}>
                  <option value="perhiasan">Perhiasan</option>
                  <option value="logam_mulia">Logam mulia emas</option>
                  <option value="perak">Logam mulia perak</option>
                  <option value="perhiasan_perak">Perhiasan perak</option>
                  <option value="susuk">Susuk emas</option>
                  <option value="tempat_perhiasan">Tempat perhiasan</option>
                  <option value="kotak_perhiasan">Kotak perhiasan</option>
                </select>

                {tanpaGram(b.jenis) ? (
                  <input
                    value={b.item_name}
                    onChange={(e) => ubahBaris(i, 'item_name', e.target.value)}
                    placeholder={`${JENIS_LABEL[b.jenis]} (opsional)`}
                    aria-label="Nama barang (opsional)"
                  />
                ) : b.jenis !== 'perhiasan' && b.jenis !== 'perhiasan_perak' ? (
                  <input
                    list={b.jenis === 'perak' ? undefined : 'merek-lm'}
                    value={b.brand}
                    onChange={(e) => ubahBaris(i, 'brand', e.target.value)}
                    placeholder={b.jenis === 'perak' ? 'Merek (Fine Silver...)' : 'Merek (Antam...)'}
                    required
                  />
                ) : (
                  <input
                    list="nama-umum"
                    value={b.item_name}
                    onChange={(e) => ubahBaris(i, 'item_name', e.target.value)}
                    placeholder="Cincin"
                    required
                  />
                )}

                {b.jenis === 'tempat_perhiasan' || b.jenis === 'kotak_perhiasan' ? (
                  <input value="-" disabled aria-label="Kadar" />
                ) : b.jenis === 'perhiasan_perak' ? (
                  <select value="925" disabled aria-label="Kadar">
                    <option value="925">925</option>
                  </select>
                ) : b.jenis !== 'perhiasan' && b.jenis !== 'susuk' ? (
                  <input value={b.jenis === 'perak' ? '999' : '24K'} disabled />
                ) : (
                  <select
                    value={b.karat}
                    onChange={(e) => ubahBaris(i, 'karat', e.target.value)}
                    required={b.jenis === 'perhiasan'}
                    aria-label="Kadar"
                  >
                    <option value="">{b.jenis === 'susuk' ? 'Kadar (opsional)' : 'Kadar'}</option>
                    <optgroup label="Umum">
                      {KARAT_UMUM.map((n) => (
                        <option key={n} value={`${n}K`}>{n}K</option>
                      ))}
                    </optgroup>
                    <optgroup label="Lainnya">
                      {KARAT_LAIN.map((n) => (
                        <option key={n} value={`${n}K`}>{n}K</option>
                      ))}
                    </optgroup>
                  </select>
                )}

                <input
                  className="in-berat"
                  type="number"
                  step="0.001"
                  min={tanpaGram(b.jenis) ? '0' : '0.001'}
                  placeholder={tanpaGram(b.jenis) ? 'Berat (opsional)' : 'Berat (gram)'}
                  value={b.weight_gram}
                  onChange={(e) => ubahBaris(i, 'weight_gram', e.target.value)}
                  onKeyDown={(e) => enterBaris(e, false)}
                  required={!tanpaGram(b.jenis)}
                />
                <input
                  type="number"
                  min="1"
                  max="300"
                  step="1"
                  value={tanpaGram(b.jenis) ? b.jumlah : '1'}
                  onChange={(e) => ubahBaris(i, 'jumlah', e.target.value)}
                  disabled={!tanpaGram(b.jenis)}
                  aria-label="Jumlah barang"
                  required
                />
                <input
                  className="in-harga"
                  type="number"
                  min="0"
                  value={b.harga_beli}
                  onChange={(e) => ubahBaris(i, 'harga_beli', e.target.value)}
                  onKeyDown={(e) => enterBaris(e, true)}
                  placeholder={wajibHarga ? 'Harga beli (Rp)' : 'Harga beli (opsional)'}
                  required={wajibHarga}
                />
                <input
                  value={b.note}
                  onChange={(e) => ubahBaris(i, 'note', e.target.value)}
                  placeholder="Polos, ukir..."
                />
                <button type="button" className="mini" onClick={() => hapusBaris(i)}>Hapus</button>
              </div>
            ))}
          </div>

          <button
            type="button"
            className="ghost"
            style={{ marginTop: 12 }}
            onClick={() => {
              tambahBaris()
              setFokus(true)
            }}
          >
            <Plus size={16} aria-hidden="true" />Tambah baris
          </button>
          </div>

          <div className="total-bar">
            <span>Total: <b>{totalBarang}</b> barang</span>
            <span><b className="utama"><AngkaBerjalan nilai={totalGram} format={formatGram} /></b> gram</span>
            {totalBeli > 0 && <span>harga beli <b className="utama"><AngkaBerjalan nilai={totalBeli} format={formatRupiah} /></b></span>}
          </div>

          <button disabled={loading}>{loading ? <><Spinner />Menyimpan...</> : 'Lanjut'}</button>
        </m.form>
      )}

      {konfirmasi && (
        <Popup
          judul={`Simpan ${totalBarang} barang masuk?`}
          labelKonfirmasi="Ya, simpan"
          loading={loading}
          onKonfirmasi={simpanMasuk}
          onTutup={() => setKonfirmasi(false)}
        >
          <p className="muted">Periksa barang dan harga sebelum disimpan:</p>
          <div className="table-wrap" style={{ maxHeight: 260, overflow: 'auto' }}>
            <table className="tabel-spek">
              <thead>
                <tr><th>Barang</th><th>Kadar</th><th className="num">Berat</th><th className="num">Jumlah</th><th className="num">Harga beli</th></tr>
              </thead>
              <tbody>
                {baris.map((b, i) => (
                  <tr key={i}>
                    <td>{labelJenis(b.jenis)}{namaBaris(b) && namaBaris(b) !== JENIS_LABEL[b.jenis] ? `: ${namaBaris(b)}` : ''}</td>
                    <td>{kadarBaris(b)}</td>
                    <td className="num">{Number(b.weight_gram) > 0 ? `${gram(b.weight_gram)} g` : '-'}</td>
                    <td className="num">{jumlahBaris(b)}</td>
                    <td className="num">{Number(b.harga_beli) > 0 ? rupiah(Number(b.harga_beli)) : '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <table className="tabel-spek" style={{ marginTop: 12 }}>
            <tbody>
              <tr><th>Lokasi</th><td>{(locations.find((l) => l.id === Number(fLoc)) || {}).name || '-'}</td></tr>
              <tr><th>Asal barang</th><td>{(fOpsi.find(([k]) => k === fKat) || [])[1] || fKat}</td></tr>
              <tr><th>Baki</th><td>{teksBaki}</td></tr>
              <tr><th>Tanggal</th><td>{fTgl}</td></tr>
              <tr><th>Total</th><td>{totalBarang} barang, {gram(totalGram)} g emas</td></tr>
              {totalBeli > 0 && <tr><th>Total harga beli</th><td>{rupiah(totalBeli)}{wajibHarga && ' (kas toko keluar)'}</td></tr>}
            </tbody>
          </table>
        </Popup>
      )}
    </>
  )
}
