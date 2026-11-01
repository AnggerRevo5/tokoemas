import { useCallback, useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { m } from 'motion/react'
import { X } from 'lucide-react'
import { api } from '../../services/api'
import { gram, hariIni, rupiah } from '../../utils/format'
import Popup from '../common/Popup'
import Pesan from '../common/Pesan'
import SpesifikasiBarang, { DaftarSpesifikasi } from './SpesifikasiBarang'

const LABEL_STATUS = {
  aktif: 'Berjalan',
  lunas: 'Lunas, menunggu diambil di pegadaian',
  kembali: 'Sudah kembali ke toko',
  tidak_ditebus: 'Tidak ditebus',
}
const KELAS_STATUS = { aktif: 'tag warn', lunas: 'tag info', kembali: 'tag in', tidak_ditebus: 'tag gray' }
const JUDUL_AKSI = {
  bayar: 'Bayar cicilan',
  tebus: 'Tebus lebih cepat',
  kembali: 'Barang sudah kembali ke toko',
  tidak: 'Tidak ditebus',
  batal: 'Batalkan gadai',
  void: 'Batalkan pembayaran',
}

const kosong = () => ({ tgl: hariIni(), amount: '', denda: '', diskon: '', alasan: '', note: '' })

// Card gadai (satu akad, satu atau beberapa barang): ringkasan, jadwal cicilan, riwayat pembayaran, dan langkah berikutnya.
// Setiap langkah menampilkan card konfirmasi berisi spesifikasi barang dan rinciannya sebelum disimpan.
// onUbah(pesan): ada perubahan data (daftar Stok dimuat ulang); onTutup().
export default function CardGadai({ item, me, onUbah, onTutup }) {
  const [g, setG] = useState(null)
  const [error, setError] = useState('')
  const [aksi, setAksi] = useState(null) // { jenis, p }
  const [f, setF] = useState(kosong)
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)
  const ubah = (k) => (e) => setF((x) => ({ ...x, [k]: e.target.value }))

  const muat = useCallback(() => {
    api(`/gadai?item_id=${item.id}&status=semua`)
      .then((daftar) => {
        if (daftar.length === 0) throw new Error('Data gadai barang ini tidak ditemukan')
        return api(`/gadai/${daftar[0].id}`)
      })
      .then(setG)
      .catch((e) => setError(e.message))
  }, [item.id])

  useEffect(() => {
    muat()
  }, [muat])

  useEffect(() => {
    const esc = (e) => e.key === 'Escape' && !konfirmasi && !loading && onTutup()
    window.addEventListener('keydown', esc)
    return () => window.removeEventListener('keydown', esc)
  }, [onTutup, konfirmasi, loading])

  const paket = !!g && g.barang.length > 1
  // DaftarSpesifikasi memakai kolom id, sedangkan data gadai memakai item_id
  const daftarBarang = paket ? g.barang.map((b) => ({ ...b, id: b.item_id })) : []
  const kelola = !!g && (me.role === 'super_admin' || (me.role === 'cabang' && g.location_id === me.location_id))
  const adaBayar = !!g && g.pembayaran.some((p) => !p.is_void)
  const sisa = g ? g.sisa : 0
  const amount = Number(f.amount) || 0
  const denda = Number(f.denda) || 0
  const diskon = Number(f.diskon) || 0

  function mulai(jenis, p) {
    setError('')
    setAksi({ jenis, p })
    setF({ ...kosong(), amount: jenis === 'bayar' ? String(Math.min(g.cicilan, g.sisa)) : '' })
    // pembatalan langsung berupa satu card (isian alasan ada di card konfirmasinya)
    if (jenis === 'batal' || jenis === 'void') setKonfirmasi(true)
  }

  // isi card konfirmasi dan permintaan ke server untuk langkah yang sedang dipilih
  function rencana() {
    const { jenis, p } = aksi
    switch (jenis) {
      case 'bayar':
        return {
          path: `/gadai/${g.id}/bayar`,
          body: { date: f.tgl, amount, denda, note: f.note.trim() },
          baris: [
            ['Cicilan ke', `${g.bulan_dibayar + 1} dari ${g.tenor}`],
            ['Tanggal bayar', f.tgl],
            ['Jumlah cicilan', rupiah(amount)],
            ['Denda keterlambatan', rupiah(denda)],
            ['Kas keluar', rupiah(amount + denda)],
            ['Sisa kewajiban sesudahnya', rupiah(Math.max(0, sisa - amount))],
          ],
        }
      case 'tebus':
        return {
          path: `/gadai/${g.id}/tebus`,
          body: { date: f.tgl, amount: sisa - diskon, diskon, denda, note: f.note.trim() },
          baris: [
            ['Tanggal tebus', f.tgl],
            ['Sisa kewajiban', rupiah(sisa)],
            ['Diskon pelunasan', rupiah(diskon)],
            ['Jumlah dibayar', rupiah(sisa - diskon)],
            ['Denda keterlambatan', rupiah(denda)],
            ['Kas keluar', rupiah(sisa - diskon + denda)],
          ],
        }
      case 'kembali':
        return { path: `/gadai/${g.id}/kembali`, body: { date: f.tgl }, baris: [['Tanggal kembali', f.tgl], ['Barang', `${paket ? 'Semua barang masuk' : 'Masuk'} lagi ke stok toko, status Tersedia`]] }
      case 'tidak':
        return {
          path: `/gadai/${g.id}/tidak-ditebus`,
          body: { date: f.tgl, reason: f.alasan.trim() },
          baris: [['Tanggal', f.tgl], ['Alasan', f.alasan.trim()], ['Akibat', `${paket ? 'Semua barang' : 'Barang'} tidak ada lagi di stok toko`]],
          danger: true,
        }
      case 'batal':
        return {
          path: `/gadai/${g.id}/batal`,
          body: { reason: f.alasan.trim() },
          baris: [['Alasan', f.alasan.trim()], ['Akibat', `Dana gadai dikeluarkan lagi dari kas dan ${paket ? 'semua barang' : 'barang'} kembali Tersedia`]],
          danger: true,
        }
      default:
        return {
          path: `/gadai-bayar/${p.id}/void`,
          body: { reason: f.alasan.trim() },
          baris: [
            ['Pembayaran', `${p.tx_date}, ${rupiah(p.amount)}${p.denda > 0 ? ` + denda ${rupiah(p.denda)}` : ''}`],
            ['Alasan', f.alasan.trim()],
            ['Akibat', 'Kas pembayaran itu dikembalikan dan sisa kewajiban bertambah'],
          ],
          danger: true,
        }
    }
  }

  function periksa(e) {
    e.preventDefault()
    setError('')
    const { jenis } = aksi
    if (jenis === 'bayar' && amount <= 0) return setError('Jumlah cicilan wajib diisi')
    if (jenis === 'bayar' && amount > sisa) return setError(`Cicilan melebihi sisa ${rupiah(sisa)}`)
    if (jenis === 'tebus' && diskon > sisa) return setError('Diskon tidak boleh melebihi sisa kewajiban')
    if ((jenis === 'tidak' || jenis === 'batal' || jenis === 'void') && !f.alasan.trim()) return setError('Alasan wajib diisi')
    setKonfirmasi(true)
  }

  async function simpan() {
    const r = rencana()
    setLoading(true)
    try {
      const res = await api(r.path, { method: 'POST', body: JSON.stringify(r.body) })
      setKonfirmasi(false)
      setAksi(null)
      onUbah(res.message)
      muat()
    } catch (err) {
      setKonfirmasi(false)
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const r = aksi && konfirmasi ? rencana() : null
  const langsung = !!aksi && (aksi.jenis === 'batal' || aksi.jenis === 'void')

  return createPortal(
    <div
      className="popup-latar latar-lebar"
      onMouseDown={(e) => e.target === e.currentTarget && !konfirmasi && !loading && onTutup()}
    >
      <m.div
        className="popup popup-lebar"
        role="dialog"
        aria-modal="true"
        aria-label={`Gadai ${item.item_no}`}
        initial={{ opacity: 0, scale: 0.97, y: 8 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        transition={{ duration: 0.18, ease: 'easeOut' }}
      >
        <div className="popup-kepala">
          <h3>{paket ? `Gadai ${g.barang.length} barang` : `Gadai ${item.item_no} - ${item.item_name} ${item.karat}`}</h3>
          <button type="button" className="mini popup-tutup" onClick={onTutup} disabled={loading} aria-label="Tutup">
            <X size={16} aria-hidden="true" />
          </button>
        </div>
        <Pesan error={error} />
        {!g && !error && <p className="muted">Memuat...</p>}

        {g && (
          <>
            {paket && (
              <>
                <h4 style={{ margin: '12px 0 6px' }}>Barang yang digadaikan ({g.barang.length} barang, {gram(g.weight_gram)} g)</h4>
                <DaftarSpesifikasi items={daftarBarang} />
              </>
            )}
            <table className="tabel-spek" style={{ marginTop: 12 }}>
              <tbody>
                <tr><th>Status</th><td><span className={KELAS_STATUS[g.status]}>{LABEL_STATUS[g.status]}</span></td></tr>
                <tr><th>Pegadaian</th><td>{g.pegadaian}</td></tr>
                <tr><th>Tanggal gadai</th><td>{g.start_date}</td></tr>
                <tr><th>Dana diterima</th><td>{rupiah(g.dana)} (kas {g.lokasi})</td></tr>
                <tr><th>Cicilan</th><td>{g.tenor} x {rupiah(g.cicilan)} = {rupiah(g.total)}</td></tr>
                <tr><th>Sudah dibayar</th><td>{rupiah(g.dibayar)}{g.diskon > 0 && ` (diskon ${rupiah(g.diskon)})`}{g.denda > 0 && `, denda ${rupiah(g.denda)}`}</td></tr>
                {g.status === 'aktif' && (
                  <>
                    <tr><th>Sisa kewajiban</th><td><b>{rupiah(g.sisa)}</b></td></tr>
                    <tr><th>Sisa bulan</th><td><b>{g.sisa_bulan}</b> bulan</td></tr>
                    <tr>
                      <th>Jatuh tempo berikut</th>
                      <td className={g.terlambat ? 'neg' : undefined}>{g.jatuh_tempo_berikut}{g.terlambat && ' (terlambat)'}</td>
                    </tr>
                  </>
                )}
                {g.end_date && <tr><th>Selesai</th><td>{g.end_date}{g.end_note && `, ${g.end_note}`}</td></tr>}
                {g.note && <tr><th>Catatan</th><td>{g.note}</td></tr>}
              </tbody>
            </table>

            <h4 style={{ marginBottom: 6 }}>Jadwal cicilan</h4>
            <div className="table-wrap" style={{ maxHeight: 200, overflow: 'auto' }}>
              <table>
                <thead><tr><th>Bulan</th><th>Jatuh tempo</th><th>Status</th></tr></thead>
                <tbody>
                  {g.jadwal.map((j) => (
                    <tr key={j.ke}>
                      <td>{j.ke}</td>
                      <td>{j.jatuh_tempo}</td>
                      <td>
                        {j.dibayar ? (
                          <span className="tag in">{j.via === 'pelunasan' ? `Dilunasi ${j.tgl_bayar}` : `Dibayar ${j.tgl_bayar}`}</span>
                        ) : g.status === 'aktif' && j.jatuh_tempo < hariIni() ? (
                          <span className="tag out">Terlambat</span>
                        ) : (
                          <span className="tag gray">Belum</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <h4 style={{ marginBottom: 6 }}>Riwayat pembayaran</h4>
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Tanggal</th><th>Jenis</th><th className="num">Jumlah</th><th className="num">Denda</th>
                    <th className="num">Diskon</th><th>Catatan</th><th></th>
                  </tr>
                </thead>
                <tbody>
                  {g.pembayaran.length === 0 && <tr><td colSpan="7" className="muted">Belum ada pembayaran</td></tr>}
                  {g.pembayaran.map((p) => (
                    <tr key={p.id} className={p.is_void ? 'muted' : undefined}>
                      <td>{p.tx_date}</td>
                      <td>{p.jenis === 'pelunasan' ? 'Tebus' : 'Cicilan'}{p.is_void && ' (dibatalkan)'}</td>
                      <td className="num">{rupiah(p.amount)}</td>
                      <td className="num">{p.denda > 0 ? rupiah(p.denda) : '-'}</td>
                      <td className="num">{p.diskon > 0 ? rupiah(p.diskon) : '-'}</td>
                      <td>{p.is_void ? p.void_reason : p.note}</td>
                      <td>
                        {p.bisa_ubah && (
                          <button type="button" className="mini" disabled={!!aksi} onClick={() => mulai('void', p)}>Batalkan</button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            {!aksi && (
              <div className="aksi-tombol" style={{ marginTop: 14 }}>
                {kelola && g.status === 'aktif' && (
                  <>
                    <button type="button" className="mini primary-fill" onClick={() => mulai('bayar')}>Bayar cicilan</button>
                    <button type="button" className="mini primary" onClick={() => mulai('tebus')}>Tebus lebih cepat</button>
                    <button type="button" className="mini" onClick={() => mulai('tidak')}>Tidak ditebus</button>
                  </>
                )}
                {kelola && g.status === 'lunas' && (
                  <button type="button" className="mini primary-fill" onClick={() => mulai('kembali')}>Sudah kembali ke toko</button>
                )}
                {g.bisa_ubah && g.status === 'aktif' && !adaBayar && (
                  <button type="button" className="mini danger-fill" onClick={() => mulai('batal')}>Batalkan gadai</button>
                )}
              </div>
            )}

            {aksi && !langsung && (
              <form className="panel-aksi" onSubmit={periksa} style={{ marginTop: 14 }}>
                <h4 style={{ margin: '0 0 8px' }}>{JUDUL_AKSI[aksi.jenis]}</h4>
                <div className="form-grid">
                  {aksi.jenis !== 'batal' && aksi.jenis !== 'void' && (
                    <div>
                      <label>Tanggal</label>
                      <input type="date" value={f.tgl} onChange={ubah('tgl')} required />
                    </div>
                  )}
                  {aksi.jenis === 'bayar' && (
                    <div>
                      <label>Jumlah cicilan (Rp)</label>
                      <input type="number" min="1" value={f.amount} onChange={ubah('amount')} required />
                    </div>
                  )}
                  {aksi.jenis === 'tebus' && (
                    <>
                      <div>
                        <label>Sisa kewajiban</label>
                        <input value={rupiah(sisa)} disabled />
                      </div>
                      <div>
                        <label>Diskon pelunasan (Rp)</label>
                        <input type="number" min="0" value={f.diskon} onChange={ubah('diskon')} />
                      </div>
                      <div>
                        <label>Jumlah yang dibayar</label>
                        <input value={rupiah(Math.max(0, sisa - diskon))} disabled />
                      </div>
                    </>
                  )}
                  {(aksi.jenis === 'bayar' || aksi.jenis === 'tebus') && (
                    <div>
                      <label>Bunga/denda keterlambatan (Rp, opsional)</label>
                      <input type="number" min="0" value={f.denda} onChange={ubah('denda')} />
                    </div>
                  )}
                  {(aksi.jenis === 'tidak' || aksi.jenis === 'batal' || aksi.jenis === 'void') && (
                    <div className="span2">
                      <label>Alasan</label>
                      <input value={f.alasan} onChange={ubah('alasan')} maxLength={250} required />
                    </div>
                  )}
                  {(aksi.jenis === 'bayar' || aksi.jenis === 'tebus') && (
                    <div className="span2">
                      <label>Catatan (opsional)</label>
                      <input value={f.note} onChange={ubah('note')} maxLength={200} />
                    </div>
                  )}
                </div>
                <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
                  <button>Lanjut</button>
                  <button type="button" className="ghost" onClick={() => { setAksi(null); setError('') }}>Batal</button>
                </div>
              </form>
            )}
          </>
        )}
      </m.div>

      {r && (
        <Popup
          judul={`${JUDUL_AKSI[aksi.jenis]}?`}
          labelKonfirmasi="Ya, lanjutkan"
          danger={!!r.danger}
          loading={loading}
          onKonfirmasi={simpan}
          nonaktif={langsung && !f.alasan.trim()}
          onTutup={() => {
            setKonfirmasi(false)
            if (langsung) setAksi(null)
          }}
        >
          <p className="muted">Periksa barang dan rinciannya sebelum disimpan:</p>
          {paket ? <DaftarSpesifikasi items={daftarBarang} /> : <SpesifikasiBarang item={item} />}
          <table className="tabel-spek" style={{ marginTop: 12 }}>
            <tbody>
              <tr><th>Pegadaian</th><td>{g.pegadaian}</td></tr>
              {r.baris.filter(([label]) => !(langsung && label === 'Alasan')).map(([label, nilai]) => (
                <tr key={label}><th>{label}</th><td>{nilai}</td></tr>
              ))}
            </tbody>
          </table>
          {langsung && (
            <>
              <label style={{ display: 'block', marginTop: 12 }}>Alasan</label>
              <input
                value={f.alasan}
                onChange={ubah('alasan')}
                placeholder="Contoh: salah input"
                maxLength={250}
                autoFocus
              />
            </>
          )}
        </Popup>
      )}
    </div>,
    document.body,
  )
}
