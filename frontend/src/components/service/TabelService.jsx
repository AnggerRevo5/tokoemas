import { gram, tgl10, rupiah } from '../../utils/format'
import { STATUS_SERVICE, aksiUntuk } from '../../constants/service'

const riwayat = (r) => {
  const out = [`Diterima ${tgl10(r.received_date)} di ${r.origin}`]
  if (r.sent_bo_date) out.push(`Diantar ke back office ${tgl10(r.sent_bo_date)}`)
  if (r.arrived_bo_date) out.push(`Sampai di back office ${tgl10(r.arrived_bo_date)}`)
  if (r.service_name) out.push(`Penyervice: ${r.service_name}, sejak ${tgl10(r.service_start_date)}`)
  if (r.service_end_date) out.push(`Selesai servis ${tgl10(r.service_end_date)}`)
  if (r.returned_bo_date) out.push(`Diterima kembali di back office ${tgl10(r.returned_bo_date)}`)
  if (r.sent_toko_date) out.push(`Dioper ke ${r.pickup} ${tgl10(r.sent_toko_date)}`)
  if (r.pickup_date) out.push(`Diambil customer ${tgl10(r.pickup_date)}`)
  if (r.service_fee !== null && r.service_fee !== undefined) {
    out.push(r.service_fee > 0 ? `Biaya service ${rupiah(r.service_fee)} masuk kas ${r.location}` : 'Tanpa biaya service')
  }
  return out
}

const STATUS_PERJALANAN = ['dikirim_bo', 'kembali_bo']

export default function TabelService({ rows, role, bisaVoid, pilih, onToggle, onBatal }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th></th>
            <th>Nota</th>
            <th>Customer</th>
            <th>Barang</th>
            <th>Pekerjaan</th>
            <th>Status</th>
            <th>Posisi</th>
            <th>Riwayat</th>
            {bisaVoid && <th></th>}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && (
            <tr><td colSpan="9" className="muted">Belum ada barang service</td></tr>
          )}
          {rows.map((r) => (
            <tr key={r.id}>
              <td>
                <input
                  type="checkbox"
                  checked={!!pilih[r.id]}
                  disabled={aksiUntuk(r.status, role).length === 0}
                  onChange={() => onToggle(r.id)}
                />
              </td>
              <td>{r.job_no}</td>
              <td>
                {r.customer_name}
                {r.customer_phone && <div className="muted">{r.customer_phone}</div>}
              </td>
              <td style={{ whiteSpace: 'normal', minWidth: 160 }}>
                {r.item_name} {r.karat} - {gram(r.weight_gram)} g
                {r.weight_after !== null && r.weight_after !== undefined && (
                  <div className="muted">setelah servis {gram(r.weight_after)} g</div>
                )}
                {r.note && <div className="muted">{r.note}</div>}
              </td>
              <td style={{ whiteSpace: 'normal', minWidth: 160 }}>
                {r.complaint}
                {r.service_note && <div className="muted">{r.service_note}</div>}
                {r.service_fee > 0 && <div className="muted">Biaya {rupiah(r.service_fee)}</div>}
              </td>
              <td>
                {STATUS_PERJALANAN.includes(r.status) && r.belum_sampai ? (
                  <span className="tag out">Belum sampai</span>
                ) : (
                  <span className={STATUS_SERVICE[r.status][1]}>{STATUS_SERVICE[r.status][0]}</span>
                )}
                {STATUS_PERJALANAN.includes(r.status) && r.belum_sampai && r.belum_sampai_note && (
                  <div className="muted">{r.belum_sampai_note}</div>
                )}
                {r.service_name && ['diproses'].includes(r.status) && (
                  <div className="muted">oleh {r.service_name}</div>
                )}
              </td>
              <td>
                {r.location}
                {r.pickup && <div className="muted">diambil di {r.pickup}</div>}
              </td>
              <td style={{ whiteSpace: 'normal', minWidth: 220 }}>
                {riwayat(r).map((t, i) => (
                  <div className="muted" key={i}>{t}</div>
                ))}
              </td>
              {bisaVoid && (
                <td>
                  {['di_toko', 'dikirim_bo', 'di_bo'].includes(r.status) && (
                    <button type="button" className="mini" onClick={() => onBatal(r.id)}>
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
