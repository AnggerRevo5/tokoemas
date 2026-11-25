import { Fragment, useState } from 'react'
import { rupiah } from '../../utils/format'
import { labelJenis } from '../../constants/hutang'
import BarisKosong from '../common/BarisKosong'

export default function TabelHutang({ rows }) {
  const [terbuka, setTerbuka] = useState(null)
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Kreditur</th>
            <th className="num">Total</th>
            <th className="num">Terbayar</th>
            <th className="num">Sisa</th>
            <th>Jatuh tempo</th>
            <th>Status</th>
            <th>Riwayat</th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && <BarisKosong kolom="7">Belum ada hutang</BarisKosong>}
          {rows.map((r) => (
            <Fragment key={r.id}>
              <tr>
                <td>
                  {r.creditor_name}
                  <div className="muted">
                    {labelJenis[r.creditor_type] || r.creditor_type}
                    {' - '}
                    {r.dari_kas ? 'kas masuk otomatis' : 'tanpa uang masuk'}
                  </div>
                  {r.note && <div className="muted">{r.note}</div>}
                </td>
                <td className="num">{rupiah(r.principal)}</td>
                <td className="num">{rupiah(r.terbayar)}</td>
                <td className="num"><b>{rupiah(r.sisa)}</b></td>
                <td>{r.due_date ? r.due_date.slice(0, 10) : '-'}</td>
                <td>
                  <span className={r.status === 'lunas' ? 'tag in' : 'tag warn'}>
                    {r.status === 'lunas' ? 'Lunas' : 'Aktif'}
                  </span>
                </td>
                <td>
                  <button type="button" className="mini" onClick={() => setTerbuka(terbuka === r.id ? null : r.id)}>
                    {terbuka === r.id ? 'Tutup' : `Pembayaran (${r.pembayaran?.length || 0})`}
                  </button>
                </td>
              </tr>
              {terbuka === r.id && (
                <tr>
                  <td colSpan="7">
                    {r.pembayaran?.length ? (
                      <div className="table-wrap">
                        <table>
                          <thead><tr><th>Tanggal</th><th className="num">Nominal</th><th>Catatan</th><th>Pencatat</th></tr></thead>
                          <tbody>
                            {r.pembayaran.map((p) => (
                              <tr key={p.id}>
                                <td>{p.paid_date.slice(0, 10)}</td>
                                <td className="num">{rupiah(p.amount)}</td>
                                <td>{p.note || '-'}</td>
                                <td>{p.pencatat || '-'}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    ) : <span className="muted">Belum ada pembayaran.</span>}
                  </td>
                </tr>
              )}
            </Fragment>
          ))}
        </tbody>
      </table>
    </div>
  )
}
