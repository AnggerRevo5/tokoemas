import { useState, useEffect } from 'react'
import { api } from '../../services/api'
import { hariIni } from '../../utils/format'
import { KATEGORI } from '../../constants/kategori'
import { JENIS_KREDITUR } from '../../constants/hutang'
import PilihanLokasi from '../common/PilihanLokasi'
import Spinner from '../common/Spinner'

const KATEGORI_INVESTOR = ['investor', 'tarik_investor', 'bulanan_investor', 'ambil_investor', 'batal_gerak_investor']
const KATEGORI_HUTANG = ['pinjaman', 'bayar_hutang', 'hutang_tanpa_kas', 'batal_hutang']

// Form kas/stok. Transaksi investor dan hutang memakai endpoint khusus agar satu simpanan
// sekaligus membuat catatan kas dan riwayat investor/hutang yang saling terhubung.
export default function FormTransaksi({
  jenis, me, locations, locId, onLocId, onMulai, onSukses, onError,
}) {
  const isKas = jenis === 'kas'

  const [direction, setDirection] = useState('in')
  const [category, setCategory] = useState('')
  const [target, setTarget] = useState('')
  const [tanggal, setTanggal] = useState(hariIni())
  const [amount, setAmount] = useState('')
  const [itemName, setItemName] = useState('')
  const [berat, setBerat] = useState('')
  const [karat, setKarat] = useState('')
  const [qty, setQty] = useState(1)
  const [note, setNote] = useState('')
  const [investorId, setInvestorId] = useState('')
  const [investors, setInvestors] = useState([])
  const [detailInvestor, setDetailInvestor] = useState(null)
  const [gerakId, setGerakId] = useState('')
  const [debtId, setDebtId] = useState('')
  const [debts, setDebts] = useState([])
  const [creditorType, setCreditorType] = useState('perorangan')
  const [creditorName, setCreditorName] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [ratePercent, setRatePercent] = useState('1')
  const [pokokDiambil, setPokokDiambil] = useState('')
  const [bungaDibayar, setBungaDibayar] = useState('')
  const [refreshDaftar, setRefreshDaftar] = useState(0)
  const [loading, setLoading] = useState(false)

  const lokasi = locations.find((l) => l.id === Number(locId))
  const tipe = lokasi ? lokasi.type : 'cabang'
  const isKasBackOffice = isKas && tipe === 'back_office'
  const opsi = (KATEGORI[jenis][tipe][direction] || []).filter(
    ([key]) => key !== 'batal_hutang' || me.role === 'super_admin',
  )
  const cabangList = locations.filter((l) => l.type === 'cabang')
  const perluInvestor = isKasBackOffice && KATEGORI_INVESTOR.includes(category)
  const perluHutang = isKasBackOffice && ['bayar_hutang', 'batal_hutang'].includes(category)
  const investorDipilih = investors.find((r) => r.id === Number(investorId))
  const hutangAktif = debts.filter((r) => r.status === 'aktif' && r.sisa > 0)
  const hutangDipilih = hutangAktif.find((r) => r.id === Number(debtId))
  const hutangBisaBatal = debts.filter((r) => r.terbayar === 0)
  const gerakInvestor = (detailInvestor?.riwayat || []).filter((r) => !r.is_void)
  const kategoriKeuangan = isKasBackOffice && (
    category === 'investor_baru' || KATEGORI_INVESTOR.includes(category) ||
    KATEGORI_HUTANG.includes(category)
  )

  useEffect(() => {
    const c = locations.find((l) => l.type === 'cabang')
    if (c) setTarget(String(c.id))
  }, [locations])

  useEffect(() => {
    if (opsi.length > 0) setCategory(opsi[0][0])
  }, [tipe, direction, jenis])

  useEffect(() => {
    let aktif = true
    async function muatPilihan() {
      try {
        if (perluInvestor) {
          setInvestors([])
          const rows = await api('/investor')
          if (aktif) setInvestors(rows)
        } else if (perluHutang) {
          setDebts([])
          const rows = await api('/hutang')
          if (aktif) setDebts(rows)
        }
      } catch (err) {
        if (aktif) onError(err.message)
      }
    }
    muatPilihan()
    return () => { aktif = false }
  }, [perluInvestor, perluHutang, refreshDaftar, onError])

  useEffect(() => {
    let aktif = true
    if (category !== 'batal_gerak_investor' || !investorId) return () => { aktif = false }
    api(`/investor/${investorId}`)
      .then((data) => { if (aktif) setDetailInvestor(data) })
      .catch((err) => { if (aktif) onError(err.message) })
    return () => { aktif = false }
  }, [category, investorId, refreshDaftar, onError])

  function kosongkanIsian() {
    setAmount('')
    setItemName('')
    setBerat('')
    setKarat('')
    setQty(1)
    setNote('')
    setInvestorId('')
    setDetailInvestor(null)
    setGerakId('')
    setDebtId('')
    setCreditorType('perorangan')
    setCreditorName('')
    setDueDate('')
    setRatePercent('1')
    setPokokDiambil('')
    setBungaDibayar('')
  }

  function ubahLokasi(e) {
    const baru = locations.find((l) => l.id === Number(e.target.value))
    if (baru?.type !== 'back_office' && direction === 'none') setDirection('in')
    kosongkanIsian()
    onLocId(e.target.value)
  }

  async function simpan(e) {
    e.preventDefault()
    if (category === 'ambil_investor' && Number(pokokDiambil) + Number(bungaDibayar) <= 0) {
      onError('Isi nominal pokok dan/atau bunga yang diambil')
      return
    }
    onMulai()
    setLoading(true)
    const lokasiQuery = `?location_id=${Number(locId)}`
    const hutangBody = {
      creditor_type: creditorType,
      creditor_name: creditorName,
      principal: Number(amount),
      start_date: tanggal,
      due_date: dueDate,
      note,
    }
    let res
    try {
      if (isKasBackOffice && category === 'investor_baru') {
        res = await api(`/investor${lokasiQuery}`, {
          method: 'POST',
          body: JSON.stringify({
            name: creditorName,
            amount: Number(amount),
            start_date: tanggal,
            rate_percent: Number(ratePercent),
            note,
            catat_kas: true,
          }),
        })
      } else if (perluInvestor && category === 'investor') {
        res = await api(`/investor/${investorId}/gerak${lokasiQuery}`, {
          method: 'POST',
          body: JSON.stringify({ jenis: 'setor', amount: Number(amount), tx_date: tanggal, note, catat_kas: true }),
        })
      } else if (isKasBackOffice && category === 'pinjaman') {
        res = await api(`/dana/terima${lokasiQuery}`, {
          method: 'POST',
          body: JSON.stringify(hutangBody),
        })
      } else if (perluHutang) {
        if (category === 'batal_hutang') {
          res = await api(`/dana/void/${debtId}${lokasiQuery}`, {
            method: 'POST',
            body: JSON.stringify({ reason: note }),
          })
        } else {
          res = await api(`/hutang/${debtId}/bayar${lokasiQuery}`, {
            method: 'POST',
            body: JSON.stringify({ amount: Number(amount), paid_date: tanggal, note }),
          })
        }
      } else if (perluInvestor && category === 'ambil_investor') {
        res = await api(`/investor/${investorId}/ambil${lokasiQuery}`, {
          method: 'POST',
          body: JSON.stringify({
            pokok: Number(pokokDiambil) || 0,
            bunga: Number(bungaDibayar) || 0,
            tx_date: tanggal,
            note,
            catat_kas: true,
          }),
        })
      } else if (isKasBackOffice && category === 'batal_gerak_investor') {
        res = await api(`/investor-gerak/${gerakId}/void${lokasiQuery}`, {
          method: 'POST',
          body: JSON.stringify({ reason: note }),
        })
      } else if (perluInvestor) {
        const jenisGerak = category === 'tarik_investor' ? 'tarik_pokok' : 'bayar_bunga'
        res = await api(`/investor/${investorId}/gerak${lokasiQuery}`, {
          method: 'POST',
          body: JSON.stringify({ jenis: jenisGerak, amount: Number(amount), tx_date: tanggal, note, catat_kas: true }),
        })
      } else if (isKasBackOffice && category === 'hutang_tanpa_kas') {
        res = await api(`/hutang${lokasiQuery}`, {
          method: 'POST',
          body: JSON.stringify(hutangBody),
        })
      } else {
        const body = isKas
          ? {
              location_id: Number(locId),
              direction,
              category,
              amount: Number(amount),
              note,
              transaction_date: tanggal,
            }
          : {
              location_id: Number(locId),
              target_location_id: category === 'ke_toko' ? Number(target) : 0,
              direction,
              category,
              item_name: itemName,
              weight_gram: Number(berat),
              karat,
              qty: Number(qty),
              note,
              transaction_date: tanggal,
            }
        res = await api(`/${jenis}`, { method: 'POST', body: JSON.stringify(body) })
      }

      kosongkanIsian()
      if (isKasBackOffice) setRefreshDaftar((v) => v + 1)
      onSukses(res)
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form className="card" onSubmit={simpan}>
      <h3>{isKas ? 'Input Kas' : 'Input Stok Emas'}</h3>

      <div className="form-section">
        <div className="section-title">Transaksi</div>
        <div className="form-grid">
          {me.role !== 'cabang' && (
            <div>
              <label>Lokasi</label>
              <PilihanLokasi locations={locations} value={locId} onChange={ubahLokasi} />
            </div>
          )}

          <div>
            <label>{isKas ? 'Arah uang' : 'Arah barang'}</label>
            <select
              value={direction}
              onChange={(e) => {
                kosongkanIsian()
                setDirection(e.target.value)
              }}
            >
              <option value="in">{isKas ? 'Uang masuk' : 'Barang masuk'}</option>
              <option value="out">{isKas ? 'Uang keluar' : 'Barang keluar'}</option>
              {isKasBackOffice && <option value="none">Aksi nonkas</option>}
            </select>
          </div>

          <div>
            <label>Kategori</label>
            <select
              value={category}
              onChange={(e) => {
                kosongkanIsian()
                setCategory(e.target.value)
              }}
            >
              {opsi.map(([k, l]) => (
                <option key={k} value={k}>{l}</option>
              ))}
            </select>
          </div>

          {!isKas && category === 'ke_toko' && (
            <div>
              <label>Kirim ke toko</label>
              <PilihanLokasi locations={cabangList} value={target} onChange={(e) => setTarget(e.target.value)} />
            </div>
          )}

          {category !== 'batal_hutang' && category !== 'batal_gerak_investor' && (
            <div>
              <label>Tanggal</label>
              <input type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} required />
            </div>
          )}
        </div>
      </div>

      <div className="form-section">
        <div className="section-title">{isKas ? 'Rincian transaksi' : 'Barang'}</div>
        <div className="form-grid">
          {isKas && isKasBackOffice && category === 'investor_baru' && (
            <>
              <div>
                <label>Nama investor</label>
                <input value={creditorName} onChange={(e) => setCreditorName(e.target.value)} maxLength={100} required />
              </div>
              <div>
                <label>Setoran awal (Rp)</label>
                <input type="number" min="1" value={amount} onChange={(e) => setAmount(e.target.value)} required />
              </div>
              <div>
                <label>Bunga per bulan (%)</label>
                <input type="number" min="0.01" max="100" step="0.01" value={ratePercent} onChange={(e) => setRatePercent(e.target.value)} required />
              </div>
            </>
          )}

          {isKasBackOffice && perluInvestor && (
            <>
              <div>
                <label>Investor</label>
                <select
                  value={investorId}
                  onChange={(e) => { setInvestorId(e.target.value); setAmount(''); setPokokDiambil(''); setBungaDibayar('') }}
                  required
                >
                  <option value="">Pilih investor</option>
                  {investors.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
                </select>
                {investorDipilih && (
                  <div className="muted">
                    Pokok {investorDipilih.pokok.toLocaleString('id-ID')} · bunga belum dibayar{' '}
                    {investorDipilih.bunga_belum.toLocaleString('id-ID')}
                  </div>
                )}
              </div>

              {category === 'ambil_investor' ? (
                <>
                  <div>
                    <label>Pokok yang ditarik (Rp)</label>
                    <input type="number" min="0" max={investorDipilih?.pokok} value={pokokDiambil} onChange={(e) => setPokokDiambil(e.target.value)} />
                  </div>
                  <div>
                    <label>Bunga yang dibayar (Rp)</label>
                    <input type="number" min="0" max={investorDipilih?.bunga_belum} value={bungaDibayar} onChange={(e) => setBungaDibayar(e.target.value)} />
                  </div>
                </>
              ) : (
                <div>
                  <label>Nominal (Rp)</label>
                  <input
                    type="number"
                    min="1"
                    max={category === 'tarik_investor' ? investorDipilih?.pokok : category === 'bulanan_investor' ? investorDipilih?.bunga_belum : undefined}
                    value={amount}
                    onChange={(e) => setAmount(e.target.value)}
                    required
                  />
                </div>
              )}
            </>
          )}

          {isKasBackOffice && (category === 'pinjaman' || category === 'hutang_tanpa_kas') && (
            <>
              <div>
                <label>Jenis kreditur</label>
                <select value={creditorType} onChange={(e) => setCreditorType(e.target.value)}>
                  {JENIS_KREDITUR.map(([key, label]) => <option key={key} value={key}>{label}</option>)}
                </select>
              </div>
              <div>
                <label>Nama kreditur</label>
                <input value={creditorName} onChange={(e) => setCreditorName(e.target.value)} maxLength={100} required />
              </div>
              <div>
                <label>Nominal (Rp)</label>
                <input type="number" min="1" value={amount} onChange={(e) => setAmount(e.target.value)} required />
              </div>
              <div>
                <label>Jatuh tempo (opsional)</label>
                <input type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />
              </div>
            </>
          )}

          {isKasBackOffice && perluHutang && (
            <>
              <div>
                <label>{category === 'batal_hutang' ? 'Hutang yang dibatalkan' : 'Hutang yang dibayar'}</label>
                <select value={debtId} onChange={(e) => { setDebtId(e.target.value); setAmount('') }} required>
                  <option value="">Pilih hutang</option>
                  {(category === 'batal_hutang' ? hutangBisaBatal : hutangAktif).map((r) => (
                    <option key={r.id} value={r.id}>{r.creditor_name} — sisa Rp {r.sisa.toLocaleString('id-ID')}</option>
                  ))}
                </select>
                {category === 'bayar_hutang' && hutangDipilih && (
                  <div className="muted">Sisa hutang {hutangDipilih.sisa.toLocaleString('id-ID')}</div>
                )}
              </div>
              {category === 'bayar_hutang' && (
                <div>
                  <label>Nominal pembayaran (Rp)</label>
                  <input type="number" min="1" max={hutangDipilih?.sisa} value={amount} onChange={(e) => setAmount(e.target.value)} required />
                </div>
              )}
            </>
          )}

          {isKasBackOffice && category === 'batal_gerak_investor' && (
            <>
              <div>
                <label>Investor</label>
                <select
                  value={investorId}
                  onChange={(e) => { setInvestorId(e.target.value); setGerakId(''); setDetailInvestor(null) }}
                  required
                >
                  <option value="">Pilih investor</option>
                  {investors.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
                </select>
              </div>
              {investorId && (
                <div>
                  <label>Catatan yang dibatalkan</label>
                  <select value={gerakId} onChange={(e) => setGerakId(e.target.value)} required>
                    <option value="">Pilih catatan investor</option>
                    {gerakInvestor.map((r) => (
                      <option key={r.id} value={r.id}>
                        {r.tx_date} — {r.jenis} — Rp {r.amount.toLocaleString('id-ID')}
                      </option>
                    ))}
                  </select>
                  {detailInvestor && gerakInvestor.length === 0 && (
                    <div className="muted">Belum ada catatan aktif yang dapat dibatalkan.</div>
                  )}
                </div>
              )}
            </>
          )}

          {isKas && !kategoriKeuangan && (
            <div>
              <label>Nominal (Rp)</label>
              <input type="number" min="1" value={amount} onChange={(e) => setAmount(e.target.value)} required />
            </div>
          )}

          {!isKas && (
            <>
              <div>
                <label>Nama barang</label>
                <input value={itemName} onChange={(e) => setItemName(e.target.value)} placeholder="Cincin, kalung, gelang..." required />
              </div>
              <div>
                <label>Berat total (gram)</label>
                <input type="number" step="0.001" min="0.001" value={berat} onChange={(e) => setBerat(e.target.value)} required />
              </div>
              <div>
                <label>Kadar</label>
                <input value={karat} onChange={(e) => setKarat(e.target.value)} placeholder="22K" />
              </div>
              <div>
                <label>Jumlah (pcs)</label>
                <input type="number" min="1" value={qty} onChange={(e) => setQty(e.target.value)} required />
              </div>
            </>
          )}

          <div className="span2">
            <label>{category === 'batal_hutang' || category === 'batal_gerak_investor' ? 'Alasan pembatalan' : 'Catatan'}</label>
            <input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              maxLength={isKasBackOffice ? 200 : undefined}
              required={category === 'batal_hutang' || category === 'batal_gerak_investor'}
            />
          </div>
        </div>
      </div>

      {isKas && direction === 'in' && category === 'omzet' && (
        <p className="muted">
          Penjualan barang lewat menu Stok sudah otomatis masuk ke kas. Pakai isian ini hanya untuk
          pemasukan penjualan di luar sistem, supaya tidak tercatat dua kali.
        </p>
      )}
      {isKasBackOffice && category === 'investor_baru' && (
        <p className="muted">Setoran awal otomatis tercatat pada riwayat Kas Back Office dan riwayat Investor.</p>
      )}
      {isKasBackOffice && category === 'pinjaman' && (
        <p className="muted">Dana masuk otomatis tercatat pada riwayat Kas Back Office dan riwayat Hutang.</p>
      )}
      {isKasBackOffice && category === 'hutang_tanpa_kas' && (
        <p className="muted">Hutang dicatat tanpa mengubah saldo kas.</p>
      )}
      {isKasBackOffice && ['investor', 'tarik_investor', 'bulanan_investor', 'ambil_investor', 'bayar_hutang'].includes(category) && (
        <p className="muted">
          Transaksi otomatis tercatat di riwayat Kas Back Office dan riwayat {category === 'bayar_hutang' ? 'Hutang' : 'Investor'}.
        </p>
      )}
      {isKasBackOffice && KATEGORI_INVESTOR.includes(category) && investors.length === 0 && (
        <p className="muted">Belum ada investor yang bisa dipilih. Catat Investor baru terlebih dahulu.</p>
      )}
      {isKasBackOffice && category === 'bayar_hutang' && debts.length === 0 && (
        <p className="muted">Belum ada catatan hutang yang bisa dibayar.</p>
      )}
      {category === 'batal_hutang' && (
        <p className="muted">Hanya hutang yang belum memiliki pembayaran yang dapat dibatalkan.</p>
      )}
      {!isKas && category === 'ke_toko' && (
        <p className="muted">Otomatis tercatat sebagai barang masuk di toko tujuan.</p>
      )}
      {!isKas && category === 'kembali_bo' && (
        <p className="muted">Otomatis tercatat sebagai barang masuk di Back Office.</p>
      )}

      <button disabled={loading}>{loading ? <><Spinner />Menyimpan...</> : 'Simpan'}</button>
    </form>
  )
}
