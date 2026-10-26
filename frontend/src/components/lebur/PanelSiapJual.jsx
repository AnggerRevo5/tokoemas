import { useState } from 'react'
import { api } from '../../services/api'
import { rupiah, gram, hariIni, toggleId } from '../../utils/format'

// Kartu "Siap Dijual ke Penadah": daftar cukim/logam mulia di back office dan form penjualannya.
// onMulai: bersihkan pesan; onSukses(message): setelah tersimpan; onError(msg).
export default function PanelSiapJual({ siap, bisaCatat, onMulai, onSukses, onError }) {
  const [loading, setLoading] = useState(false)
  const [pilih, setPilih] = useState({})
  const [penadah, setPenadah] = useState('')
  const [hargaJual, setHargaJual] = useState('')
  const [jualTgl, setJualTgl] = useState(hariIni())
  const [jualNote, setJualNote] = useState('')

  const terpilih = siap.filter((r) => pilih[r.id])
  const nCukim = terpilih.filter((r) => r.jenis === 'hasil_lebur').length
  const nLM = terpilih.filter((r) => r.jenis === 'logam_mulia').length
  const modalPilih = terpilih.reduce((s, r) => s + r.harga_beli, 0)
  const beratPilih = terpilih.reduce((s, r) => s + r.weight_gram, 0)
  const salahPilih =
    nCukim > 0 && nLM > 0 ? 'Cukim dan logam mulia dijual terpisah.' : nCukim > 1 ? 'Cukim dijual satu per satu.' : ''
  const untungPreview = Number(hargaJual || 0) - modalPilih

  async function jual(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    try {
      const res = await api('/penjualan', {
        method: 'POST',
        body: JSON.stringify({
          item_ids: terpilih.map((r) => r.id),
          penadah_name: penadah,
          sale_total: Number(hargaJual),
          sale_date: jualTgl,
          note: jualNote,
        }),
      })
      setPilih({})
      setPenadah('')
      setHargaJual('')
      setJualNote('')
      onSukses(`${res.message} (untung atau rugi ${rupiah(res.untung)})`)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="card" style={{ marginTop: 16 }}>
      <h3>Siap Dijual ke Penadah</h3>
      <p className="muted">
        Berisi cukim dan logam mulia di back office. Satu penjualan berisi satu cukim, atau beberapa
        logam mulia. Uang masuk kas back office.
      </p>

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              {bisaCatat && <th></th>}
              <th>No</th>
              <th>Barang</th>
              <th>Berat</th>
              <th>Modal</th>
              <th>Asal</th>
            </tr>
          </thead>
          <tbody>
            {siap.length === 0 && (
              <tr><td colSpan="6" className="muted">Belum ada barang siap jual</td></tr>
            )}
            {siap.map((r) => (
              <tr key={r.id}>
                {bisaCatat && (
                  <td>
                    <input type="checkbox" checked={!!pilih[r.id]} onChange={() => setPilih((p) => toggleId(p, r.id))} />
                  </td>
                )}
                <td>{r.item_no}</td>
                <td>{r.item_name} {r.karat}</td>
                <td>{r.weight_gram} g</td>
                <td>{r.harga_beli > 0 ? rupiah(r.harga_beli) : '-'}</td>
                <td>{r.batch_id ? `Lebur #${r.batch_id}` : 'Beli dari customer'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {bisaCatat && terpilih.length > 0 && (
        <form className="panel-aksi" onSubmit={jual}>
          <div>
            <b>{terpilih.length}</b> barang dipilih ({gram(beratPilih)} g), modal {rupiah(modalPilih)}
          </div>
          {salahPilih && <div className="error">{salahPilih}</div>}
          <div className="form-grid">
            <div>
              <label>Nama penadah</label>
              <input value={penadah} onChange={(e) => setPenadah(e.target.value)} required />
            </div>
            <div>
              <label>Harga jual (Rp)</label>
              <input type="number" min="1" value={hargaJual} onChange={(e) => setHargaJual(e.target.value)} required />
            </div>
            <div>
              <label>Tanggal</label>
              <input type="date" value={jualTgl} onChange={(e) => setJualTgl(e.target.value)} required />
            </div>
            <div>
              <label>Catatan</label>
              <input value={jualNote} onChange={(e) => setJualNote(e.target.value)} />
            </div>
          </div>
          {hargaJual && (
            <p className={untungPreview < 0 ? 'neg' : 'muted'}>
              {untungPreview < 0 ? 'Rugi' : 'Untung'}: <b>{rupiah(Math.abs(untungPreview))}</b> (harga jual dikurangi modal)
            </p>
          )}
          <button disabled={loading || !!salahPilih}>{loading ? 'Memproses...' : 'Jual ke penadah'}</button>
        </form>
      )}
    </div>
  )
}
