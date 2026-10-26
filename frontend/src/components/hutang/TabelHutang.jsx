import { rupiah } from '../../utils/format'
import { labelJenis } from '../../constants/hutang'
import BarisKosong from '../common/BarisKosong'

export default function TabelHutang({ rows, bisaVoid, onBayar, onBatal }) {
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
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && <BarisKosong kolom="7">Belum ada hutang</BarisKosong>}
          {rows.map((r) => (
            <tr key={r.id}>
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
                {r.status === 'aktif' && (
                  <button type="button" className="mini primary" onClick={() => onBayar(r)}>
                    Bayar
                  </button>
                )}{' '}
                {bisaVoid && r.terbayar === 0 && (
                  <button type="button" className="mini" onClick={() => onBatal(r)}>
                    Batalkan
                  </button>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
