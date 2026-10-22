import { rupiah, gram, hariIni } from '../../utils/format'

const tampil = (v, tipe) => {
  if (v === null || v === undefined || v === '') return ''
  if (tipe === 'rupiah') return rupiah(v)
  if (tipe === 'gram') return String(gram(v))
  return String(v)
}

// hasil pratinjau/cetak laporan: judul periode dan satu tabel per jenis laporan
export default function PratinjauLaporan({ hasil, username }) {
  return (
    <div className="cetak-area" style={{ marginTop: 16 }}>
      <div className="card cetak-judul">
        <h2 style={{ margin: 0 }}>Pembukuan Toko Emas - Bulan Purnama</h2>
        <div className="muted">
          Periode {hasil.dari} sampai {hasil.sampai}. Dicetak oleh {username} pada {hariIni()}.
        </div>
      </div>

      {hasil.tabel.map((t) => (
        <div className="card cetak-blok" key={t.kode} style={{ marginTop: 16 }}>
          <h3>{t.judul}</h3>
          <div className="table-wrap">
            <table className="tetap">
              <thead>
                <tr>
                  {t.kolom.map((k, j) => (
                    <th key={j} className={t.tipe[j] === 'teks' ? '' : 'kanan'}>{k}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {(t.baris || []).map((b, i) => (
                  <tr key={i} className={(t.tebal || []).includes(i) ? 'baris-tebal' : ''}>
                    {b.map((v, j) => (
                      <td key={j} className={t.tipe[j] === 'teks' ? '' : 'kanan'}>{tampil(v, t.tipe[j])}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ))}
    </div>
  )
}
