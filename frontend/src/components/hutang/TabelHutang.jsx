import { rupiah } from '../../utils/format'
import { labelJenis } from '../../constants/hutang'

export default function TabelHutang({ rows, bisaVoid, onBayar, onBatal }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Kreditur</th>
            <th>Total</th>
            <th>Terbayar</th>
            <th>Sisa</th>
            <th>Jatuh tempo</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr><td colSpan="7" className="muted">Belum ada hutang</td></tr>
          )}
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
              <td>{rupiah(r.principal)}</td>
              <td>{rupiah(r.terbayar)}</td>
              <td><b>{rupiah(r.sisa)}</b></td>
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
