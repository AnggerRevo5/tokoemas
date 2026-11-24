import { useState } from 'react'
import { api } from '../../services/api'
import { rupiah, gram } from '../../utils/format'
import { LABEL_STATUS, KELAS_STATUS, aksiTalangan } from '../../constants/talangan'
import Popup from '../common/Popup'
import BarisKosong from '../common/BarisKosong'

const NAMA_AKSI = {
  'kirim-bo': 'Bawa ke BO',
  'terima-bo': 'Terima di BO',
  'koreksi-harga': 'Koreksi harga',
}

const wkt = (s) => (s ? new Date(s).toLocaleString('id-ID', { dateStyle: 'short', timeStyle: 'short' }) : '')

// Daftar talangan (MASUK atau KELUAR) dengan aksi sesuai role dan status. Setiap aksi meminta konfirmasi lebih dulu.
export default function TabelTalangan({ rows, me, onUbah, onError, onInfo }) {
  const [konfirmasi, setKonfirmasi] = useState(null) // { row, aksi }
  const [harga, setHarga] = useState('')
  const [alasan, setAlasan] = useState('')
  const [riwayat, setRiwayat] = useState(null)
  const [loading, setLoading] = useState(false)

  function buka(row, aksi) {
    setHarga(String(row.harga_beli))
    setAlasan('')
    setKonfirmasi({ row, aksi })
  }

  async function jalankan() {
    const { row, aksi } = konfirmasi
    let body = {}
    if (aksi === 'koreksi-harga') {
      if (!alasan.trim()) return onError('Alasan koreksi wajib diisi')
      body = { harga_baru: Math.round(Number(harga)), alasan }
    }
    setLoading(true)
    try {
      const res = await api(`/talangan/${row.id}/${aksi}`, { method: 'POST', body: JSON.stringify(body) })
      setKonfirmasi(null)
      onInfo(res.message)
      onUbah()
    } catch (err) {
      setKonfirmasi(null)
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function lihatRiwayat(row) {
    try {
      setRiwayat({ judul: `${row.nama_barang} (${row.bp})`, rows: await api(`/talangan/${row.id}/riwayat`) })
    } catch (err) {
      onError(err.message)
    }
  }

  return (
    <>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Tanggal</th>
              <th>BP</th>
              <th>Barang</th>
              <th className="num">Berat</th>
              <th className="num">Harga</th>
              <th>Status</th>
              <th>Toko</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && <BarisKosong kolom={8}>Belum ada data talangan</BarisKosong>}
            {rows.map((r) => (
              <tr key={r.id}>
                <td>{r.tanggal_beli}</td>
                <td><b>{r.bp}</b></td>
                <td>
                  {r.nama_barang} <span className="muted">{r.kadar}</span>
                  {r.no_surat && <div className="muted">Surat {r.no_surat}</div>}
                  {r.no_referensi && <div className="muted">Nota {r.no_referensi}</div>}
                </td>
                <td className="num">{gram(r.berat)} g</td>
                <td className="num">
                  {rupiah(r.harga_beli)}
                  {r.ubah_harga && <div className="muted">dikoreksi owner</div>}
                </td>
                <td>
                  <span className={KELAS_STATUS[r.status]}>{LABEL_STATUS[r.status] || r.status}</span>
                  {r.tukar_id > 0 && <div className="muted">Penukaran #{r.tukar_id}</div>}
                </td>
                <td>{r.toko_input || '-'}</td>
                <td>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    {aksiTalangan(r, me.role).map((a) => (
                      <button key={a} type="button" className="mini" onClick={() => buka(r, a)}>{NAMA_AKSI[a]}</button>
                    ))}
                    <button type="button" className="mini" onClick={() => lihatRiwayat(r)}>Riwayat</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {konfirmasi && (
        <Popup
          judul={`${NAMA_AKSI[konfirmasi.aksi]}: ${konfirmasi.row.nama_barang}`}
          labelKonfirmasi="Ya, lanjutkan"
          onKonfirmasi={jalankan}
          onTutup={() => setKonfirmasi(null)}
          loading={loading}
        >
          <p>
            {konfirmasi.row.nama_barang} {konfirmasi.row.kadar}, {gram(konfirmasi.row.berat)} g, milik <b>{konfirmasi.row.bp}</b>,
            harga {rupiah(konfirmasi.row.harga_beli)}.
          </p>
          {konfirmasi.aksi === 'kirim-bo' && <p>Barang dibawa PJ ke back office dan menunggu konfirmasi owner.</p>}
          {konfirmasi.aksi === 'terima-bo' && <p>Barang sudah diterima di back office. Barang masuk stok BO dengan tanda talangan (bukan stok milik kita).</p>}
          {konfirmasi.aksi === 'koreksi-harga' && (
            <div className="form-grid">
              <div>
                <label>Harga baru (Rp)</label>
                <input type="number" min="1" step="1" value={harga} onChange={(e) => setHarga(e.target.value)} />
              </div>
              <div className="span2">
                <label>Alasan koreksi (wajib)</label>
                <input value={alasan} onChange={(e) => setAlasan(e.target.value)} maxLength={250} />
              </div>
              <p className="muted span2">Kas toko tidak ikut berubah; koreksi kas dilakukan Super Admin bila perlu.</p>
            </div>
          )}
        </Popup>
      )}

      {riwayat && (
        <Popup judul={`Riwayat: ${riwayat.judul}`} onKonfirmasi={() => setRiwayat(null)} onTutup={() => setRiwayat(null)} labelKonfirmasi="Tutup" tanpaBatal>
          <table className="tabel-spek">
            <tbody>
              {riwayat.rows.map((x, i) => (
                <tr key={i}>
                  <td>{wkt(x.waktu)}</td>
                  <td>
                    {x.jenis === 'cuci' ? <span className="tag info">Cuci</span> : x.dari ? `${LABEL_STATUS[x.dari] || x.dari} → ` : ''}{' '}
                    <b>{LABEL_STATUS[x.ke] || x.ke}</b>
                  </td>
                  <td>{x.oleh}</td>
                  <td className="muted">{x.catatan}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Popup>
      )}
    </>
  )
}
