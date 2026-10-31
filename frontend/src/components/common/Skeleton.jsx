// Kerangka abu-sampanye (terang) / emas redup (gelap) pengganti teks "Memuat...".
export default function Skeleton({ lebar = '100%', tinggi = 14, bulat = false, style }) {
  return (
    <span
      className="skeleton"
      style={{ width: lebar, height: tinggi, borderRadius: bulat ? '50%' : undefined, ...style }}
      aria-hidden="true"
    />
  )
}

// kartu ringkasan yang sedang dimuat
export function KerangkaKartu({ baris = 2 }) {
  return (
    <div className="card stat">
      <div className="stat-kepala">
        <Skeleton lebar="40%" tinggi={11} />
        <Skeleton lebar={32} tinggi={32} bulat />
      </div>
      <Skeleton lebar="65%" tinggi={30} style={{ marginTop: 12 }} />
      {Array.from({ length: baris }, (_, i) => (
        <Skeleton key={i} lebar={i % 2 ? '45%' : '80%'} tinggi={12} style={{ marginTop: 10 }} />
      ))}
    </div>
  )
}

// isi halaman (kartu dan tabel) yang sedang dimuat
export function KerangkaHalaman({ kartu = 3 }) {
  return (
    <div role="status" aria-label="Memuat">
      <div className="ringkasan">
        {Array.from({ length: kartu }, (_, i) => (
          <KerangkaKartu key={i} />
        ))}
      </div>
      <div className="card">
        <Skeleton lebar="30%" tinggi={14} />
        {Array.from({ length: 5 }, (_, i) => (
          <Skeleton key={i} tinggi={36} style={{ marginTop: 12 }} />
        ))}
      </div>
    </div>
  )
}

// kerangka aplikasi saat data akun belum dimuat
export function KerangkaAplikasi() {
  return (
    <div className="app">
      <div className="main">
        <header className="topbar">
          <Skeleton lebar={38} tinggi={38} bulat />
          <Skeleton lebar={160} tinggi={20} />
          <span style={{ flex: 1 }} />
          <Skeleton lebar={34} tinggi={34} bulat />
        </header>
        <main className="content">
          <KerangkaHalaman kartu={4} />
        </main>
      </div>
    </div>
  )
}
