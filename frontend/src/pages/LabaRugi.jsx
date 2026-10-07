import { useState, useEffect, useCallback } from 'react'
import { api } from '../api'
import { labelKategori } from '../kategori'

const rupiah = (n) =>
  new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(n)

const awalBulan = () => {
  const d = new Date()
  return new Date(d.getFullYear(), d.getMonth(), 1).toLocaleDateString('en-CA')
}
const hariIni = () => new Date().toLocaleDateString('en-CA')

function Baris({ data }) {
  const daftar = Object.entries(data || {})
  if (daftar.length === 0) return <div className="muted">Tidak ada data</div>
  return daftar.map(([k, v]) => (
    <div className="row-line" key={k}>
      <span>{labelKategori(k)}</span>
      <span>{rupiah(v)}</span>
    </div>
  ))
}

export default function LabaRugi() {
  const [dari, setDari] = useState(awalBulan())
  const [sampai, setSampai] = useState(hariIni())
  const [data, setData] = useState(null)
  const [error, setError] = useState('')

  const muat = useCallback(() => {
    setError('')
    api(`/laporan/labarugi?from=${dari}&to=${sampai}`)
      .then(setData)
      .catch((e) => setError(e.message))
  }, [dari, sampai])

  useEffect(() => {
    muat()
  }, [muat])

  return (
    <div>
      <div className="card">
        <h3>Laporan Laba Rugi</h3>
        <div className="form-grid">
          <div>
            <label>Dari tanggal</label>
            <input type="date" value={dari} onChange={(e) => setDari(e.target.value)} />
          </div>
          <div>
            <label>Sampai tanggal</label>
            <input type="date" value={sampai} onChange={(e) => setSampai(e.target.value)} />
          </div>
        </div>
      </div>

      {error && <div className="error" style={{ marginTop: 16 }}>{error}</div>}

      {data && (
        <>
          <div className="card" style={{ marginTop: 16 }}>
            <div className="label">Laba / Rugi</div>
            <div className={data.laba_rugi < 0 ? 'big neg' : 'big'}>{rupiah(data.laba_rugi)}</div>
            <div className="muted">
              {data.dari} sampai {data.sampai}
            </div>
          </div>

          <div className="grid" style={{ marginTop: 16 }}>
            <div className="card">
              <h3>Uang Masuk</h3>
              <Baris data={data.uang_masuk.rincian} />
              <div className="row-line total">
                <span>Total</span>
                <span>{rupiah(data.uang_masuk.total)}</span>
              </div>
            </div>

            <div className="card">
              <h3>Uang Keluar</h3>
              <Baris data={data.uang_keluar.rincian} />
              <div className="row-line total">
                <span>Total</span>
                <span>{rupiah(data.uang_keluar.total)}</span>
              </div>
            </div>
          </div>

          <div className="card" style={{ marginTop: 16 }}>
            <h3>Dana Non-Operasional</h3>
            <p className="muted">
              Uang dari investor, pinjaman, dan pembayaran hutang. Ini tidak dihitung sebagai
              untung atau rugi usaha.
            </p>
            <Baris data={data.dana_non_operasional} />
          </div>
        </>
      )}
    </div>
  )
}