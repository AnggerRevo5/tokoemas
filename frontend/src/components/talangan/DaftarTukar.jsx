import { useState } from 'react'
import { api } from '../../services/api'
import { rupiah } from '../../utils/format'
import { LABEL_UANG, KELAS_UANG, LABEL_STATUS, teksSaldo } from '../../constants/talangan'
import Popup from '../common/Popup'
import BarisKosong from '../common/BarisKosong'

const nominalSeharusnya = (t) => (t.koreksi_alasan && t.nominal_bayar > 0 ? t.nominal_bayar : Math.abs(t.saldo))

// Daftar penukaran dan pembayarannya. Cara 1: PJ terima uang lalu setor ke BO. Cara 2: transfer langsung ke/dari BO.
// Owner dapat mengoreksi nominal dengan alasan. Kas BO baru berubah saat uang benar-benar masuk (setor atau transfer).
export default function DaftarTukar({ rows, me, onUbah, onError, onInfo }) {
  const [aksi, setAksi] = useState(null) // { t, jenis: 'terima-pj' | 'setor' | 'transfer' | 'koreksi-nominal' }
  const [nominal, setNominal] = useState('')
  const [noRef, setNoRef] = useState('')
  const [alasan, setAlasan] = useState('')
  const [rincian, setRincian] = useState(null)
  const [loading, setLoading] = useState(false)

  const pj = me.role === 'pj' || me.role === 'super_admin'
  const owner = me.role === 'owner' || me.role === 'super_admin'

  function buka(t, jenis) {
    setNominal(String(nominalSeharusnya(t)))
    setNoRef('')
    setAlasan('')
    setAksi({ t, jenis })
  }

  async function jalankan() {
    const { t, jenis } = aksi
    let body = {}
    if (jenis === 'terima-pj') body = { nominal: Number(nominal) }
    if (jenis === 'transfer') body = { nominal: Number(nominal), no_referensi: noRef }
    if (jenis === 'koreksi-nominal') {
      if (!alasan.trim()) return onError('Alasan koreksi wajib diisi')
      body = { nominal: Number(nominal), alasan }
    }
    setLoading(true)
    try {
      const res = await api(`/talangan/tukar/${t.id}/${jenis}`, { method: 'POST', body: JSON.stringify(body) })
      setAksi(null)
      onInfo(res.message)
      onUbah()
    } catch (err) {
      setAksi(null)
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function lihat(t) {
    try {
      setRincian(await api(`/talangan/tukar/${t.id}`))
    } catch (err) {
      onError(err.message)
    }
  }

  const judulAksi = { 'terima-pj': 'Terima uang di tangan PJ', setor: 'Setor ke back office', transfer: 'Transfer langsung ke BO', 'koreksi-nominal': 'Koreksi nominal' }

  return (
    <>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>#</th>
              <th>Tanggal</th>
              <th>BP</th>
              <th className="num">Masuk</th>
              <th className="num">Keluar</th>
              <th>Saldo</th>
              <th>Uang</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && <BarisKosong kolom={8}>Belum ada penukaran</BarisKosong>}
            {rows.map((t) => (
              <tr key={t.id}>
                <td>{t.id}</td>
                <td>{t.tanggal}</td>
                <td><b>{t.bp}</b> <span className="muted">({t.jumlah_barang} barang)</span></td>
                <td className="num">{rupiah(t.total_masuk)}</td>
                <td className="num">{rupiah(t.total_keluar)}</td>
                <td>{teksSaldo(t.saldo, rupiah)}</td>
                <td>
                  <span className={KELAS_UANG[t.status_uang]}>{LABEL_UANG[t.status_uang]}</span>
                  {t.tanpa_bukti && <div><span className="tag warn" title="Transfer tanpa nomor referensi, perlu dicek owner">Tanpa bukti</span></div>}
                  {t.koreksi_alasan && <div className="muted">Nominal dikoreksi: {t.koreksi_alasan}</div>}
                </td>
                <td>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    {pj && t.status_uang === 'BELUM_BAYAR' && t.saldo !== 0 && (
                      <>
                        <button type="button" className="mini" onClick={() => buka(t, 'terima-pj')}>Terima uang</button>
                        <button type="button" className="mini" onClick={() => buka(t, 'transfer')}>Transfer</button>
                      </>
                    )}
                    {pj && t.status_uang === 'DI_TANGAN_PJ' && <button type="button" className="mini" onClick={() => buka(t, 'setor')}>Setor ke BO</button>}
                    {owner && (t.status_uang === 'BELUM_BAYAR' || t.status_uang === 'DI_TANGAN_PJ') && t.saldo !== 0 && (
                      <button type="button" className="mini" onClick={() => buka(t, 'koreksi-nominal')}>Koreksi nominal</button>
                    )}
                    <button type="button" className="mini" onClick={() => lihat(t)}>Rincian</button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {aksi && (
        <Popup judul={`${judulAksi[aksi.jenis]}: penukaran #${aksi.t.id}`} labelKonfirmasi="Ya, lanjutkan" onKonfirmasi={jalankan} onTutup={() => setAksi(null)} loading={loading}>
          <p>{aksi.t.bp}: <b>{teksSaldo(aksi.t.saldo, rupiah)}</b></p>
          {aksi.jenis === 'setor' ? (
            <p>
              Uang <b>{rupiah(aksi.t.nominal_bayar)}</b> {aksi.t.saldo < 0 ? 'keluar dari' : 'disetor ke'} kas back office. Kas BO berubah sekarang dan
              semua barang penukaran menjadi LUNAS.
            </p>
          ) : (
            <div className="form-grid">
              <div>
                <label>Nominal (Rp)</label>
                <input type="number" min="1" step="1" value={nominal} onChange={(e) => setNominal(e.target.value)} />
              </div>
              {aksi.jenis === 'transfer' && (
                <div>
                  <label>Nomor referensi transfer (disarankan)</label>
                  <input value={noRef} onChange={(e) => setNoRef(e.target.value)} maxLength={60} />
                </div>
              )}
              {aksi.jenis === 'koreksi-nominal' && (
                <div className="span2">
                  <label>Alasan (wajib)</label>
                  <input value={alasan} onChange={(e) => setAlasan(e.target.value)} maxLength={250} />
                </div>
              )}
              <p className="muted span2">
                {aksi.jenis === 'terima-pj' && 'Kas BO belum berubah sampai uang disetor ke BO.'}
                {aksi.jenis === 'transfer' && 'Kas BO langsung berubah dan semua barang penukaran menjadi LUNAS. Nominal harus sama dengan saldo.'}
                {aksi.jenis === 'koreksi-nominal' && 'Nominal pembayaran berikutnya harus sama dengan nominal ini, bukan saldo.'}
              </p>
            </div>
          )}
        </Popup>
      )}

      {rincian && (
        <Popup judul={`Penukaran #${rincian.id} dengan ${rincian.bp}`} labelKonfirmasi="Tutup" onKonfirmasi={() => setRincian(null)} onTutup={() => setRincian(null)} tanpaBatal>
          <table className="tabel-spek">
            <tbody>
              {rincian.items.map((x) => (
                <tr key={x.barang_id}>
                  <td>{x.arah === 'MASUK' ? 'BP' : 'Kita'}</td>
                  <td>{x.nama_barang}</td>
                  <td className="num">{rupiah(x.nilai)}</td>
                  <td>{LABEL_STATUS[x.status] || x.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p>
            {rincian.metode_bayar === 'transfer' && `Transfer${rincian.no_referensi ? `, ref ${rincian.no_referensi}` : ', tanpa nomor referensi'}. `}
            {rincian.metode_bayar === 'via_pj' && 'Lewat PJ. '}
            PJ: {rincian.pj}.
          </p>
        </Popup>
      )}
    </>
  )
}
