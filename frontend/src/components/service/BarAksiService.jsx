import { useState } from 'react'
import { api } from '../../services/api'
import { hariIni, gram, rupiah } from '../../utils/format'
import PilihanLokasi from '../common/PilihanLokasi'

// Bar aksi untuk barang service yang dipilih: tombol langkah dan form konfirmasinya.
// aksi/onAksi dipegang halaman. onMulai: bersihkan pesan; onSelesai(message); onError(msg); onBatalPilih().
export default function BarAksiService({
  terpilih, statusPilih, aksiTersedia, aksi, onAksi, cabangList,
  onMulai, onSelesai, onError, onBatalPilih,
}) {
  const [loading, setLoading] = useState(false)
  const [tgl, setTgl] = useState(hariIni())
  const [penyervice, setPenyervice] = useState('')
  const [target, setTarget] = useState('')
  const [beratSetelah, setBeratSetelah] = useState('')
  const [catatan, setCatatan] = useState('')
  const [biaya, setBiaya] = useState({}) // biaya service per barang { id: 'nominal' }

  // tujuan oper: cabang pertama bila belum dipilih
  const targetAktif = target || (cabangList.length > 0 ? String(cabangList[0].id) : '')

  async function jalankan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      if (aksi === 'diambil' && terpilih.some((r) => biaya[r.id] === undefined || biaya[r.id] === '' || Number(biaya[r.id]) < 0)) {
        throw new Error('Biaya service wajib diisi untuk setiap barang (isi 0 kalau tanpa biaya)')
      }
      const res = await api('/service/aksi', {
        method: 'POST',
        body: JSON.stringify({
          action: aksi,
          ids: terpilih.map((r) => r.id),
          date: tgl,
          service_name: penyervice,
          target_location_id: Number(targetAktif),
          weight_after: Number(beratSetelah || 0),
          note: catatan,
          fees: aksi === 'diambil' ? Object.fromEntries(terpilih.map((r) => [r.id, Number(biaya[r.id])])) : undefined,
        }),
      })
      setBiaya({})
      setPenyervice('')
      setBeratSetelah('')
      setCatatan('')
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card action-bar" style={{ marginTop: 16 }}>
      <div><b>{terpilih.length}</b> barang service dipilih</div>
      {!statusPilih && <div className="muted">Pilih barang dengan status yang sama.</div>}
      {statusPilih && aksiTersedia.length === 0 && (
        <div className="muted">Tidak ada langkah yang bisa kamu lakukan untuk status ini.</div>
      )}

      <div className="aksi-tombol">
        {aksiTersedia.map(([k, l]) => (
          <button
            key={k}
            type="button"
            className={aksi === k ? 'mini primary-fill' : 'mini primary'}
            onClick={() => {
              onAksi(k)
              onMulai()
            }}
          >
            {l}
          </button>
        ))}
        <button type="button" className="mini" onClick={onBatalPilih}>
          Batal pilih
        </button>
      </div>

      {aksi && (
        <form className="panel-aksi" onSubmit={jalankan}>
          <div className="form-grid">
            <div>
              <label>Tanggal</label>
              <input type="date" value={tgl} onChange={(e) => setTgl(e.target.value)} required />
            </div>

            {aksi === 'proses' && (
              <div>
                <label>Nama penyervice</label>
                <input value={penyervice} onChange={(e) => setPenyervice(e.target.value)} required />
              </div>
            )}

            {aksi === 'oper' && (
              <div>
                <label>Customer mengambil di</label>
                <PilihanLokasi locations={cabangList} value={targetAktif} onChange={(e) => setTarget(e.target.value)} />
              </div>
            )}

            {aksi === 'selesai' && terpilih.length === 1 && (
              <div>
                <label>Berat setelah servis (g, opsional)</label>
                <input type="number" step="0.001" min="0" value={beratSetelah} onChange={(e) => setBeratSetelah(e.target.value)} />
              </div>
            )}

            {(aksi === 'belum_sampai' || aksi === 'belum_kembali' || aksi === 'selesai') && (
              <div className="span2">
                <label>Catatan (opsional)</label>
                <input value={catatan} onChange={(e) => setCatatan(e.target.value)} />
              </div>
            )}
          </div>

          {aksi === 'diambil' && (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Nota</th>
                    <th>Barang</th>
                    <th>Pekerjaan</th>
                    <th>Biaya service (Rp)</th>
                  </tr>
                </thead>
                <tbody>
                  {terpilih.map((r) => (
                    <tr key={r.id}>
                      <td>{r.job_no}</td>
                      <td>{r.item_name} {r.karat} - {gram(r.weight_gram)} g ({r.customer_name})</td>
                      <td>{r.complaint}</td>
                      <td>
                        <input
                          type="number"
                          min="0"
                          value={biaya[r.id] ?? ''}
                          onChange={(e) => setBiaya((p) => ({ ...p, [r.id]: e.target.value }))}
                          placeholder="0 jika gratis"
                          required
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <p className="muted">
                Total biaya (masuk ke kas toko): <b>{rupiah(terpilih.reduce((s, r) => s + (Number(biaya[r.id]) || 0), 0))}</b>
              </p>
            </div>
          )}

          {aksi === 'proses' && <p className="muted">Barang dikirim ke penyervice dan berstatus sedang diservis.</p>}
          {aksi === 'selesai' && <p className="muted">Barang dikembalikan ke back office dan menunggu konfirmasi owner, lalu owner mengopernya ke cabang.</p>}
          {aksi === 'oper' && <p className="muted">Pegawai membawa barang ke cabang yang dipilih.</p>}
          {aksi === 'diambil' && (
            <p className="muted">
              Isi biaya service tiap barang (0 kalau tanpa biaya). Biaya otomatis masuk ke kas toko tempat
              barang diambil dan tercatat di riwayat.
            </p>
          )}

          <button disabled={loading}>{loading ? 'Memproses...' : 'Konfirmasi'}</button>
        </form>
      )}
    </div>
  )
}
