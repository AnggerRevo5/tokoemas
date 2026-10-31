import { rupiah } from '../../utils/format'
import { nominalBiaya, nominalBiayaTeks, persenBiayaTeks } from '../../utils/biaya'

// Pilihan metode pembayaran penjualan. Tunai tetap di kas yang menjual; QRIS/Debit masuk kas back office.
// Potongan QRIS/Debit (opsional) diisi dalam rupiah atau persen, yang satunya terhitung otomatis dari total.
// metode: 'tunai' | 'qris'. biaya: state dari utils/biaya. tujuanTunai: teks tujuan kas untuk tunai.
export default function PilihMetodeBayar({ metode, onMetode, total, biaya, onBiaya, tujuanTunai = 'kas toko' }) {
  const qris = metode === 'qris'
  const potongan = nominalBiaya(total, biaya)
  const melebihi = qris && potongan > Number(total)

  return (
    <div className="panel-aksi">
      <div className="form-grid">
        <div>
          <label>Metode pembayaran</label>
          <select value={metode} onChange={(e) => onMetode(e.target.value)}>
            <option value="tunai">Tunai</option>
            <option value="qris">QRIS/Debit</option>
          </select>
        </div>
        {qris && (
          <>
            <div>
              <label>Potongan QRIS/Debit (Rp)</label>
              <input
                type="number"
                min="0"
                step="1"
                placeholder="0"
                value={nominalBiayaTeks(total, biaya)}
                onChange={(e) => onBiaya({ mode: 'nominal', nilai: e.target.value })}
              />
            </div>
            <div>
              <label>Potongan (%)</label>
              <input
                type="number"
                min="0"
                max="100"
                step="0.01"
                placeholder="0"
                value={persenBiayaTeks(total, biaya)}
                onChange={(e) => onBiaya({ mode: 'persen', nilai: e.target.value })}
              />
            </div>
          </>
        )}
      </div>
      <p className="muted" style={{ marginBottom: 0 }}>
        {qris
          ? `QRIS/Debit masuk ke kas back office${potongan > 0 ? `, diterima ${rupiah(Math.max(Number(total) - potongan, 0))} setelah potongan ${rupiah(potongan)}` : ''}. Potongan boleh dikosongkan.`
          : `Tunai tetap masuk ke ${tujuanTunai}.`}
      </p>
      {melebihi && <div className="error">Potongan tidak boleh melebihi total penjualan.</div>}
    </div>
  )
}
