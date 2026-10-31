import { rupiah } from '../../utils/format'
import { labelKategori } from '../../constants/kategori'
import BarisKosong from '../common/BarisKosong'

export default function TabelTransaksi({ rows, isKas, bisaVoid, onBatal, onJual, onGadai }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Tanggal</th>
            <th>Kategori</th>
            <th className={isKas ? 'num' : ''}>{isKas ? 'Nominal' : 'Barang'}</th>
            <th>Catatan</th>
            {(bisaVoid || onJual || onGadai) && <th></th>}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && <BarisKosong kolom="5">Belum ada data</BarisKosong>}
          {rows.map((r) => (
            <tr key={r.id}>
              <td>{r.transaction_date.slice(0, 10)}</td>
              <td>
                <span className={r.direction === 'in' ? 'tag in' : 'tag out'}>
                  {r.direction === 'in' ? 'Masuk' : 'Keluar'}
                </span>{' '}
                {labelKategori(r.category)}
              </td>
              <td className={isKas ? 'num' : ''}>
                {isKas
                  ? rupiah(r.amount)
                  : `${r.item_name} - ${r.weight_gram} g ${r.karat} (${r.qty} pcs)`}
                {r.di_kas_lain && (
                  <div className="muted">tercatat di kas {r.nama_kas}, tidak mengubah kas ini</div>
                )}
              </td>
              <td>{r.note}</td>
              {(bisaVoid || onJual || onGadai) && (
                <td>
                  {isKas && onJual && r.dari_penjualan && (
                    <button type="button" className="mini primary" onClick={() => onJual(r)}>
                      Ubah penjualan
                    </button>
                  )}
                  {isKas && onGadai && r.gadai_id > 0 && (
                    <button type="button" className="mini primary" onClick={() => onGadai(r)}>
                      Lihat gadai
                    </button>
                  )}
                  {bisaVoid && !r.di_kas_lain && !r.dari_penjualan && !(r.gadai_id > 0) && (
                    <button type="button" className="mini" style={{ marginLeft: 6 }} onClick={() => onBatal(r.id)}>
                      Batalkan
                    </button>
                  )}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
