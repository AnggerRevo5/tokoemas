import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { rupiah, hariIni, gram, toggleId } from '../utils/format'
import Pesan from '../components/common/Pesan'
import Popup from '../components/common/Popup'
import KotakBatal from '../components/common/KotakBatal'
import BarisKosong from '../components/common/BarisKosong'
import Spinner from '../components/common/Spinner'
import { KARAT_UMUM, KARAT_LAIN } from '../constants/barang'
import { NAMA_ROLE } from '../constants/roles'

const ARAH = { kita_di_bp: 'Barang kita di BP', bp_di_kita: 'Barang BP di kita' }
const kepalaKosong = () => ({ arah: 'kita_di_bp', entry_date: hariIni(), note: '' })
const barisKosong = () => ({ item_name: '', karat: '', weight_gram: '', nilai: '' })

const NAMA_FIELD = { bp_nama: 'BP', arah: 'Arah', item_name: 'Barang', karat: 'Kadar', weight_gram: 'Berat', nilai: 'Nilai', entry_date: 'Tanggal', note: 'Catatan' }
const tampilField = (k, v) => (k === 'nilai' ? rupiah(v) : k === 'arah' ? ARAH[v] || v : k === 'weight_gram' ? `${gram(v)} g` : v === '' || v == null ? '-' : String(v))

// satu kejadian dari jejak audit menjadi kalimat yang mudah dibaca
function teksKejadian(e) {
  const b = e.baru || {}
  switch (e.aksi) {
    case 'talangan_baru':
      return `Dicatat: ${b.item_name} ${b.karat || ''}, nilai ${rupiah(b.nilai)} (${ARAH[b.arah]}, ${b.bp_nama})`
    case 'talangan_edit': {
      const lama = e.lama || {}
      const sama = (k) => (k === 'nilai' || k === 'weight_gram' ? Number(lama[k]) === Number(b[k]) : String(lama[k] ?? '') === String(b[k] ?? ''))
      const beda = Object.keys(NAMA_FIELD).filter((k) => !sama(k))
      return beda.length
        ? `Diedit: ${beda.map((k) => `${NAMA_FIELD[k]} ${tampilField(k, lama[k])} menjadi ${tampilField(k, b[k])}`).join('; ')}`
        : 'Disimpan ulang tanpa perubahan'
    }
    case 'talangan_konfirmasi':
      return b.konfirmasi ? `Konfirmasi telepon${b.note ? `: ${b.note}` : ''}` : 'Tanda konfirmasi telepon dicabut'
    case 'talangan_batal':
      return `Dibatalkan: ${b.reason}`
    case 'tukar_ajukan':
      return b.tukar_id
        ? `Masuk pengajuan tukar #${b.tukar_id}`
        : `Diajukan: barang kita di BP ${rupiah(b.kita)}, barang BP di kita ${rupiah(b.bp_di_kita)}, selisih ${teksSelisih(b.selisih)}`
    case 'tukar_setujui':
      return b.tukar_id ? `Pengajuan tukar #${b.tukar_id} disetujui owner` : `Disetujui owner, selisih (${teksSelisih(b.selisih)}) dicatat di kas back office`
    case 'tukar_batal':
      return b.tukar_id ? `Pengajuan tukar #${b.tukar_id} dibatalkan (${b.reason}), catatan aktif kembali` : `Pengajuan dibatalkan: ${b.reason}`
    default:
      return e.aksi
  }
}

const teksSelisih = (s) => (s > 0 ? `Kita bayar ${rupiah(s)}` : s < 0 ? `BP bayar ${rupiah(-s)}` : 'Seimbang')

// Menu Talangan BP: catatan barang kita di BP dan barang BP di kita, konfirmasi telepon manual, lalu pengajuan tukar.
// Toko (cabang) menginput dan mengajukan; owner memantau dan menyetujui, baru selisihnya masuk kas back office.
export default function Talangan({ me }) {
  const bisaTulis = me.role === 'cabang' || me.role === 'super_admin'
  const bisaSetuju = me.role === 'owner' || me.role === 'super_admin'

  const [data, setData] = useState({ rows: [], rekap: [], daftar_bp: [] })
  const [tukar, setTukar] = useState([])
  const [tokoList, setTokoList] = useState([])
  const [bp, setBp] = useState('')
  const [tokoId, setTokoId] = useState('')
  const [kepala, setKepala] = useState(kepalaKosong) // isian yang sama untuk semua barang
  const [baris, setBaris] = useState([barisKosong()]) // satu baris = satu barang
  const [editId, setEditId] = useState(null)
  const [pilih, setPilih] = useState({})
  const [tglTukar, setTglTukar] = useState(hariIni())
  const [catatTukar, setCatatTukar] = useState('')
  const [konfNote, setKonfNote] = useState({}) // { id: catatan telepon }
  const [batalKotak, setBatalKotak] = useState(null) // { jenis: 'talangan' | 'tukar', id, teks }
  const [alasan, setAlasan] = useState('')
  const [riwayat, setRiwayat] = useState(null) // { judul, rows }
  const [konfirm, setKonfirm] = useState(null) // { judul, label, isi, aksi }
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')

  const muat = useCallback(() => {
    api('/talangan?status=semua').then(setData).catch((e) => setError(e.message))
    api('/tukar').then(setTukar).catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    muat()
  }, [muat])

  useEffect(() => {
    if (me.role !== 'super_admin') return
    api('/locations')
      .then((l) => {
        const toko = l.filter((x) => x.type === 'cabang')
        setTokoList(toko)
        if (toko.length) setTokoId(String(toko[0].id))
      })
      .catch(() => {})
  }, [me.role])

  const bpAktif = bp || (data.daftar_bp[0] || '')
  const barisBP = data.rows.filter((r) => r.bp_nama === bpAktif)
  const rekapBP = data.rekap.find((r) => r.bp_nama === bpAktif) || { total_kita: 0, total_bp: 0, selisih: 0 }

  // catatan terpilih untuk diajukan tukar (hanya yang aktif dan sudah dikonfirmasi telepon)
  const terpilih = barisBP.filter((r) => pilih[r.id])
  const totalKita = terpilih.filter((r) => r.arah === 'kita_di_bp').reduce((s, r) => s + r.nilai, 0)
  const totalBP = terpilih.filter((r) => r.arah === 'bp_di_kita').reduce((s, r) => s + r.nilai, 0)
  const bisaPilih = (r) => r.status === 'aktif' && r.konfirmasi

  function reset() {
    setError('')
    setInfo('')
  }

  const ubahKepala = (k) => (e) => setKepala((f) => ({ ...f, [k]: e.target.value }))
  const ubahBaris = (i, k, v) => setBaris((b) => b.map((x, idx) => (idx === i ? { ...x, [k]: v } : x)))
  const tambahBaris = () => setBaris((b) => [...b, { ...barisKosong(), karat: b[b.length - 1].karat }])
  const hapusBaris = (i) => setBaris((b) => (b.length === 1 ? b : b.filter((_, idx) => idx !== i)))
  const totalNilaiForm = baris.reduce((s2, x) => s2 + Number(x.nilai || 0), 0)

  function bersihForm() {
    setEditId(null)
    setKepala(kepalaKosong())
    setBaris([barisKosong()])
  }

  function periksaSimpan(e) {
    e.preventDefault()
    reset()
    const items = baris.map((x) => ({
      item_name: x.item_name,
      karat: x.karat,
      weight_gram: Number(x.weight_gram || 0),
      nilai: Number(x.nilai),
    }))
    const umum = { bp_nama: bpAktif, arah: kepala.arah, entry_date: kepala.entry_date, note: kepala.note, location_id: Number(tokoId || 0) }
    setKonfirm({
      judul: editId ? `Simpan perubahan catatan #${editId}?` : `Simpan ${items.length} catatan talangan?`,
      label: 'Ya, simpan',
      isi: (
        <>
          <table className="tabel-spek">
            <tbody>
              <tr><th>BP</th><td>{bpAktif}</td></tr>
              <tr><th>Arah</th><td>{ARAH[kepala.arah]}</td></tr>
              <tr><th>Tanggal</th><td>{kepala.entry_date}</td></tr>
            </tbody>
          </table>
          <div className="table-wrap" style={{ maxHeight: 220, overflow: 'auto', marginTop: 8 }}>
            <table className="tabel-spek">
              <thead>
                <tr><th>Barang</th><th>Kadar</th><th className="num">Berat</th><th className="num">Nilai</th></tr>
              </thead>
              <tbody>
                {items.map((x, i) => (
                  <tr key={i}>
                    <td>{x.item_name}</td>
                    <td>{x.karat || '-'}</td>
                    <td className="num">{x.weight_gram > 0 ? `${gram(x.weight_gram)} g` : '-'}</td>
                    <td className="num">{rupiah(x.nilai)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <p><b>Total: {rupiah(items.reduce((s2, x) => s2 + x.nilai, 0))}</b></p>
        </>
      ),
      aksi: async () => {
        const res = editId
          ? await api(`/talangan/${editId}`, { method: 'PUT', body: JSON.stringify({ ...umum, ...items[0] }) })
          : await api('/talangan', { method: 'POST', body: JSON.stringify({ ...umum, items }) })
        bersihForm()
        return res
      },
    })
  }

  function mulaiEdit(r2) {
    reset()
    setEditId(r2.id)
    setBp(r2.bp_nama)
    setKepala({ arah: r2.arah, entry_date: r2.entry_date, note: r2.note })
    setBaris([{ item_name: r2.item_name, karat: r2.karat, weight_gram: String(r2.weight_gram || ''), nilai: String(r2.nilai) }])
  }

  function bukaRiwayat(jenis, id, judul) {
    reset()
    api(`/${jenis}/${id}/riwayat`)
      .then((rows) => setRiwayat({ judul, rows }))
      .catch((e) => setError(e.message))
  }

  function periksaKonfirmasiTelepon(r, ya) {
    reset()
    setKonfirm({
      judul: ya ? 'Tandai sudah dikonfirmasi lewat telepon?' : 'Cabut tanda konfirmasi telepon?',
      label: 'Ya',
      isi: (
        <p>
          {r.item_name} ({ARAH[r.arah]}, {r.bp_nama}) senilai {rupiah(r.nilai)}.
          {ya && ' Pastikan pegawai sudah menelepon cabang yang memegang barang ini.'}
        </p>
      ),
      aksi: () => api(`/talangan/${r.id}/konfirmasi`, { method: 'POST', body: JSON.stringify({ konfirmasi: ya, note: konfNote[r.id] || '', date: hariIni() }) }),
    })
  }

  function periksaAjukan() {
    reset()
    if (terpilih.length === 0) return setError('Centang catatan yang ikut ditukar')
    setKonfirm({
      judul: `Ajukan tukar dengan ${bpAktif}?`,
      label: 'Ya, ajukan ke owner',
      isi: (
        <>
          <table className="tabel-spek">
            <tbody>
              <tr><th>Barang kita di BP</th><td>{rupiah(totalKita)}</td></tr>
              <tr><th>Barang BP di kita</th><td>{rupiah(totalBP)}</td></tr>
              <tr><th>Selisih</th><td><b>{teksSelisih(totalBP - totalKita)}</b></td></tr>
              <tr><th>Tanggal</th><td>{tglTukar}</td></tr>
            </tbody>
          </table>
          <p className="muted">{terpilih.length} catatan ikut. Kas back office baru tercatat setelah owner menyetujui.</p>
        </>
      ),
      aksi: async () => {
        const res = await api('/tukar', {
          method: 'POST',
          body: JSON.stringify({ bp_nama: bpAktif, talangan_ids: terpilih.map((r) => r.id), tukar_date: tglTukar, note: catatTukar, location_id: Number(tokoId || 0) }),
        })
        setPilih({})
        setCatatTukar('')
        return res
      },
    })
  }

  function periksaSetuju(t) {
    reset()
    setKonfirm({
      judul: `Setujui tukar dengan ${t.bp_nama}?`,
      label: 'Ya, setujui dan catat ke kas',
      isi: (
        <table className="tabel-spek">
          <tbody>
            <tr><th>Toko</th><td>{t.toko}</td></tr>
            <tr><th>Barang kita di BP</th><td>{rupiah(t.total_kita)}</td></tr>
            <tr><th>Barang BP di kita</th><td>{rupiah(t.total_bp)}</td></tr>
            <tr><th>Selisih</th><td><b>{teksSelisih(t.selisih)}</b></td></tr>
            <tr><th>Kas back office</th><td>{t.selisih > 0 ? `keluar ${rupiah(t.selisih)}` : t.selisih < 0 ? `masuk ${rupiah(-t.selisih)}` : 'tidak berubah'}</td></tr>
          </tbody>
        </table>
      ),
      aksi: () => api(`/tukar/${t.id}/setujui`, { method: 'POST', body: JSON.stringify({}) }),
    })
  }

  async function kirimBatal() {
    if (!alasan.trim()) return setError('Alasan pembatalan wajib diisi')
    reset()
    try {
      const url = batalKotak.jenis === 'tukar' ? `/tukar/${batalKotak.id}/batal` : `/talangan/${batalKotak.id}/batal`
      const res = await api(url, { method: 'POST', body: JSON.stringify({ reason: alasan }) })
      setBatalKotak(null)
      setAlasan('')
      setInfo(res.message)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  const bisaBatalTukar = (t) => (t.status === 'diajukan' && (me.role === 'cabang' || bisaSetuju)) || (t.status === 'disetujui' && bisaSetuju)

  return (
    <div>
      <Pesan error={error} info={info} />

      <div className="card">
        <h3>Rekap per BP</h3>
        <p className="muted">
          Selisih = nilai barang BP di kita dikurangi nilai barang kita di BP. Positif berarti kita yang bayar, negatif berarti BP yang bayar.
          Hanya catatan yang belum diajukan tukar yang dihitung.
        </p>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>BP</th>
                <th className="num">Barang kita di BP</th>
                <th className="num">Barang BP di kita</th>
                <th>Selisih</th>
              </tr>
            </thead>
            <tbody>
              {data.rekap.map((r) => (
                <tr key={r.bp_nama} onClick={() => setBp(r.bp_nama)} style={{ cursor: 'pointer' }} className={r.bp_nama === bpAktif ? 'dipilih' : ''}>
                  <td><b>{r.bp_nama}</b></td>
                  <td className="num">{rupiah(r.total_kita)} <span className="muted">({r.n_kita})</span></td>
                  <td className="num">{rupiah(r.total_bp)} <span className="muted">({r.n_bp})</span></td>
                  <td><b>{teksSelisih(r.selisih)}</b></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Catatan talangan: {bpAktif}</h3>
        <div className="form-grid">
          <div>
            <label>BP</label>
            <select value={bpAktif} onChange={(e) => { setBp(e.target.value); setPilih({}) }}>
              {data.daftar_bp.map((b) => <option key={b} value={b}>{b}</option>)}
            </select>
          </div>
        </div>

        {bisaTulis && (
          <form className="panel-aksi" onSubmit={periksaSimpan} style={{ marginTop: 12 }}>
            <h4>{editId ? `Edit catatan #${editId}` : 'Tambah catatan (bisa banyak barang sekaligus)'}</h4>
            <div className="form-grid">
              {me.role === 'super_admin' && (
                <div>
                  <label>Toko</label>
                  <select value={tokoId} onChange={(e) => setTokoId(e.target.value)}>
                    {tokoList.map((t) => <option key={t.id} value={t.id}>{t.name}</option>)}
                  </select>
                </div>
              )}
              <div>
                <label>Arah</label>
                <select value={kepala.arah} onChange={ubahKepala('arah')}>
                  <option value="kita_di_bp">Barang kita yang ada di {bpAktif}</option>
                  <option value="bp_di_kita">Barang {bpAktif} yang ada di kita</option>
                </select>
              </div>
              <div>
                <label>Tanggal</label>
                <input type="date" value={kepala.entry_date} onChange={ubahKepala('entry_date')} required />
              </div>
              <div className="span2">
                <label>Catatan untuk semua barang</label>
                <input value={kepala.note} onChange={ubahKepala('note')} />
              </div>
            </div>

            <div style={{ marginTop: 12 }}>
              <div className="baris-grid baris-head baris-talangan">
                <span>Nama barang</span>
                <span>Kadar</span>
                <span>Berat (gram)</span>
                <span>Nilai (Rp)</span>
                <span></span>
              </div>
              {baris.map((x, i) => (
                <div className="baris-grid baris-talangan" key={i}>
                  <input value={x.item_name} onChange={(e) => ubahBaris(i, 'item_name', e.target.value)} placeholder="Cincin" required aria-label="Nama barang" />
                  <select value={x.karat} onChange={(e) => ubahBaris(i, 'karat', e.target.value)} aria-label="Kadar">
                    <option value="">Kadar (opsional)</option>
                    <optgroup label="Umum">
                      {KARAT_UMUM.map((n) => <option key={n} value={`${n}K`}>{n}K</option>)}
                    </optgroup>
                    <optgroup label="Lainnya">
                      {KARAT_LAIN.map((n) => <option key={n} value={`${n}K`}>{n}K</option>)}
                    </optgroup>
                  </select>
                  <input type="number" step="0.001" min="0" value={x.weight_gram} onChange={(e) => ubahBaris(i, 'weight_gram', e.target.value)} placeholder="Opsional" aria-label="Berat gram" />
                  <input type="number" min="1" value={x.nilai} onChange={(e) => ubahBaris(i, 'nilai', e.target.value)} required aria-label="Nilai rupiah" />
                  {!editId && <button type="button" className="mini" onClick={() => hapusBaris(i)}>Hapus</button>}
                </div>
              ))}
              {!editId && (
                <button type="button" className="ghost" style={{ marginTop: 8 }} onClick={tambahBaris}>Tambah baris</button>
              )}
            </div>

            <div className="total-bar">
              <span>Total: <b>{baris.length}</b> barang</span>
              <span>nilai <b className="utama">{rupiah(totalNilaiForm)}</b></span>
            </div>
            <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
              <button>{editId ? 'Lanjut simpan perubahan' : 'Lanjut'}</button>
              {editId && (
                <button type="button" className="ghost" onClick={bersihForm}>Batal edit</button>
              )}
            </div>
          </form>
        )}

        {batalKotak && (
          <KotakBatal onKonfirmasi={kirimBatal} onTutup={() => { setBatalKotak(null); setAlasan('') }}>
            <div>{batalKotak.teks} Tulis alasannya:</div>
            <input value={alasan} onChange={(e) => setAlasan(e.target.value)} autoFocus />
          </KotakBatal>
        )}

        <div className="table-wrap" style={{ marginTop: 12 }}>
          <table>
            <thead>
              <tr>
                {bisaTulis && <th></th>}
                <th>Tanggal</th>
                <th>Arah</th>
                <th>Barang</th>
                <th className="num">Nilai</th>
                <th>Toko</th>
                <th>Konfirmasi telepon</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {barisBP.length === 0 && <BarisKosong kolom="9">Belum ada catatan untuk {bpAktif}</BarisKosong>}
              {barisBP.map((r) => (
                <tr key={r.id} style={r.status === 'batal' ? { opacity: 0.55 } : undefined}>
                  {bisaTulis && (
                    <td>
                      <input
                        type="checkbox"
                        checked={!!pilih[r.id]}
                        disabled={!bisaPilih(r)}
                        onChange={() => setPilih((p) => toggleId(p, r.id))}
                        aria-label={`Pilih ${r.item_name}`}
                      />
                    </td>
                  )}
                  <td>{r.entry_date}</td>
                  <td>{ARAH[r.arah]}</td>
                  <td style={{ whiteSpace: 'normal', minWidth: 160 }}>
                    {r.item_name} {r.karat} {r.weight_gram > 0 && <span className="muted">{gram(r.weight_gram)} g</span>}
                    {r.note && <div className="muted">{r.note}</div>}
                  </td>
                  <td className="num">{rupiah(r.nilai)}</td>
                  <td>{r.toko}</td>
                  <td style={{ whiteSpace: 'normal', minWidth: 150 }}>
                    {r.konfirmasi ? (
                      <>
                        <span className="tag in">Sudah</span>
                        <div className="muted">{r.konfirmasi_tgl}{r.konfirmasi_note && `, ${r.konfirmasi_note}`}</div>
                      </>
                    ) : (
                      <span className="tag warn">Belum</span>
                    )}
                    {bisaTulis && r.status === 'aktif' && (
                      r.konfirmasi ? (
                        <div><button type="button" className="mini" onClick={() => periksaKonfirmasiTelepon(r, false)}>Cabut</button></div>
                      ) : (
                        <div style={{ display: 'flex', gap: 4, marginTop: 4 }}>
                          <input
                            value={konfNote[r.id] || ''}
                            onChange={(e) => setKonfNote((n) => ({ ...n, [r.id]: e.target.value }))}
                            placeholder="Catatan telepon"
                            aria-label="Catatan telepon"
                          />
                          <button type="button" className="mini primary" onClick={() => periksaKonfirmasiTelepon(r, true)}>Sudah ditelepon</button>
                        </div>
                      )
                    )}
                  </td>
                  <td>
                    <span className={r.status === 'aktif' ? 'tag in' : r.status === 'tukar' ? 'tag info' : 'tag gray'}>
                      {r.status === 'aktif' ? 'Aktif' : r.status === 'tukar' ? `Diajukan tukar #${r.tukar_id}` : 'Dibatalkan'}
                    </span>
                    {r.status === 'batal' && r.batal_alasan && <div className="muted">{r.batal_alasan}</div>}
                  </td>
                  <td>
                    <button type="button" className="mini" onClick={() => bukaRiwayat('talangan', r.id, `Riwayat catatan #${r.id}: ${r.item_name}`)}>
                      Riwayat
                    </button>
                    {bisaTulis && r.status === 'aktif' && (
                        <>
                          <button type="button" className="mini" onClick={() => mulaiEdit(r)}>Edit</button>
                          <button
                            type="button"
                            className="mini"
                            onClick={() => { setBatalKotak({ jenis: 'talangan', id: r.id, teks: `Batalkan catatan ${r.item_name}.` }); setAlasan('') }}
                          >
                            Batalkan
                          </button>
                        </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {bisaTulis && (
          <div className="panel-aksi" style={{ marginTop: 12 }}>
            <h4>Ajukan tukar dengan {bpAktif}</h4>
            <p className="muted">
              Centang catatan yang ikut ditukar (hanya yang sudah dikonfirmasi lewat telepon). Terpilih: {terpilih.length} catatan,
              barang kita {rupiah(totalKita)}, barang BP {rupiah(totalBP)}, selisih <b>{teksSelisih(totalBP - totalKita)}</b>.
            </p>
            <div className="form-grid">
              <div>
                <label>Tanggal tukar</label>
                <input type="date" value={tglTukar} onChange={(e) => setTglTukar(e.target.value)} />
              </div>
              <div>
                <label>Catatan</label>
                <input value={catatTukar} onChange={(e) => setCatatTukar(e.target.value)} />
              </div>
            </div>
            <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
              <button type="button" className="ghost" onClick={() => setPilih(Object.fromEntries(barisBP.filter(bisaPilih).map((r) => [r.id, true])))}>
                Pilih semua
              </button>
              <button type="button" onClick={periksaAjukan} disabled={terpilih.length === 0}>Lanjut ajukan tukar</button>
            </div>
          </div>
        )}
        <p className="muted" style={{ marginTop: 8 }}>
          Total aktif {bpAktif}: barang kita {rupiah(rekapBP.total_kita)}, barang BP {rupiah(rekapBP.total_bp)}.
        </p>
      </div>

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Pengajuan tukar</h3>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>No</th>
                <th>Tanggal</th>
                <th>BP</th>
                <th>Toko</th>
                <th className="num">Kita di BP</th>
                <th className="num">BP di kita</th>
                <th>Selisih</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {tukar.length === 0 && <BarisKosong kolom="9">Belum ada pengajuan tukar</BarisKosong>}
              {tukar.map((t) => (
                <tr key={t.id} style={t.status === 'batal' ? { opacity: 0.55 } : undefined}>
                  <td>#{t.id}</td>
                  <td>{t.tukar_date}</td>
                  <td>{t.bp_nama}</td>
                  <td>{t.toko}</td>
                  <td className="num">{rupiah(t.total_kita)}</td>
                  <td className="num">{rupiah(t.total_bp)}</td>
                  <td><b>{teksSelisih(t.selisih)}</b></td>
                  <td style={{ whiteSpace: 'normal', minWidth: 130 }}>
                    <span className={t.status === 'disetujui' ? 'tag in' : t.status === 'diajukan' ? 'tag warn' : 'tag gray'}>
                      {t.status === 'disetujui' ? 'Disetujui, masuk kas' : t.status === 'diajukan' ? 'Menunggu owner' : 'Dibatalkan'}
                    </span>
                    <div className="muted">{t.jumlah} catatan, diajukan {t.oleh}</div>
                    {t.status === 'batal' && t.batal_alasan && <div className="muted">{t.batal_alasan}</div>}
                  </td>
                  <td>
                    <button type="button" className="mini" onClick={() => bukaRiwayat('tukar', t.id, `Riwayat pengajuan tukar #${t.id}`)}>
                      Riwayat
                    </button>
                    {bisaSetuju && t.status === 'diajukan' && (
                      <button type="button" className="mini primary-fill" onClick={() => periksaSetuju(t)}>Setujui</button>
                    )}
                    {bisaBatalTukar(t) && (
                      <button
                        type="button"
                        className="mini"
                        onClick={() => { setBatalKotak({ jenis: 'tukar', id: t.id, teks: `Batalkan pengajuan tukar #${t.id}. Catatan aktif kembali${t.status === 'disetujui' ? ' dan kas dikembalikan' : ''}.` }); setAlasan('') }}
                      >
                        Batalkan
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {riwayat && (
        <Popup judul={riwayat.judul} labelKonfirmasi="Tutup" tanpaBatal onKonfirmasi={() => setRiwayat(null)} onTutup={() => setRiwayat(null)}>
          {riwayat.rows.length === 0 && <p className="muted">Belum ada riwayat tercatat.</p>}
          <ol style={{ paddingLeft: 18, margin: 0, maxHeight: 340, overflow: 'auto' }}>
            {riwayat.rows.map((e, i) => (
              <li key={i} style={{ marginBottom: 8 }}>
                <div>{teksKejadian(e)}</div>
                <div className="muted">{e.waktu} · {e.user || '-'} ({NAMA_ROLE[e.role] || e.role || '-'})</div>
              </li>
            ))}
          </ol>
        </Popup>
      )}

      {konfirm && (
        <Popup judul={konfirm.judul} labelKonfirmasi={konfirm.label} loading={loading} onKonfirmasi={jalankan} onTutup={() => setKonfirm(null)}>
          {konfirm.isi}
          {loading && <Spinner />}
        </Popup>
      )}
    </div>
  )
}
