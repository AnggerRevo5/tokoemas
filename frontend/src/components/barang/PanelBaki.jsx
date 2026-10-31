import { useEffect, useRef, useState } from 'react'
import { ChevronDown, Pencil, Plus, X } from 'lucide-react'
import { api } from '../../services/api'
import Spinner from '../common/Spinner'
import Popup from '../common/Popup'

// Ringkasan baki (perhiasan dan logam mulia) di lokasi yang dipilih: dropdown pencari untuk memfilter baki (aman untuk ratusan baki),
// tambah baki, dan hapus baki kosong.
// bolehTambah: false bila lokasi belum dipilih (mode "Semua lokasi"). onUbah(pesan): setelah baki ditambah/dihapus.
export default function PanelBaki({ baki, kapasitas, bakiF, onPilih, lokasiId, bolehTambah, onUbah, onError }) {
  const [loading, setLoading] = useState(false)
  const [hapus, setHapus] = useState(null)
  const [form, setForm] = useState(null) // { id?, nama }: kartu tambah baki atau ubah nama
  const [galat, setGalat] = useState('')
  const [buka, setBuka] = useState(false)
  const [cari, setCari] = useState('')
  const [saring, setSaring] = useState('semua')
  const wadah = useRef(null)

  // dropdown ditutup saat klik di luar atau tekan Esc
  useEffect(() => {
    if (!buka) return undefined
    const luar = (e) => wadah.current && !wadah.current.contains(e.target) && setBuka(false)
    const esc = (e) => e.key === 'Escape' && setBuka(false)
    document.addEventListener('mousedown', luar)
    window.addEventListener('keydown', esc)
    return () => {
      document.removeEventListener('mousedown', luar)
      window.removeEventListener('keydown', esc)
    }
  }, [buka])

  // baki diberi nama sendiri (bukan nomor otomatis); card yang sama dipakai untuk tambah baki dan ubah nama
  async function simpanNama() {
    if (!form.nama.trim()) return
    setLoading(true)
    setGalat('')
    try {
      const res = form.id
        ? await api(`/baki/${form.id}/nama`, { method: 'POST', body: JSON.stringify({ name: form.nama }) })
        : await api('/baki', { method: 'POST', body: JSON.stringify({ location_id: Number(lokasiId), name: form.nama }) })
      setForm(null)
      onUbah(res.message)
    } catch (err) {
      setGalat(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function konfirmasiHapus() {
    setLoading(true)
    try {
      const res = await api(`/baki/${hapus.id}/hapus`, { method: 'POST' })
      if (bakiF === String(hapus.id)) onPilih('')
      setHapus(null)
      onUbah(res.message)
    } catch (err) {
      setHapus(null)
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const terpilih = baki.find((b) => String(b.id) === bakiF)
  const jml = {
    kosong: baki.filter((b) => b.isi === 0).length,
    penuh: baki.filter((b) => b.isi >= kapasitas).length,
  }
  jml.terisi = baki.length - jml.kosong

  const cocok = (b) => {
    if (saring === 'kosong' && b.isi !== 0) return false
    if (saring === 'terisi' && b.isi === 0) return false
    if (saring === 'penuh' && b.isi < kapasitas) return false
    const q = cari.trim().toLowerCase()
    return !q || (b.name || '').toLowerCase().includes(q) || (b.lokasi || '').toLowerCase().includes(q)
  }
  const tampil = baki.filter(cocok)
  // mode "Semua lokasi": nomor baki bisa sama antar lokasi, jadi dikelompokkan per lokasi
  const kelompok = []
  for (const b of tampil) {
    const nama = lokasiId === '0' ? b.lokasi : ''
    let k = kelompok.find((x) => x.nama === nama)
    if (!k) {
      k = { nama, isi: [] }
      kelompok.push(k)
    }
    k.isi.push(b)
  }

  function pilih(id) {
    onPilih(id)
    setBuka(false)
  }

  return (
    <div className="card" style={{ marginTop: 16 }}>
      <div className="baki-bar">
        <b>Baki</b>
        <div className="baki-pilih" ref={wadah}>
          <button
            type="button"
            className="mini baki-pemicu"
            aria-haspopup="listbox"
            aria-expanded={buka}
            onClick={() => setBuka((x) => !x)}
          >
            {terpilih
              ? `${terpilih.name}${lokasiId === '0' ? ` (${terpilih.lokasi})` : ''}: ${terpilih.isi}/${kapasitas}`
              : 'Semua baki'}
            <ChevronDown size={14} aria-hidden="true" />
          </button>
          {buka && (
            <div className="baki-daftar">
              <input
                className="baki-cari"
                value={cari}
                onChange={(e) => setCari(e.target.value)}
                placeholder="Cari nama baki"
                inputMode="search"
                autoFocus
              />
              <div className="baki-saring">
                {[
                  ['semua', 'Semua', baki.length],
                  ['terisi', 'Terisi', jml.terisi],
                  ['kosong', 'Kosong', jml.kosong],
                  ['penuh', 'Penuh', jml.penuh],
                ].map(([k, l, n]) => (
                  <button
                    key={k}
                    type="button"
                    className={saring === k ? 'mini primary-fill' : 'mini'}
                    onClick={() => setSaring(k)}
                  >
                    {l} {n}
                  </button>
                ))}
              </div>
              <div className="baki-gulung" role="listbox" aria-label="Daftar baki">
                <button
                  type="button"
                  role="option"
                  aria-selected={bakiF === ''}
                  className={bakiF === '' ? 'baki-baris aktif' : 'baki-baris'}
                  onClick={() => pilih('')}
                >
                  Semua baki
                </button>
                {tampil.length === 0 && <p className="muted baki-kosong">Tidak ada baki yang cocok.</p>}
                {kelompok.map((k) => (
                  <div key={k.nama || 'satu'}>
                    {k.nama && <div className="baki-lokasi">{k.nama}</div>}
                    {k.isi.map((b) => (
                      <div key={b.id} className="baki-chip">
                        <button
                          type="button"
                          role="option"
                          aria-selected={String(b.id) === bakiF}
                          className={String(b.id) === bakiF ? 'baki-baris aktif' : 'baki-baris'}
                          onClick={() => pilih(String(b.id))}
                        >
                          <span>{b.name}</span>
                          <span className="muted">
                            {b.isi}/{kapasitas}
                            {b.isi === 0 ? ' kosong' : b.isi >= kapasitas ? ' penuh' : ''}
                          </span>
                        </button>
                        <button
                          type="button"
                          className="mini"
                          title={`Ubah nama ${b.name}`}
                          aria-label={`Ubah nama ${b.name}`}
                          onClick={() => {
                            setBuka(false)
                            setGalat('')
                            setForm({ id: b.id, nama: b.name })
                          }}
                        >
                          <Pencil size={14} aria-hidden="true" />
                        </button>
                        {b.isi === 0 && (
                          <button
                            type="button"
                            className="mini"
                            title={`Hapus ${b.name}`}
                            aria-label={`Hapus ${b.name}`}
                            onClick={() => {
                              setBuka(false)
                              setHapus(b)
                            }}
                          >
                            <X size={14} aria-hidden="true" />
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
        <span className="muted">
          Terisi {jml.terisi} - Kosong {jml.kosong} - Penuh {jml.penuh}
        </span>
        {bolehTambah && (
          <button type="button" className="ghost" style={{ marginLeft: 'auto' }} disabled={loading} onClick={() => {
            setGalat('')
            setForm({ nama: '' })
          }}>
            {loading && !hapus ? <Spinner /> : <Plus size={16} aria-hidden="true" />}Tambah baki
          </button>
        )}
      </div>
      {baki.length === 0 && (
        <p className="muted" style={{ marginBottom: 0 }}>
          Belum ada baki. Tambah baki, lalu masukkan barang ke dalamnya (maksimal {kapasitas} barang per baki).
        </p>
      )}

      {form && (
        <Popup
          judul={form.id ? 'Ubah nama baki' : 'Tambah baki'}
          labelKonfirmasi={form.id ? 'Simpan nama' : 'Tambah baki'}
          loading={loading}
          nonaktif={!form.nama.trim()}
          onKonfirmasi={simpanNama}
          onTutup={() => setForm(null)}
        >
          <label>Nama baki</label>
          <input
            value={form.nama}
            onChange={(e) => setForm((x) => ({ ...x, nama: e.target.value }))}
            onKeyDown={(e) => e.key === 'Enter' && simpanNama()}
            placeholder="Contoh: Nampan CC1"
            maxLength={50}
            autoFocus
          />
          {galat && <div className="error" style={{ marginTop: 10 }}>{galat}</div>}
          <p className="muted" style={{ marginBottom: 0 }}>
            Nama bebas (maksimal 50 karakter) dan tidak boleh sama dengan baki lain di lokasi ini. Satu baki memuat
            maksimal {kapasitas} barang.
          </p>
        </Popup>
      )}

      {hapus && (
        <Popup
          judul={`Hapus ${hapus.name}?`}
          labelKonfirmasi="Ya, hapus baki"
          danger
          loading={loading}
          onKonfirmasi={konfirmasiHapus}
          onTutup={() => setHapus(null)}
        >
          <table className="tabel-spek">
            <tbody>
              <tr><th>Baki</th><td>{hapus.name}</td></tr>
              <tr><th>Lokasi</th><td>{hapus.lokasi}</td></tr>
              <tr><th>Isi</th><td>{hapus.isi} dari {kapasitas} barang (kosong)</td></tr>
            </tbody>
          </table>
          <p className="muted" style={{ marginTop: 10, marginBottom: 0 }}>
            Baki lain tidak terpengaruh. Nama ini bisa dipakai lagi untuk baki baru.
          </p>
        </Popup>
      )}
    </div>
  )
}
