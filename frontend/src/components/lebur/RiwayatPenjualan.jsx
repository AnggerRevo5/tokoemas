import { useState } from 'react'
import { api } from '../../services/api'
import { rupiah } from '../../utils/format'
import KotakBatal from '../common/KotakBatal'
import BarisKosong from '../common/BarisKosong'

// Kartu "Riwayat Penjualan ke Penadah": periode, total, tabel, dan pembatalan penjualan.
// onMulai: bersihkan pesan; onSelesai(message): setelah dibatalkan; onError(msg).
export default function RiwayatPenjualan({
  penj, dari, onDari, sampai, onSampai, bisaCatat, bisaKonfirmasi, onMulai, onSelesai, onError,
}) {
  const [voidSale, setVoidSale] = useState(null)
  const [alasanSale, setAlasanSale] = useState('')
  const [edit, setEdit] = useState(null) // catatan yang sedang diedit PJ
  const [konfirmId, setKonfirmId] = useState(null) // catatan yang menunggu konfirmasi owner

  async function simpanEdit(e) {
    e.preventDefault()
    onError('')
    try {
      const res = await api(`/penjualan/${edit.id}`, {
        method: 'PUT',
        body: JSON.stringify({
          penadah_name: edit.penadah_name,
          sale_total: Number(edit.sale_total),
          sale_date: edit.sale_date,
          note: edit.note,
          biaya: Number(edit.biaya || 0),
        }),
      })
      setEdit(null)
      onSelesai(res.message)
    } catch (err) {
      onError(err.message)
    }
  }

  async function konfirmasi() {
    onError('')
    try {
      const res = await api(`/penjualan/${konfirmId.id}/konfirmasi`, { method: 'POST', body: JSON.stringify({}) })
      setKonfirmId(null)
      onSelesai(res.message)
    } catch (err) {
      setKonfirmId(null)
      onError(err.message)
    }
  }

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

      {konfirmId && (
        <KotakBatal onKonfirmasi={konfirmasi} onTutup={() => setKonfirmId(null)} labelKonfirmasi="Ya, konfirmasi" danger={false}>
          <div>
            Konfirmasi penjualan ke <b>{konfirmId.penadah_name}</b> senilai <b>{rupiah(konfirmId.sale_total)}</b>
            {konfirmId.biaya > 0 && ` (potongan ${rupiah(konfirmId.biaya)})`}? Kas back office akan bertambah.
          </div>
        </KotakBatal>
      )}

      {edit && (
        <form className="panel-aksi" onSubmit={simpanEdit} style={{ marginBottom: 16 }}>
          <h4>Edit catatan penjualan #{edit.id}</h4>
          <div className="form-grid">
            <div>
              <label>Nama penadah</label>
              <input value={edit.penadah_name} onChange={(e) => setEdit({ ...edit, penadah_name: e.target.value })} required />
            </div>
            <div>
              <label>Harga laku (Rp)</label>
              <input type="number" min="1" value={edit.sale_total} onChange={(e) => setEdit({ ...edit, sale_total: e.target.value })} required />
            </div>
            <div>
              <label>Tanggal</label>
              <input type="date" value={edit.sale_date} onChange={(e) => setEdit({ ...edit, sale_date: e.target.value })} required />
            </div>
            {edit.metode === 'qris' && (
              <div>
                <label>Potongan QRIS/Debit (Rp)</label>
                <input type="number" min="0" value={edit.biaya} onChange={(e) => setEdit({ ...edit, biaya: e.target.value })} />
              </div>
            )}
            <div className="span2">
              <label>Catatan</label>
              <input value={edit.note} onChange={(e) => setEdit({ ...edit, note: e.target.value })} />
            </div>
          </div>
          <div style={{ display: 'flex', gap: 8, marginTop: 12 }}>
            <button>Simpan perubahan</button>
            <button type="button" className="ghost" onClick={() => setEdit(null)}>Batal</button>
          </div>
        </form>
      )}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Tanggal</th>
              <th>Penadah</th>
              <th>Barang</th>
              <th className="num">Modal</th>
              <th className="num">Harga jual</th>
              <th className="num">Untung / rugi</th>
              <th>PJ</th>
              {(bisaCatat || bisaKonfirmasi) && <th></th>}
            </tr>
          </thead>
          <tbody>
            {penj.rows.length === 0 && <BarisKosong kolom="8">Belum ada penjualan</BarisKosong>}
            {penj.rows.map((r) => (
              <tr key={r.id}>
                <td>{r.sale_date.slice(0, 10)}</td>
                <td>
                  {r.penadah_name}
                  {!r.konfirmasi && <div><span className="tag warn">Menunggu konfirmasi owner</span></div>}
                </td>
                <td style={{ whiteSpace: 'normal', minWidth: 200 }}>
                  {r.barang}
                  {r.note && <div className="muted">{r.note}</div>}
                </td>
                <td className="num">{rupiah(r.modal)}</td>
                <td className="num">
                  {rupiah(r.sale_total)}
                  <div className="muted">
                    {r.metode === 'qris' ? 'QRIS/Debit' : 'Tunai'}
                    {r.biaya > 0 && `, potongan ${rupiah(r.biaya)}`}
                  </div>
                </td>
                <td className={r.untung < 0 ? 'num neg' : 'num'}>
                  <b>{r.untung < 0 ? 'Rugi ' : 'Untung '}{rupiah(Math.abs(r.untung))}</b>
                </td>
                <td>{r.pj}</td>
                {(bisaCatat || bisaKonfirmasi) && (
                  <td>
                    {bisaKonfirmasi && !r.konfirmasi && (
                      <button type="button" className="mini primary-fill" onClick={() => { setKonfirmId(r); onMulai() }}>
                        Konfirmasi
                      </button>
                    )}
                    {bisaCatat && !r.konfirmasi && (
                      <button
                        type="button"
                        className="mini"
                        onClick={() => {
                          setEdit({ id: r.id, penadah_name: r.penadah_name, sale_total: String(r.sale_total), sale_date: r.sale_date.slice(0, 10), note: r.note || '', biaya: String(r.biaya || 0), metode: r.metode })
                          onMulai()
                        }}
                      >
                        Edit
                      </button>
                    )}
                    {bisaCatat && <button
                      type="button"
                      className="mini"
                      onClick={() => {
                        setVoidSale(r.id)
                        setAlasanSale('')
                        onMulai()
                      }}
                    >
                      Batalkan
                    </button>}
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
