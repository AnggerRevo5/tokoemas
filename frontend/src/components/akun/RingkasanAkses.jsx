import { NAMA_ROLE, DESKRIPSI_ROLE, URUTAN_ROLE, menuUntuk } from '../../constants/roles'

// Kartu per role: jumlah akun dan menu yang bisa diakses.
export default function RingkasanAkses({ ringkasan }) {
  const per = Object.fromEntries(ringkasan.map((r) => [r.role, r]))
  const total = ringkasan.reduce((s, r) => s + r.aktif + r.nonaktif, 0)

  return (
    <div className="card">
      <h3>Ringkasan Akun dan Akses</h3>
      <p className="muted">
        Total {total} akun. Semua akun dengan role yang sama punya fitur yang sama. Cabang baru memakai role
        Akun Cabang, jadi fiturnya sama dengan cabang yang sudah ada.
      </p>
      <div className="grid">
        {URUTAN_ROLE.map((role) => {
          const r = per[role] || { aktif: 0, nonaktif: 0 }
          return (
            <div key={role}>
              <div className="label">{NAMA_ROLE[role]}</div>
              <div className="big">
                {r.aktif} <span className="muted">aktif{r.nonaktif > 0 && `, ${r.nonaktif} nonaktif`}</span>
              </div>
              <div className="muted">{DESKRIPSI_ROLE[role]}</div>
              <div className="muted">
                Menu: {menuUntuk(role).map(([, label]) => label).join(', ')}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
