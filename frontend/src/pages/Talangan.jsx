import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { rupiah, hariIni, gram, toggleId } from '../utils/format'
import Pesan from '../components/common/Pesan'
import Popup from '../components/common/Popup'
import KotakBatal from '../components/common/KotakBatal'
import BarisKosong from '../components/common/BarisKosong'
import Spinner from '../components/common/Spinner'
import { NAMA_ROLE } from '../constants/roles'

const ARAH = { kita_di_bp: 'Barang kita di BP', bp_di_kita: 'Barang BP di kita' }
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

// Talangan BP menampilkan catatan yang dihasilkan dari Stok; pengguna mengajukan tukar dari catatan tersebut.
export default function Talangan({ me }) {
  const bisaTulis = me.role === 'cabang' || me.role === 'owner' || me.role === 'super_admin'
  const bisaSetuju = me.role === 'owner' || me.role === 'super_admin'

  const [data, setData] = useState({ rows: [], rekap: [], daftar_bp: [] })
  const [tukar, setTukar] = useState([])
  const [bp, setBp] = useState('')
  const [pilih, setPilih] = useState({})
  const [tglTukar, setTglTukar] = useState(hariIni())
  const [catatTukar, setCatatTukar] = useState('')
  const [konfNote, setKonfNote] = useState({}) // { id: catatan telepon }
  const [batalKotak, setBatalKotak] = useState(null) // { id, teks }
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
          body: JSON.stringify({ bp_nama: bpAktif, talangan_ids: terpilih.map((r) => r.id), tukar_date: tglTukar, note: catatTukar }),
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
      const res = await api(`/tukar/${batalKotak.id}/batal`, { method: 'POST', body: JSON.stringify({ reason: alasan }) })
      setBatalKotak(null)
      setAlasan('')
      setInfo(res.message)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  async function jalankan() {
    if (!konfirm) return
    setLoading(true)
    reset()
    try {
      const res = await konfirm.aksi()
      setKonfirm(null)
      setInfo(res.message)
      muat()
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
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

        <p className="muted" style={{ marginTop: 12 }}>
          Catatan Talangan dibuat otomatis dari transaksi Stok: oper ke BP atau barang masuk dengan asal BP.
        </p>

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
                        onClick={() => { setBatalKotak({ id: t.id, teks: `Batalkan pengajuan tukar #${t.id}. Catatan aktif kembali${t.status === 'disetujui' ? ' dan stok serta kas dikembalikan' : ''}.` }); setAlasan('') }}
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
