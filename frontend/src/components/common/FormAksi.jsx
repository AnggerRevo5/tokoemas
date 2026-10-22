// Form konfirmasi aksi (void-box berbentuk form), dipakai di menu Cuci dan Lebur.
export default function FormAksi({ children, onSubmit, onTutup, loading, danger, labelSubmit }) {
  return (
    <form className="void-box" onSubmit={onSubmit}>
      {children}
      <div className="void-actions">
        <button className={danger ? 'mini danger-fill' : 'mini primary-fill'} disabled={loading}>
          {loading ? 'Memproses...' : labelSubmit}
        </button>
        <button type="button" className="mini" onClick={onTutup}>Tidak jadi</button>
      </div>
    </form>
  )
}
