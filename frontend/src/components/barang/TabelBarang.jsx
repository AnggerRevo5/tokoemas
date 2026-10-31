import { AnimatePresence } from 'motion/react'
import { rupiah } from '../../utils/format'
import { STATUS_LABEL, STATUS_CLASS } from '../../constants/barang'
import { bisaPilih } from '../../utils/barang'
import useAnimasiDaftar from '../../hooks/useAnimasiDaftar'
import BarisAnimasi from '../common/BarisAnimasi'
import BarisKosong from '../common/BarisKosong'

export default function TabelBarang({
  rows, pilih, onToggle, semuaDipilih, onToggleSemua, bisaVoid, onBatal, onEdit, onPindah,
}) {
  const animasi = useAnimasiDaftar(rows.map((r) => r.id))
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>
              <input type="checkbox" checked={semuaDipilih} onChange={onToggleSemua} />
            </th>
            <th>No</th>
            <th>Barang</th>
            <th>Baki</th>
            <th className="num">Berat</th>
            <th>Kondisi</th>
            <th className="num">Harga beli</th>
            <th className="num">Harga jual</th>
            <th>Keterangan</th>
            <th>Lokasi</th>
            <th>Asal</th>
            <th>Status</th>
            <th>Masuk</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 && <BarisKosong kolom="14">Belum ada barang</BarisKosong>}
          <AnimatePresence initial={false} custom={animasi.instan}>
          {rows.map((r) => (
            <BarisAnimasi key={r.id} layout={animasi.layout}>
              <td>
                <input
                  type="checkbox"
                  checked={!!pilih[r.id]}
                  disabled={!bisaPilih(r)}
                  onChange={() => onToggle(r.id)}
                />
              </td>
              <td>{r.item_no}</td>
              <td>
                {r.status === 'tersedia' && r.jenis !== 'hasil_lebur' ? (
                  <button type="button" className="link-barang" onClick={() => onEdit(r)} title="Klik untuk edit barang">
                    {r.item_name} {r.karat}
                  </button>
                ) : (
                  <>{r.item_name} {r.karat}</>
                )}
              </td>
              <td>{r.baki_no > 0 ? `Baki ${r.baki_no}` : '-'}</td>
              <td className="num">{r.weight_gram} g</td>
              <td>
                <span className={r.kondisi === 'kotor' ? 'tag warn' : 'tag in'}>
                  {r.kondisi === 'kotor' ? 'Kotor' : 'Bersih'}
                </span>
              </td>
              <td className="num">{r.harga_beli > 0 ? rupiah(r.harga_beli) : '-'}</td>
              <td className="num">
                {r.harga_jual ? rupiah(r.harga_jual) : '-'}
                {r.harga_jual > 0 && r.harga_beli > 0 && (
                  <div className={r.harga_jual < r.harga_beli ? 'neg' : 'muted'}>
                    {r.harga_jual < r.harga_beli ? 'rugi' : 'untung'} {rupiah(Math.abs(r.harga_jual - r.harga_beli))}
                  </div>
                )}
                {r.tanggal_jual && <div className="muted">{r.tanggal_jual.slice(0, 10)}</div>}
              </td>
              <td>{r.note}</td>
              <td>{r.lokasi}</td>
              <td>{r.asal}</td>
              <td>
                {r.status === 'dikirim' && r.belum_sampai ? (
                  <span className="tag out">{r.tujuan_id > 0 ? 'Belum diterima' : 'Belum sampai'}</span>
                ) : (
                  <span className={STATUS_CLASS[r.status]}>{STATUS_LABEL[r.status]}</span>
                )}
                {r.status === 'dikirim' && (
                  <div className="muted">
                    {r.tujuan_id > 0 ? `menuju ${r.tujuan}` : 'menuju Back Office'}
                    {r.kirim_tgl && `, ${r.kirim_tgl.slice(0, 10)}`}
                  </div>
                )}
                {r.status === 'dikirim' && r.belum_sampai && r.belum_sampai_note && (
                  <div className="muted">{r.belum_sampai_note}</div>
                )}
                {r.status === 'cuci' && (
                  <div className="muted">
                    {r.pencuci || 'Menunggu PJ mengirim ke pencuci'}
                    {r.pencuci && r.cuci_tgl && `, sejak ${r.cuci_tgl.slice(0, 10)}`}
                  </div>
                )}
                {r.status === 'hasil' && <div className="muted">dikelola di menu Lebur</div>}
                {r.status === 'kembali' && (
                  <div className="muted">
                    dari PJ, menunggu konfirmasi back office
                    {r.belum_sampai && ' (ditandai belum diterima)'}
                  </div>
                )}
              </td>
              <td>{r.entry_date.slice(0, 10)}</td>
              <td>
                {r.status === 'tersedia' && r.jenis !== 'hasil_lebur' && (
                  <button type="button" className="mini" onClick={() => onEdit(r)}>
                    Edit
                  </button>
                )}
                {r.status === 'tersedia' && r.jenis === 'perhiasan' && (
                  <button type="button" className="mini" style={{ marginLeft: 6 }} onClick={() => onPindah(r)}>
                    Pindah
                  </button>
                )}
                {bisaVoid && (r.status === 'tersedia' || r.status === 'dikirim') && (
                  <button type="button" className="mini" style={{ marginLeft: 6 }} onClick={() => onBatal(r)}>
                    Batalkan
                  </button>
                )}
              </td>
            </BarisAnimasi>
          ))}
          </AnimatePresence>
        </tbody>
      </table>
    </div>
  )
}
