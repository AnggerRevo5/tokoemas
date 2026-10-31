import { m } from 'motion/react'
import Spinner from './Spinner'

// Form konfirmasi aksi (void-box berbentuk form), dipakai di menu Cuci dan Lebur.
export default function FormAksi({ children, onSubmit, onTutup, loading, danger, labelSubmit }) {
  return (
    <m.form
      className="void-box"
      onSubmit={onSubmit}
      initial={{ opacity: 0, y: -10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: 'easeOut' }}
    >
      {children}
      <div className="void-actions">
        <button className={danger ? 'mini danger-fill' : 'mini primary-fill'} disabled={loading}>
          {loading ? <><Spinner size={14} />Memproses...</> : labelSubmit}
        </button>
        <button type="button" className="mini" onClick={onTutup}>Tidak jadi</button>
      </div>
    </m.form>
  )
}
