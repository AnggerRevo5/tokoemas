import { useState, useCallback } from 'react'
import { api } from '../../services/api'
import { hariIni } from '../../utils/format'
import { KATEGORI } from '../../constants/kategori'
import { JENIS_KREDITUR } from '../../constants/hutang'
import PilihanLokasi from '../common/PilihanLokasi'
import Spinner from '../common/Spinner'
import DetailInvestor from '../investor/DetailInvestor'

const KATEGORI_INVESTOR = new Set([
  'investor_baru',
  'setor_investor',
  'tarik_pokok_investor',
  'bayar_bunga_investor',
  'ambil_investor',
])
const KATEGORI_HUTANG = new Set(['terima_pinjaman', 'catat_hutang', 'batalkan_hutang', 'bayar_hutang'])

// Form input kas atau stok emas. Lokasi (locId) dipegang halaman karena dipakai juga oleh riwayat.
export default function FormTransaksi({
  jenis, me, locations, locId, onLocId, onMulai, onSukses, onError,
}) {
  const isKas = jenis === 'kas'

  const [direction, setDirection] = useState('in')
  const [kategoriDipilih, setKategoriDipilih] = useState('')
  const [target, setTarget] = useState('')
  const [tanggal, setTanggal] = useState(hariIni())
  const [amount, setAmount] = useState('')
  const [itemName, setItemName] = useState('')
  const [berat, setBerat] = useState('')
  const [karat, setKarat] = useState('')
  const [qty, setQty] = useState(1)
  const [note, setNote] = useState('')
  const [investors, setInvestors] = useState([])
  const [debts, setDebts] = useState([])
  const [investorId, setInvestorId] = useState('')
  const [debtId, setDebtId] = useState('')
  const [investorName, setInvestorName] = useState('')
  const [ratePercent, setRatePercent] = useState('1')
  const [creditorType, setCreditorType] = useState('perorangan')
  const [creditorName, setCreditorName] = useState('')
  const [dueDate, setDueDate] = useState('')
  const [voidReason, setVoidReason] = useState('')
  const [principal, setPrincipal] = useState('')
  const [interest, setInterest] = useState('')
  const [investorDetail, setInvestorDetail] = useState(undefined)
  const [loading, setLoading] = useState(false)

  const lokasi = locations.find((l) => l.id === Number(locId))
  const tipe = lokasi ? lokasi.type : 'cabang'
  const kasBackOffice = isKas && tipe === 'back_office'
  const arahAktif = direction === 'nonkas' && !kasBackOffice ? 'in' : direction
  const opsi = KATEGORI[jenis][tipe][arahAktif]
  const kategoriTampil = opsi.filter(([key]) => key !== 'batalkan_hutang' || me.role === 'super_admin')
  const category = kategoriTampil.some(([key]) => key === kategoriDipilih) ? kategoriDipilih : kategoriTampil[0][0]
  const cabangList = locations.filter((l) => l.type === 'cabang')
  const targetLocation = target || String(cabangList[0]?.id || '')
  const investorDipilih = investors.find((r) => r.id === Number(investorId))
  const debtAktif = debts.filter((r) => r.status === 'aktif' && r.sisa > 0)
  const debtBisaBatal = debts.filter((r) => r.terbayar === 0)

  const muatRelasi = useCallback(async () => {
    if (!kasBackOffice) return
    try {
      const [daftarInvestor, daftarHutang] = await Promise.all([api('/investor'), api('/hutang')])
      setInvestors(daftarInvestor)
      setDebts(daftarHutang)
      setInvestorId((lama) => (daftarInvestor.some((r) => r.id === Number(lama)) ? lama : ''))
      setDebtId((lama) => (daftarHutang.some((r) => r.id === Number(lama) && r.status === 'aktif' && r.sisa > 0) ? lama : ''))
    } catch (err) {
      onError(err.message)
    }
  }, [kasBackOffice, onError])

  async function bukaInvestor() {
    if (!investorDipilih) return
    setInvestorDetail(null)
    try {
      setInvestorDetail(await api(`/investor/${investorDipilih.id}`))
    } catch (err) {
      setInvestorDetail(undefined)
      onError(err.message)
    }
  }

  async function segarkanDetail() {
    await muatRelasi()
    if (investorDipilih) {
      try {
        setInvestorDetail(await api(`/investor/${investorDipilih.id}`))
      } catch (err) {
        setInvestorDetail(undefined)
        onError(err.message)
      }
    }
  }

  async function simpan(e) {
    e.preventDefault()
    onMulai()
    setLoading(true)
    let endpoint = `/${jenis}`
    let body

    if (!isKas) {
      body = {
        location_id: Number(locId),
        target_location_id: category === 'ke_toko' ? Number(targetLocation) : 0,
        direction: arahAktif,
        category,
        item_name: itemName,
        weight_gram: Number(berat),
        karat,
        qty: Number(qty),
        note,
        transaction_date: tanggal,
      }
    } else if (category === 'investor_baru') {
      endpoint = '/investor'
      body = { name: investorName, amount: Number(amount), rate_percent: Number(ratePercent), start_date: tanggal, note }
    } else if (category === 'setor_investor' || category === 'tarik_pokok_investor' || category === 'bayar_bunga_investor') {
      endpoint = `/investor/${investorId}/gerak`
      const gerak = {
        setor_investor: 'setor',
        tarik_pokok_investor: 'tarik_pokok',
        bayar_bunga_investor: 'bayar_bunga',
      }[category]
      body = { jenis: gerak, amount: Number(amount), tx_date: tanggal, note }
    } else if (category === 'ambil_investor') {
      endpoint = `/investor/${investorId}/ambil`
      body = { pokok: Number(principal) || 0, bunga: Number(interest) || 0, tx_date: tanggal, note }
    } else if (category === 'terima_pinjaman' || category === 'catat_hutang') {
      endpoint = category === 'terima_pinjaman' ? '/dana/terima' : '/hutang'
      body = {
        creditor_type: creditorType,
        creditor_name: creditorName,
        principal: Number(amount),
        start_date: tanggal,
        due_date: dueDate,
        note,
      }
    } else if (category === 'batalkan_hutang') {
      endpoint = `/dana/void/${debtId}`
      body = { reason: voidReason }
    } else if (category === 'bayar_hutang') {
      endpoint = `/hutang/${debtId}/bayar`
      body = { amount: Number(amount), paid_date: tanggal, note }
    } else {
      body = {
        location_id: Number(locId),
        direction: arahAktif,
        category,
        amount: Number(amount),
        note,
        transaction_date: tanggal,
      }
    }

    try {
      const res = await api(endpoint, { method: 'POST', body: JSON.stringify(body) })
      setAmount('')
      setItemName('')
      setBerat('')
      setKarat('')
      setQty(1)
      setInvestorName('')
      setCreditorName('')
      setPrincipal('')
      setInterest('')
      setDueDate('')
      setVoidReason('')
      setNote('')
      if (kasBackOffice && (KATEGORI_INVESTOR.has(category) || KATEGORI_HUTANG.has(category))) await muatRelasi()
      const warning = res.peringatan ? ` — ${res.peringatan}` : ''
      onSukses({ ...res, message: `${res.message}${warning}` })
    } catch (err) {
      onError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const formKelola = kasBackOffice && KATEGORI_INVESTOR.has(category)
  const formHutang = kasBackOffice && KATEGORI_HUTANG.has(category)
  const formNonkas = category === 'catat_hutang'

  return (
    <>
      <form className="card" onSubmit={simpan}>
        <h3>{isKas ? 'Input Kas' : 'Input Stok Emas'}</h3>

        <div className="form-section">
          <div className="section-title">Transaksi</div>
          <div className="form-grid">
            {me.role !== 'cabang' && (
              <div>
                <label>Lokasi</label>
                <PilihanLokasi locations={locations} value={locId} onChange={(e) => onLocId(e.target.value)} />
              </div>
            )}

            <div>
              <label>Arah</label>
              <select value={arahAktif} onChange={(e) => setDirection(e.target.value)}>
                <option value="in">{isKas ? 'Uang masuk' : 'Barang masuk'}</option>
                <option value="out">{isKas ? 'Uang keluar' : 'Barang keluar'}</option>
                {kasBackOffice && <option value="nonkas">Aksi tanpa perubahan kas</option>}
              </select>
            </div>

            <div>
              <label>Kategori</label>
              <select
                value={category}
                onChange={(e) => {
                  const value = e.target.value
                  setKategoriDipilih(value)
                  if (kasBackOffice && (KATEGORI_INVESTOR.has(value) || KATEGORI_HUTANG.has(value))) muatRelasi()
                }}
              >
                {kategoriTampil.map(([k, l]) => (
                  <option key={k} value={k}>{l}</option>
                ))}
              </select>
            </div>

            {!isKas && category === 'ke_toko' && (
              <div>
                <label>Kirim ke toko</label>
                <PilihanLokasi locations={cabangList} value={targetLocation} onChange={(e) => setTarget(e.target.value)} />
              </div>
            )}

            <div>
              <label>Tanggal</label>
              <input type="date" value={tanggal} onChange={(e) => setTanggal(e.target.value)} required />
            </div>
          </div>
        </div>

        {formKelola && category === 'investor_baru' && (
          <div className="form-section">
            <div className="section-title">Investor baru</div>
            <div className="form-grid">
              <div><label>Nama investor</label><input value={investorName} onChange={(e) => setInvestorName(e.target.value)} maxLength={100} required /></div>
              <div><label>Setoran awal (Rp)</label><input type="number" min="1" step="1" value={amount} onChange={(e) => setAmount(e.target.value)} required /></div>
              <div><label>Bunga per bulan (%)</label><input type="number" min="0.01" max="100" step="0.01" value={ratePercent} onChange={(e) => setRatePercent(e.target.value)} required /></div>
              <div className="span2"><label>Catatan</label><input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} /></div>
            </div>
          </div>
        )}

        {formKelola && category !== 'investor_baru' && (
          <div className="form-section">
            <div className="section-title">Pengelolaan investor</div>
            <div className="form-grid">
              <div>
                <label>Investor</label>
                <select value={investorId} onChange={(e) => setInvestorId(e.target.value)} required>
                  <option value="">Pilih investor</option>
                  {investors.map((r) => <option key={r.id} value={r.id}>{r.name} — pokok Rp {r.pokok.toLocaleString('id-ID')}</option>)}
                </select>
              </div>
              {category === 'ambil_investor' ? (
                <>
                  <div><label>Pokok yang diambil (Rp)</label><input type="number" min="0" step="1" value={principal} onChange={(e) => setPrincipal(e.target.value)} placeholder="0" /></div>
                  <div><label>Bunga yang dibayar (Rp)</label><input type="number" min="0" step="1" value={interest} onChange={(e) => setInterest(e.target.value)} placeholder="0" /></div>
                </>
              ) : (
                <div>
                  <label>Nominal (Rp)</label>
                  <input type="number" min="1" step="1" value={amount} onChange={(e) => setAmount(e.target.value)} required />
                  {category === 'bayar_bunga_investor' && investorDipilih && <div className="muted">Bunga belum dibayar Rp {investorDipilih.bunga_belum.toLocaleString('id-ID')}</div>}
                </div>
              )}
              <div>
                <label>Catatan</label>
                <input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
              </div>
            </div>
            {investorDipilih && (
              <button type="button" className="ghost" style={{ marginTop: 10 }} onClick={bukaInvestor}>
                Kelola investor: {investorDipilih.name}
              </button>
            )}
          </div>
        )}

        {formHutang && (category === 'terima_pinjaman' || category === 'catat_hutang') && (
          <div className="form-section">
            <div className="section-title">{category === 'terima_pinjaman' ? 'Terima pinjaman' : 'Catat hutang tanpa uang masuk'}</div>
            <div className="form-grid">
              <div>
                <label>Jenis kreditur</label>
                <select value={creditorType} onChange={(e) => setCreditorType(e.target.value)}>
                  {JENIS_KREDITUR.map(([key, label]) => <option key={key} value={key}>{label}</option>)}
                </select>
              </div>
              <div><label>Nama kreditur</label><input value={creditorName} onChange={(e) => setCreditorName(e.target.value)} maxLength={100} required /></div>
              <div><label>Nominal (Rp)</label><input type="number" min="1" step="1" value={amount} onChange={(e) => setAmount(e.target.value)} required /></div>
              <div><label>Jatuh tempo (opsional)</label><input type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} /></div>
              <div className="span2"><label>Catatan</label><input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} /></div>
            </div>
            {formNonkas && <p className="muted">Hutang dicatat tanpa membuat transaksi atau mengubah saldo kas.</p>}
          </div>
        )}

        {formHutang && category === 'batalkan_hutang' && (
          <div className="form-section">
            <div className="section-title">Batalkan catatan hutang</div>
            <div className="form-grid">
              <div>
                <label>Hutang tanpa pembayaran</label>
                <select value={debtId} onChange={(e) => setDebtId(e.target.value)} required>
                  <option value="">Pilih catatan hutang</option>
                  {debtBisaBatal.map((r) => <option key={r.id} value={r.id}>{r.creditor_name} — Rp {r.principal.toLocaleString('id-ID')}</option>)}
                </select>
              </div>
              <div><label>Alasan pembatalan</label><input value={voidReason} onChange={(e) => setVoidReason(e.target.value)} maxLength={250} required /></div>
            </div>
            <p className="muted">Hanya Super Admin yang dapat membatalkan hutang yang belum memiliki pembayaran.</p>
          </div>
        )}

        {formHutang && category === 'bayar_hutang' && (
          <div className="form-section">
            <div className="section-title">Pembayaran hutang</div>
            <div className="form-grid">
              <div>
                <label>Hutang aktif</label>
                <select value={debtId} onChange={(e) => setDebtId(e.target.value)} required>
                  <option value="">Pilih hutang aktif</option>
                  {debtAktif.map((r) => <option key={r.id} value={r.id}>{r.creditor_name} — sisa Rp {r.sisa.toLocaleString('id-ID')}</option>)}
                </select>
              </div>
              <div><label>Nominal pembayaran (Rp)</label><input type="number" min="1" step="1" value={amount} onChange={(e) => setAmount(e.target.value)} required /></div>
              <div className="span2"><label>Catatan</label><input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} /></div>
            </div>
          </div>
        )}

        {(!formKelola && !formHutang) && (
          <div className="form-section">
            <div className="section-title">{isKas ? 'Nominal' : 'Barang'}</div>
            <div className="form-grid">
              {isKas ? (
                <div>
                  <label>Nominal (Rp)</label>
                  <input type="number" min="1" value={amount} onChange={(e) => setAmount(e.target.value)} required />
                </div>
              ) : (
                <>
                  <div><label>Nama barang</label><input value={itemName} onChange={(e) => setItemName(e.target.value)} placeholder="Cincin, kalung, gelang..." required /></div>
                  <div><label>Berat total (gram)</label><input type="number" step="0.001" min="0.001" value={berat} onChange={(e) => setBerat(e.target.value)} required /></div>
                  <div><label>Kadar</label><input value={karat} onChange={(e) => setKarat(e.target.value)} placeholder="22K" /></div>
                  <div><label>Jumlah (pcs)</label><input type="number" min="1" value={qty} onChange={(e) => setQty(e.target.value)} required /></div>
                </>
              )}
              <div className="span2"><label>Catatan</label><input value={note} onChange={(e) => setNote(e.target.value)} /></div>
            </div>
          </div>
        )}

        {isKas && arahAktif === 'in' && category === 'omzet' && (
          <p className="muted">Penjualan barang lewat menu Stok sudah otomatis masuk ke kas. Pakai isian ini hanya untuk pemasukan penjualan di luar sistem, supaya tidak tercatat dua kali.</p>
        )}
        {!isKas && category === 'ke_toko' && <p className="muted">Otomatis tercatat sebagai barang masuk di toko tujuan.</p>}
        {!isKas && category === 'kembali_bo' && <p className="muted">Otomatis tercatat sebagai barang masuk di Back Office.</p>}

        <button disabled={loading}>{loading ? <><Spinner />Menyimpan...</> : 'Simpan'}</button>
      </form>

      {investorDetail !== undefined && (
        <DetailInvestor
          nama={investorDipilih?.name || ''}
          data={investorDetail}
          onTutup={() => setInvestorDetail(undefined)}
          onSimpan={segarkanDetail}
          onPesan={(message) => onSukses({ message })}
        />
      )}
    </>
  )
}
