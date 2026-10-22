import { gram } from '../../utils/format'
import { STATUS_CUCI } from '../../constants/cuci'

export default function TabelDaftarCuci({ rows, bisaCatat, bisaVoid, onAksi }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Mulai</th>
            <th>Asal</th>
            <th>Barang</th>
            <th>Pencuci</th>
            <th>Progres dan susut</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr><td colSpan="7" className="muted">Belum ada data</td></tr>
          )}
          {rows.map((r) => (
            <tr key={r.id}>
              <td>
                #{r.id}
                <div className="muted">{r.start_date.slice(0, 10)}</div>
              </td>
              <td>{r.cabang}</td>
              <td style={{ whiteSpace: 'normal', minWidth: 180 }}>
                {r.item_name} - {gram(r.weight_gram)} g ({r.qty} pcs)
                {r.note && <div className="muted">{r.note}</div>}
              </td>
              <td>{r.washer_name || <span className="muted">Belum dikirim</span>}</td>
              <td>
                {r.status === 'batal' ? (
                  <>{r.qty - r.cancel_returned} selesai, {r.cancel_returned} dikembalikan</>
                ) : (
                  <>{r.qty - r.sisa} dari {r.qty} selesai</>
                )}
                {r.berat_sesudah > 0 && (
                  <div className="muted">
                    {gram(r.berat_awal_selesai)} g jadi {gram(r.berat_sesudah)} g, susut{' '}
                    {gram(r.berat_awal_selesai - r.berat_sesudah)} g
                  </div>
                )}
              </td>
              <td>
                <span className={STATUS_CUCI[r.status][1]}>{STATUS_CUCI[r.status][0]}</span>
                {r.finish_date && <div className="muted">{r.finish_date.slice(0, 10)}</div>}
                {r.status === 'batal' && r.cancel_reason && <div className="muted">{r.cancel_reason}</div>}
              </td>
              <td>
                {bisaCatat && r.status === 'menunggu' && (
                  <button type="button" className="mini primary" onClick={() => onAksi('kirim', r)}>
                    Kirim ke pencuci
                  </button>
                )}{' '}
                {bisaCatat && (r.status === 'menunggu' || r.status === 'proses') && r.sisa > 0 && (
                  <button type="button" className="mini" onClick={() => onAksi('kembali', r)}>
                    Batalkan, kembalikan ke back office
                  </button>
                )}{' '}
                {bisaVoid && (r.status === 'menunggu' || r.status === 'proses') && r.sisa === r.qty && (
                  <button type="button" className="mini" onClick={() => onAksi('batal', r)}>
                    Salah input (batalkan)
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
