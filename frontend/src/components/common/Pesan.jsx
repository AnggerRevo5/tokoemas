// Kotak pesan di atas halaman: error, info, dan peringatan (banner).
export default function Pesan({ error, info, warn }) {
  return (
    <>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}
      {info && <div className="success" style={{ marginBottom: 16 }}>{info}</div>}
      {warn && <div className="banner" style={{ marginBottom: 16 }}>{warn}</div>}
    </>
  )
}
