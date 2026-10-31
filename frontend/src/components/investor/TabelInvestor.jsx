import { rupiah, tgl10 } from '../../utils/format'
import BarisKosong from '../common/BarisKosong'

export default function TabelInvestor({ rows, terpilih, onPilih }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Investor</th>
            <th className="num">Pokok</th>
            <th className="num">Bunga belum dibayar</th>
            <th>Jatuh tempo berikut</th>
            <th className="num">Perkiraan bunga</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && <BarisKosong kolom="6">Belum ada investor</BarisKosong>}
          {rows.map((r) => (
            <tr key={r.id} className={terpilih === r.id ? 'dipilih' : undefined}>
              <td>
                {r.name}
                <div className="muted">
                  Sejak {tgl10(r.start_date)} - bunga {r.rate_percent}% per bulan
                </div>
                {r.note && <div className="muted">{r.note}</div>}
              </td>
              <td className="num"><b>{rupiah(r.pokok)}</b></td>
              <td className="num">
                {r.bunga_belum > 0 ? <span className="tag warn">{rupiah(r.bunga_belum)}</span> : rupiah(0)}
              </td>
              <td>{r.jatuh_tempo_berikut ? tgl10(r.jatuh_tempo_berikut) : '-'}</td>
              <td className="num">{r.jatuh_tempo_berikut ? rupiah(r.perkiraan_bunga_berikut) : '-'}</td>
              <td>
                <button type="button" className="mini primary" onClick={() => onPilih(r.id)}>
                  Kelola
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
