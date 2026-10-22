import { rupiah, gram } from '../../utils/format'
import { TAHAP_LEBUR } from '../../constants/lebur'

// rantai berat: awal → setelah dipotong → leburan kotor → cukim
const rantai = (b) =>
  [b.source_weight, b.cut_weight, b.leburan_weight, b.cukim_weight]
    .filter((x) => x !== null && x !== undefined)
    .map((x) => gram(x))
    .join(' → ')

export default function TabelLebur({ batches, bisaCatat, bisaUndo, onAksi }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Lebur</th>
            <th>Barang</th>
            <th>Berat (g)</th>
            <th>Modal</th>
            <th>Tahap</th>
            <th>Penjualan</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {batches.length === 0 && (
            <tr><td colSpan="7" className="muted">Belum ada data</td></tr>
          )}
          {batches.map((b) => (
            <tr key={b.id}>
              <td>
                #{b.id}
                <div className="muted">{b.start_date.slice(0, 10)}</div>
              </td>
              <td style={{ whiteSpace: 'normal', minWidth: 220 }}>
                {b.item_count} barang
                <div className="muted">{b.barang}</div>
                {b.note && <div className="muted">{b.note}</div>}
              </td>
              <td>
                {rantai(b)}
                {b.susut !== null && b.susut !== undefined && (
                  <div className="muted">susut {gram(b.susut)} g</div>
                )}
              </td>
              <td>{b.modal > 0 ? rupiah(b.modal) : '-'}</td>
              <td>
                <span className={TAHAP_LEBUR[b.status][1]}>{TAHAP_LEBUR[b.status][0]}</span>
                {b.leburan_by && <div className="muted">leburan: {b.leburan_by}</div>}
                {b.cukim_by && <div className="muted">cukim: {b.cukim_by}</div>}
              </td>
              <td>
                {b.sale_total !== null && b.sale_total !== undefined ? (
                  <>
                    {rupiah(b.sale_total)}
                    <div className="muted">{b.penadah}</div>
                    <div className={b.untung < 0 ? 'neg' : ''}>
                      <b>{b.untung < 0 ? 'Rugi' : 'Untung'} {rupiah(Math.abs(b.untung))}</b>
                    </div>
                  </>
                ) : (
                  '-'
                )}
              </td>
              <td>
                {bisaCatat && b.status === 'proses' && (
                  <button type="button" className="mini primary" onClick={() => onAksi('leburan', b)}>
                    Catat leburan kotor
                  </button>
                )}{' '}
                {bisaCatat && b.status === 'proses' && (
                  <button type="button" className="mini" onClick={() => onAksi('kembali', b)}>
                    Tidak jadi, kembalikan
                  </button>
                )}
                {bisaCatat && b.status === 'leburan_kotor' && (
                  <button type="button" className="mini primary" onClick={() => onAksi('cukim', b)}>
                    Catat cukim
                  </button>
                )}{' '}
                {bisaUndo && b.status !== 'terjual' && (
                  <button type="button" className="mini" onClick={() => onAksi('undo', b)}>
                    Batalkan langkah
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
