import { useState } from 'react'
import { api } from '../../services/api'
import { rupiah } from '../../utils/format'
import BarisKosong from '../common/BarisKosong'

// Posisi per cabang BP (belum lunas). Super admin juga mengelola daftar cabang BP (tambah, aktif/nonaktif).
export default function PiutangBP({ rows, me, onUbah, onError, onInfo }) {
  const [nama, setNama] = useState('')
  const [loading, setLoading] = useState(false)

  async function tambah(e) {
    e.preventDefault()
    setLoading(true)
    try {
      await api('/talangan/bp', { method: 'POST', body: JSON.stringify({ nama }) })
      setNama('')
      onInfo('Cabang BP ditambahkan dan langsung bisa dipakai di Talangan')
      onUbah()
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  async function ubahAktif(r) {
    try {
      await api(`/talangan/bp/${r.bp_id}`, { method: 'PUT', body: JSON.stringify({ aktif: !r.aktif }) })
      onInfo(`${r.bp} ${r.aktif ? 'dinonaktifkan' : 'diaktifkan'}`)
      onUbah()
    } catch (err) {
      onError(err.message)
    }
  }

  return (
    <div>
      <h3>Piutang per cabang BP</h3>
      <p className="muted">
        Piutang = BP masih harus membayar ke kita. Utang = kita masih harus membayar ke BP. Dihitung dari penukaran yang uangnya belum masuk kas BO.
      </p>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>BP</th>
              <th className="num">Barang BP di BO (belum ditukar)</th>
              <th className="num">Penukaran belum lunas</th>
              <th className="num">Piutang</th>
              <th className="num">Utang</th>
              {me.role === 'super_admin' && <th></th>}
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 && <BarisKosong kolom={6}>Belum ada data</BarisKosong>}
            {rows.map((r) => (
              <tr key={r.bp_id}>
                <td><b>{r.bp}</b> {!r.aktif && <span className="tag gray">Nonaktif</span>}</td>
                <td className="num">{r.di_bo_jumlah} barang, {rupiah(r.di_bo_nilai)}</td>
                <td className="num">{r.tukar_belum_lunas}</td>
                <td className="num">{rupiah(r.piutang)}</td>
                <td className="num">{rupiah(r.hutang)}</td>
                {me.role === 'super_admin' && (
                  <td><button type="button" className="mini" onClick={() => ubahAktif(r)}>{r.aktif ? 'Nonaktifkan' : 'Aktifkan'}</button></td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {me.role === 'super_admin' && (
        <form className="panel-aksi" onSubmit={tambah}>
          <h4>Tambah cabang BP</h4>
          <div className="form-grid">
            <div>
              <label>Nama cabang BP</label>
              <input value={nama} onChange={(e) => setNama(e.target.value)} maxLength={30} required placeholder="BP Malang" />
            </div>
            <div style={{ alignSelf: 'end' }}>
              <button type="submit" className="primary" disabled={loading}>Tambah</button>
            </div>
          </div>
        </form>
      )}
    </div>
  )
}
