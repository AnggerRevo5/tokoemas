import { LAPORAN, KELOMPOK_LAPORAN } from '../../constants/laporan'
import PilihanLokasi from '../common/PilihanLokasi'
import { CircleAlert, Download, Eye, Printer } from 'lucide-react'
import Spinner from '../common/Spinner'

// pilihan periode, lokasi, jenis laporan, dan tombol pratinjau/unduh/cetak
export default function FormLaporan({
  dari, onDari, sampai, onSampai, locId, onLocId, locations,
  pilih, onToggle, onSemua, error, loading, onPratinjau, onExcel, onCetak,
}) {
  return (
    <div className="card no-print">
      <h3>Laporan</h3>
      <p className="muted">
        Pilih laporan dan periode, lalu lihat pratinjau, unduh Excel, atau cetak. Untuk PDF, klik Cetak dan
        pilih Simpan sebagai PDF di dialog cetak browser.
      </p>

      <div className="form-grid">
        <div>
          <label>Dari tanggal</label>
          <input type="date" value={dari} onChange={(e) => onDari(e.target.value)} />
        </div>
        <div>
          <label>Sampai tanggal</label>
          <input type="date" value={sampai} onChange={(e) => onSampai(e.target.value)} />
        </div>
        <div>
          <label>Lokasi (untuk Kas dan Riwayat stok)</label>
          <PilihanLokasi locations={locations} value={locId} onChange={(e) => onLocId(e.target.value)}>
            <option value="0">Semua lokasi</option>
          </PilihanLokasi>
        </div>
      </div>

      {KELOMPOK_LAPORAN.map((g) => (
        <div key={g}>
          <label>{g}</label>
          <div className="cek-grup">
            {LAPORAN.filter(([, , gr]) => gr === g).map(([k, nama]) => (
              <label className="cek-item" key={k}>
                <input type="checkbox" checked={!!pilih[k]} onChange={() => onToggle(k)} />
                {nama}
              </label>
            ))}
          </div>
        </div>
      ))}

      <div className="aksi-tombol" style={{ marginTop: 14 }}>
        <button type="button" className="mini" onClick={() => onSemua(true)}>Pilih semua</button>
        <button type="button" className="mini" onClick={() => onSemua(false)}>Kosongkan</button>
      </div>

      {error && (
        <div className="error" role="alert">
          <CircleAlert size={18} aria-hidden="true" />
          <span>{error}</span>
        </div>
      )}

      <div className="aksi-tombol">
        <button type="button" className="mini primary-fill" disabled={loading} onClick={onPratinjau}>
          {loading ? <Spinner size={14} /> : <Eye size={15} aria-hidden="true" />}Pratinjau
        </button>
        <button type="button" className="mini primary" disabled={loading} onClick={onExcel}>
          <Download size={15} aria-hidden="true" />Unduh Excel
        </button>
        <button type="button" className="mini primary" disabled={loading} onClick={onCetak}>
          <Printer size={15} aria-hidden="true" />Cetak / PDF
        </button>
      </div>
    </div>
  )
}
