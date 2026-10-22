// Kotak konfirmasi (void-box) dengan tombol aksi dan "Tidak jadi".
// children = isi kotak (teks penjelasan dan kolom input).
export default function KotakBatal({
  children,
  onKonfirmasi,
  onTutup,
  labelKonfirmasi = 'Ya, batalkan',
  danger = true,
}) {
  return (
    <div className="void-box">
      {children}
      <div className="void-actions">
        <button
          type="button"
          className={danger ? 'mini danger-fill' : 'mini primary-fill'}
          onClick={onKonfirmasi}
        >
          {labelKonfirmasi}
        </button>
        <button type="button" className="mini" onClick={onTutup}>Tidak jadi</button>
      </div>
    </div>
  )
}
