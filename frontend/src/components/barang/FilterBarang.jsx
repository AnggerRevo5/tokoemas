import { gram, rupiah } from '../../utils/format'
import PilihanLokasi from '../common/PilihanLokasi'

// Filter daftar barang (lokasi, status, jenis, cari) dan ringkasan jumlah/berat.
export default function FilterBarang({
  isCabang, locations, locId, onLocId, status, onStatus, jenisF, onJenis, cari, onCari, data,
}) {
  return (
    <div className="card">
      <div className="form-grid">
        {!isCabang && (
          <div>
            <label>Lokasi</label>
            <PilihanLokasi locations={locations} value={locId} onChange={(e) => onLocId(e.target.value)}>
              <option value="0">Semua lokasi</option>
            </PilihanLokasi>
          </div>
        )}
        <div>
          <label>Status</label>
          <select value={status} onChange={(e) => onStatus(e.target.value)}>
            <option value="tersedia">Tersedia</option>
            <option value="dikirim">Dalam perjalanan</option>
            <option value="cuci">Proses cuci</option>
            <option value="lebur">Dilebur</option>
            <option value="hasil">Hasil lebur</option>
            <option value="terjual">Terjual</option>
            <option value="semua">Semua</option>
          </select>
        </div>
        <div>
          <label>Jenis</label>
          <select value={jenisF} onChange={(e) => onJenis(e.target.value)}>
            <option value="semua">Semua</option>
            <option value="perhiasan">Perhiasan</option>
            <option value="logam_mulia">Logam mulia</option>
            <option value="hasil_lebur">Hasil lebur</option>
          </select>
        </div>
        <div>
          <label>Cari nama, keterangan, atau pencuci</label>
          <input value={cari} onChange={(e) => onCari(e.target.value)} placeholder="Cincin, Adip..." />
        </div>
      </div>
      <p className="muted" style={{ marginBottom: 0 }}>
        {data.jumlah} barang, {gram(data.total_gram)} gram
        {data.total_beli > 0 && <>, harga beli {rupiah(data.total_beli)}</>}
        {data.total_jual > 0 && <>, total harga jual {rupiah(data.total_jual)}</>}
      </p>
    </div>
  )
}
