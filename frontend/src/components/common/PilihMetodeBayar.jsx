// Pilihan metode pembayaran penjualan: tunai atau transfer. Keduanya masuk ke kas lokasi yang menjual.
// metode: 'tunai' | 'transfer'. tujuan: teks tujuan kas.
export default function PilihMetodeBayar({ metode, onMetode, tujuan = 'kas toko' }) {
  return (
    <div className="panel-aksi">
      <div className="form-grid">
        <div>
          <label>Metode pembayaran</label>
          <select value={metode} onChange={(e) => onMetode(e.target.value)}>
            <option value="tunai">Tunai</option>
            <option value="transfer">Transfer</option>
          </select>
        </div>
      </div>
      <p className="muted" style={{ marginBottom: 0 }}>
        {metode === 'transfer' ? 'Transfer' : 'Tunai'} masuk ke {tujuan}.
      </p>
    </div>
  )
}
