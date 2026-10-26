import { NAMA_ROLE, menuUntuk } from '../../constants/roles'
import BarisKosong from '../common/BarisKosong'

export default function TabelAkun({ rows, meId, onUbah, onPassword, onToggleAktif }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Username</th>
            <th>Role</th>
            <th>Cabang</th>
            <th>Akses menu</th>
            <th>Status</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && <BarisKosong kolom="6">Belum ada akun</BarisKosong>}
          {rows.map((r) => (
            <tr key={r.id}>
              <td>
                {r.username}
                {r.id === meId && <div className="muted">akun kamu</div>}
              </td>
              <td>{NAMA_ROLE[r.role] || r.role}</td>
              <td>{r.lokasi || '-'}</td>
              <td style={{ whiteSpace: 'normal', minWidth: 200 }}>
                {menuUntuk(r.role).map(([, label]) => label).join(', ')}
              </td>
              <td>
                <span className={r.is_active ? 'tag in' : 'tag gray'}>{r.is_active ? 'Aktif' : 'Nonaktif'}</span>
              </td>
              <td>
                <button type="button" className="mini primary" onClick={() => onUbah(r)}>Ubah</button>{' '}
                <button type="button" className="mini" onClick={() => onPassword(r)}>Ganti password</button>{' '}
                {r.id !== meId && (
                  <button type="button" className="mini" onClick={() => onToggleAktif(r)}>
                    {r.is_active ? 'Nonaktifkan' : 'Aktifkan'}
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
