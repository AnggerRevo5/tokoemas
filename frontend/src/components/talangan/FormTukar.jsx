import { useState, useEffect, useCallback } from 'react'
import { api } from '../../services/api'
import { rupiah, gram, toggleId } from '../../utils/format'
import { teksSaldo } from '../../constants/talangan'
import Popup from '../common/Popup'
import BarisKosong from '../common/BarisKosong'
import FormKeluar from './FormKeluar'

// Layar Penukaran (PJ): satu cabang BP = satu kunjungan. Pilih barang MASUK (di BO) yang dibawa, pilih atau input barang
// KELUAR yang diambil dari BP, lihat saldo langsung, lalu simpan. Angka resmi dihitung ulang oleh backend.
export default function FormTukar({ bpList, onSelesai, onError, onInfo, onMulai }) {
  const [bpId, setBpId] = useState('')
  const [rows, setRows] = useState([])
  const [masuk, setMasuk] = useState({})
  const [keluar, setKeluar] = useState({})
  const [catatan, setCatatan] = useState('')
  const [bukaKeluar, setBukaKeluar] = useState(false)
  const [konfirmasi, setKonfirmasi] = useState(false)
  const [loading, setLoading] = useState(false)

  const bpAktif = bpId || (bpList[0] ? String(bpList[0].id) : '')
  const namaBP = (bpList.find((b) => String(b.id) === bpAktif) || {}).nama || ''

  const muat = useCallback(
    (pilihBaru) => {
      if (!bpAktif) return
      api(`/talangan?bp_id=${bpAktif}`)
        .then((r) => {
          const siap = r.filter((x) => !x.tukar_id && ((x.arah === 'MASUK' && x.status === 'DI_BO') || (x.arah === 'KELUAR' && x.status === 'TERCATAT')))
          setRows(siap)
          if (pilihBaru) setKeluar((k) => ({ ...k, [pilihBaru]: true }))
        })
        .catch((e) => onError(e.message))
    },
    [bpAktif, onError],
  )

  useEffect(() => {
    setMasuk({})
    setKeluar({})
    muat()
  }, [muat])

  const dataMasuk = rows.filter((r) => r.arah === 'MASUK')
  const dataKeluar = rows.filter((r) => r.arah === 'KELUAR')
  const idMasuk = dataMasuk.filter((r) => masuk[r.id])
  const idKeluar = dataKeluar.filter((r) => keluar[r.id])
  const totalMasuk = idMasuk.reduce((s, r) => s + r.harga_beli, 0)
  const totalKeluar = idKeluar.reduce((s, r) => s + r.harga_beli, 0)
  const saldo = totalMasuk - totalKeluar
  const kosong = idMasuk.length + idKeluar.length === 0

  const semua = (data, pilih, set) => {
    const penuh = data.length > 0 && data.every((r) => pilih[r.id])
    set(penuh ? {} : Object.fromEntries(data.map((r) => [r.id, true])))
  }

  async function simpan() {
    setLoading(true)
    try {
      const res = await api('/talangan/tukar', {
        method: 'POST',
        body: JSON.stringify({
          bp_id: Number(bpAktif),
          masuk_ids: idMasuk.map((r) => r.id),
          keluar_ids: idKeluar.map((r) => r.id),
          catatan,
        }),
      })
      setKonfirmasi(false)
      setCatatan('')
      onInfo(`Penukaran #${res.id} dibuat. ${teksSaldo(res.saldo, rupiah)}.`)
      onSelesai()
      muat()
    } catch (err) {
      setKonfirmasi(false)
      onError(err.message)
      muat()
    } finally {
      setLoading(false)
    }
  }

  const tabel = (judul, data, pilih, set, teksKosong) => (
    <div style={{ marginTop: 14 }}>
      <h4>{judul}</h4>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th style={{ width: 36 }}>
                <input type="checkbox" aria-label="Pilih semua" checked={data.length > 0 && data.every((r) => pilih[r.id])} onChange={() => semua(data, pilih, set)} />
              </th>
              <th>Barang</th>
              <th className="num">Berat</th>
              <th className="num">Nilai</th>
            </tr>
          </thead>
          <tbody>
            {data.length === 0 && <BarisKosong kolom={4}>{teksKosong}</BarisKosong>}
            {data.map((r) => (
              <tr key={r.id}>
                <td>
                  <input type="checkbox" aria-label={`Pilih ${r.nama_barang}`} checked={!!pilih[r.id]} onChange={() => set(toggleId(pilih, r.id))} />
                </td>
                <td>{r.nama_barang} <span className="muted">{r.kadar}</span></td>
                <td className="num">{gram(r.berat)} g</td>
                <td className="num">{rupiah(r.harga_beli)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )

  return (
    <div className="card">
      <p className="muted">
        Pilih barang BP yang dibawa (di back office) dan barang kita yang diambil dari BP. Satu barang hanya bisa masuk satu penukaran.
      </p>
      <div className="form-grid">
        <div>
          <label>Cabang BP</label>
          <select value={bpAktif} onChange={(e) => setBpId(e.target.value)}>
            {bpList.map((b) => <option key={b.id} value={b.id}>{b.nama}</option>)}
          </select>
        </div>
        <div>
          <label>Catatan (opsional)</label>
          <input value={catatan} onChange={(e) => setCatatan(e.target.value)} maxLength={200} />
        </div>
      </div>

      {tabel(`Barang ${namaBP} di back office (MASUK)`, dataMasuk, masuk, setMasuk, 'Tidak ada barang BP ini yang menunggu ditukar')}
      {tabel('Barang kita yang diambil dari BP (KELUAR)', dataKeluar, keluar, setKeluar, 'Belum ada barang kita yang tercatat dari BP ini')}

      <div style={{ marginTop: 8 }}>
        <button type="button" className="ghost" onClick={() => setBukaKeluar(!bukaKeluar)}>
          {bukaKeluar ? 'Tutup form barang keluar' : '+ Input barang kita yang diambil dari BP'}
        </button>
        {bukaKeluar && (
          <FormKeluar
            bpId={bpAktif}
            namaBP={namaBP}
            onMulai={onMulai}
            onError={onError}
            onSukses={(id, pesan) => {
              onInfo(pesan)
              muat(id) // barang baru langsung ikut terpilih
            }}
          />
        )}
      </div>

      <div className="panel-aksi">
        <div className="form-grid">
          <div><label>Total barang BP (masuk)</label><b>{rupiah(totalMasuk)}</b></div>
          <div><label>Total barang kita (keluar)</label><b>{rupiah(totalKeluar)}</b></div>
          <div className="span2">
            <label>Saldo</label>
            <b>{teksSaldo(saldo, rupiah)}</b>
          </div>
        </div>
        <div style={{ marginTop: 10 }}>
          <button type="button" className="primary" disabled={kosong} onClick={() => setKonfirmasi(true)}>Buat penukaran</button>
        </div>
      </div>

      {konfirmasi && (
        <Popup judul={`Buat penukaran dengan ${namaBP}?`} labelKonfirmasi="Ya, buat penukaran" onKonfirmasi={simpan} onTutup={() => setKonfirmasi(false)} loading={loading}>
          <p>{idMasuk.length} barang BP (masuk) {rupiah(totalMasuk)} dan {idKeluar.length} barang kita (keluar) {rupiah(totalKeluar)}.</p>
          <p><b>{teksSaldo(saldo, rupiah)}</b></p>
          <p className="muted">Barang BP keluar dari stok back office. Setelah dibuat, nilai tidak bisa diubah; pembayaran dicatat terpisah.</p>
        </Popup>
      )}
    </div>
  )
}
