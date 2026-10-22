import { useState } from 'react'
import { api } from '../../services/api'
import { rupiah } from '../../utils/format'
import KotakBatal from '../common/KotakBatal'

// Kartu "Riwayat Penjualan ke Penadah": periode, total, tabel, dan pembatalan penjualan.
// onMulai: bersihkan pesan; onSelesai(message): setelah dibatalkan; onError(msg).
export default function RiwayatPenjualan({
  penj, dari, onDari, sampai, onSampai, bisaCatat, onMulai, onSelesai, onError,
}) {
  const [voidSale, setVoidSale] = useState(null)
  const [alasanSale, setAlasanSale] = useState('')

  async function batalJual() {
    if (!alasanSale.trim()) {
      onError('Alasan pembatalan wajib diisi')
      return
    }
    onError('')
    try {
      const res = await api(`/penjualan/batal/${voidSale}`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasanSale }),
      })
      setVoidSale(null)
      setAlasanSale('')
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    }
  }

  return (
    <div className="card" style={{ marginTop: 16 }}>
      <h3>Riwayat Penjualan ke Penadah</h3>
      <div className="form-grid">
        <div>
          <label>Dari tanggal</label>
          <input type="date" value={dari} onChange={(e) => onDari(e.target.value)} />
        </div>
        <div>
          <label>Sampai tanggal</label>
          <input type="date" value={sampai} onChange={(e) => onSampai(e.target.value)} />
        </div>
      </div>

      <div className="grid" style={{ margin: '16px 0' }}>
        <div>
          <div className="label">Total modal</div>
          <div className="big">{rupiah(penj.total_modal)}</div>
        </div>
        <div>
          <div className="label">Total harga jual</div>
          <div className="big">{rupiah(penj.total_jual)}</div>
        </div>
        <div>
          <div className="label">{penj.total_untung < 0 ? 'Rugi' : 'Untung'}</div>
          <div className={penj.total_untung < 0 ? 'big neg' : 'big'}>{rupiah(Math.abs(penj.total_untung))}</div>
        </div>
      </div>

      {voidSale && (
        <KotakBatal onKonfirmasi={batalJual} onTutup={() => setVoidSale(null)}>
          <div>Batalkan penjualan #{voidSale}. Barang dan kas dikembalikan. Tulis alasannya:</div>
          <input value={alasanSale} onChange={(e) => setAlasanSale(e.target.value)} autoFocus />
        </KotakBatal>
      )}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Tanggal</th>
              <th>Penadah</th>
              <th>Barang</th>
              <th>Modal</th>
              <th>Harga jual</th>
              <th>Untung / rugi</th>
              <th>PJ</th>
              {bisaCatat && <th></th>}
            </tr>
          </thead>
          <tbody>
            {penj.rows.length === 0 && (
              <tr><td colSpan="8" className="muted">Belum ada penjualan</td></tr>
            )}
            {penj.rows.map((r) => (
              <tr key={r.id}>
                <td>{r.sale_date.slice(0, 10)}</td>
                <td>{r.penadah_name}</td>
                <td style={{ whiteSpace: 'normal', minWidth: 200 }}>
                  {r.barang}
                  {r.note && <div className="muted">{r.note}</div>}
                </td>
                <td>{rupiah(r.modal)}</td>
                <td>{rupiah(r.sale_total)}</td>
                <td className={r.untung < 0 ? 'neg' : ''}>
                  <b>{r.untung < 0 ? 'Rugi ' : 'Untung '}{rupiah(Math.abs(r.untung))}</b>
                </td>
                <td>{r.pj}</td>
                {bisaCatat && (
                  <td>
                    <button
                      type="button"
                      className="mini"
                      onClick={() => {
                        setVoidSale(r.id)
                        setAlasanSale('')
                        onMulai()
                      }}
                    >
                      Batalkan
                    </button>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
