import { rupiah } from '../../utils/format'
import { labelKategori } from '../../constants/kategori'

export default function TabelTransaksi({ rows, isKas, bisaVoid, onBatal }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Tanggal</th>
            <th>Kategori</th>
            <th>{isKas ? 'Nominal' : 'Barang'}</th>
            <th>Catatan</th>
            {bisaVoid && <th></th>}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr><td colSpan="5" className="muted">Belum ada data</td></tr>
          )}
          {rows.map((r) => (
            <tr key={r.id}>
              <td>{r.transaction_date.slice(0, 10)}</td>
              <td>
                <span className={r.direction === 'in' ? 'tag in' : 'tag out'}>
                  {r.direction === 'in' ? 'Masuk' : 'Keluar'}
                </span>{' '}
                {labelKategori(r.category)}
              </td>
              <td>
                {isKas
                  ? rupiah(r.amount)
                  : `${r.item_name} - ${r.weight_gram} g ${r.karat} (${r.qty} pcs)`}
              </td>
              <td>{r.note}</td>
              {bisaVoid && (
                <td>
                  <button type="button" className="mini" onClick={() => onBatal(r.id)}>
                    Batalkan
                  </button>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
