import { useState, useEffect, useRef } from 'react'
import { api } from '../../services/api'
import { gram, hariIni, rupiah } from '../../utils/format'
import { KATEGORI } from '../../constants/kategori'
import { NAMA_UMUM, MEREK_LM, KARAT_UMUM, KARAT_LAIN } from '../../constants/barang'
import { barisBaru } from '../../utils/barang'
import PilihanLokasi from '../common/PilihanLokasi'

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
  const [kapasitas, setKapasitas] = useState(100)
  const [baris, setBaris] = useState([barisBaru()])
  const [fokus, setFokus] = useState(false)
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

  // daftar baki di lokasi tujuan; otomatis memilih baki pertama yang masih punya tempat
  useEffect(() => {
    if (!formBuka || !fLoc) return
    api(`/baki?location_id=${fLoc}`)
      .then((d) => {
        const list = d.rows.filter((b) => b.location_id === Number(fLoc))
        setBakiList(list)
        setKapasitas(d.kapasitas)
        setFBaki((lama) => {
          if (lama && list.some((b) => String(b.id) === lama && b.isi < d.kapasitas)) return lama
          const luang = list.find((b) => b.isi < d.kapasitas)
          return luang ? String(luang.id) : ''
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

  const bakiDipilih = bakiList.find((b) => String(b.id) === fBaki)
  const sisaBaki = bakiDipilih ? kapasitas - bakiDipilih.isi : 0
  const jumlahPerhiasan = baris.filter((b) => b.jenis === 'perhiasan').length
  const bakiKurang = bakiDipilih && jumlahPerhiasan > sisaBaki

  const totalGram = baris.reduce((s, b) => s + Number(b.weight_gram || 0), 0)
  const totalBeli = baris.reduce((s, b) => s + Number(b.harga_beli || 0), 0)

  async function simpanMasuk(e) {
    e.preventDefault()
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
            jenis: b.jenis,
            item_name: b.item_name,
            brand: b.brand,
            karat: b.karat,
            weight_gram: Number(b.weight_gram),
            harga_beli: Number(b.harga_beli || 0),
            note: b.note,
            tray_id: b.jenis === 'perhiasan' ? Number(fBaki || 0) : 0,
          })),
        }),
      })
      const message = res.message + (res.kas_keluar > 0 ? ` - kas keluar ${rupiah(res.kas_keluar)}` : '')
      const status = fPosisi === 'bo' && wajibHarga ? 'dikirim' : 'tersedia'
      setBaris([barisBaru()])
      setFNote('')
      setFPosisi('toko')
      setFormBuka(false)
      onSukses({ message, lokasiId: fLoc, status })
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <button type="button" className="ghost" onClick={() => setFormBuka(!formBuka)}>
          {formBuka ? 'Tutup form' : '+ Tambah barang masuk'}
        </button>
      </div>

      {formBuka && (
        <form
          className="card"
          style={{ marginBottom: 16 }}
          onSubmit={simpanMasuk}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && e.target.tagName === 'INPUT') e.preventDefault()
          }}
        >
          <h3>Tambah Barang Masuk</h3>
          <p className="muted">
            Satu baris = satu barang. Tekan Enter di kolom terakhir untuk menambah baris baru
            (jenis, nama, dan kadar terisi otomatis).
          </p>

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
              <label>Masukkan perhiasan ke baki</label>
              <select value={fBaki} onChange={(e) => setFBaki(e.target.value)}>
                <option value="">Tanpa baki</option>
                {bakiList.map((b) => (
                  <option key={b.id} value={b.id} disabled={b.isi >= kapasitas}>
                    Baki {b.no} ({b.isi}/{kapasitas}{b.isi >= kapasitas ? ', penuh' : ''})
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

          {bakiKurang && (
            <p className="neg" style={{ marginTop: 10 }}>
              Baki {bakiDipilih.no} hanya punya sisa tempat {sisaBaki}, sedangkan ada {jumlahPerhiasan} perhiasan.
              Kurangi barang, pilih baki lain, atau tambah baki baru di bawah daftar stok.
            </p>
          )}
          {wajibHarga && (
            <p className="muted">
              Harga beli wajib diisi. Kas toko otomatis keluar sebesar total harga beli.
              {fPosisi === 'bo' && ' Barang berstatus dalam perjalanan sampai back office mengonfirmasi.'}
            </p>
          )}

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

          <div ref={wadah}>
            <div className="baris-grid baris-head">
              <span>Jenis</span>
              <span>Nama / merek</span>
              <span>Kadar</span>
              <span>Berat (gram)</span>
              <span>Harga beli (Rp)</span>
              <span>Keterangan</span>
              <span></span>
            </div>
            {baris.map((b, i) => (
              <div className="baris-grid" key={i}>
                <select value={b.jenis} onChange={(e) => ubahBaris(i, 'jenis', e.target.value)}>
                  <option value="perhiasan">Perhiasan</option>
                  <option value="logam_mulia">Logam mulia</option>
                </select>

                {b.jenis === 'logam_mulia' ? (
                  <input
                    list="merek-lm"
                    value={b.brand}
                    onChange={(e) => ubahBaris(i, 'brand', e.target.value)}
                    placeholder="Merek (Antam...)"
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

                {b.jenis === 'logam_mulia' ? (
                  <input value="24K" disabled />
                ) : (
                  <select value={b.karat} onChange={(e) => ubahBaris(i, 'karat', e.target.value)} required>
                    <option value="">Kadar</option>
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
                  min="0.001"
                  placeholder="Berat (gram)"
                  value={b.weight_gram}
                  onChange={(e) => ubahBaris(i, 'weight_gram', e.target.value)}
                  onKeyDown={(e) => enterBaris(e, false)}
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
            style={{ marginTop: 10 }}
            onClick={() => {
              tambahBaris()
              setFokus(true)
            }}
          >
            + Tambah baris
          </button>

          <p className="muted">
            Total: {baris.length} barang, {gram(totalGram)} gram
            {totalBeli > 0 && <>, harga beli {rupiah(totalBeli)}</>}
          </p>

<<<<<<< HEAD
          <button disabled={loading}>{loading ? 'Menyimpan...' : 'Simpan semua'}</button>
        </form>
=======
          <button disabled={loading || bakiKurang}>{loading ? <><Spinner />Menyimpan...</> : 'Simpan semua'}</button>
        </m.form>
>>>>>>> 196f87a (fix fitur investor terbaru dan revmap ui interface terbaru)
      )}
    </>
  )
}
