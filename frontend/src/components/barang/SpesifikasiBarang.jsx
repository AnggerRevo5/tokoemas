import { rupiah, gram } from '../../utils/format'
import { JENIS_LABEL } from '../../utils/barang'

const beratTeks = (g) => (g > 0 ? `${gram(g)} g` : '-')

// baris spesifikasi satu barang: [label, nilai]
function barisSpesifikasi(r) {
  const lm = r.jenis === 'logam_mulia'
  return [
    ['No', r.item_no],
    ['Jenis', r.bahan === 'perak' ? (lm ? 'Logam mulia perak' : 'Perhiasan perak') : JENIS_LABEL[r.jenis] || r.jenis],
    lm ? ['Merek', r.brand || '-'] : ['Nama', r.item_name],
    ['Kadar', r.karat || '-'],
    ['Berat', beratTeks(r.weight_gram)],
    ['Harga beli', r.harga_beli > 0 ? rupiah(r.harga_beli) : '-'],
    ['Kondisi', r.kondisi === 'kotor' ? 'Kotor' : 'Bersih'],
    ['Baki', r.baki_nama || 'Tanpa baki'],
    ['Lokasi', r.lokasi],
    ['Keterangan', r.note || '-'],
  ]
}

// Tabel spesifikasi barang. Bila `sesudah` diisi, tampil kolom sebelum dan sesudah dan baris yang berubah ditandai.
export default function SpesifikasiBarang({ item, sesudah }) {
  const lama = barisSpesifikasi(item)
  const baru = sesudah ? barisSpesifikasi(sesudah) : null
  return (
    <table className="tabel-spek">
      {baru && (
        <thead>
          <tr><th></th><th>Sebelum</th><th>Sesudah</th></tr>
        </thead>
      )}
      <tbody>
        {lama.map(([label, nilai], i) => {
          const berubah = baru && baru[i][1] !== nilai
          return (
            <tr key={label} className={berubah ? 'berubah' : undefined}>
              <th>{label}</th>
              <td>{nilai}</td>
              {baru && <td>{baru[i][1]}</td>}
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

// Daftar ringkas beberapa barang (dipakai saat pindah baki).
export function DaftarSpesifikasi({ items }) {
  return (
    <div className="table-wrap" style={{ maxHeight: 280, overflow: 'auto' }}>
      <table className="tabel-spek">
        <thead>
          <tr><th>No</th><th>Barang</th><th>Kadar</th><th className="num">Berat</th><th>Baki sekarang</th></tr>
        </thead>
        <tbody>
          {items.map((r) => (
            <tr key={r.id}>
              <td>{r.item_no}</td>
              <td>{r.item_name}</td>
              <td>{r.karat}</td>
              <td className="num">{beratTeks(r.weight_gram)}</td>
              <td>{r.baki_nama || 'Tanpa baki'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
