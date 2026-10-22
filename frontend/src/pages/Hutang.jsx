import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { rupiah, hariIni } from '../utils/format'
import Pesan from '../components/common/Pesan'
import KotakBatal from '../components/common/KotakBatal'
import FormHutang from '../components/hutang/FormHutang'
import TabelHutang from '../components/hutang/TabelHutang'

export default function Hutang({ me }) {
  const bisaVoid = me && me.role === 'super_admin'

  const [rows, setRows] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')
  const [warn, setWarn] = useState('')

  const [payId, setPayId] = useState(null)
  const [payAmount, setPayAmount] = useState('')
  const [payNote, setPayNote] = useState('')

  const [voidId, setVoidId] = useState(null)
  const [alasan, setAlasan] = useState('')

  const muat = useCallback(() => {
    api('/hutang')
      .then(setRows)
      .catch((e) => setError(e.message))
  }, [])

  useEffect(() => {
    muat()
  }, [muat])

  function resetPesan() {
    setError('')
    setInfo('')
    setWarn('')
  }

  function mulaiBayar(r) {
    setPayId(r.id)
    setVoidId(null)
    setPayAmount('')
    setPayNote('')
    resetPesan()
  }

  async function konfirmasiBayar() {
    resetPesan()
    try {
      const res = await api(`/hutang/${payId}/bayar`, {
        method: 'POST',
        body: JSON.stringify({ amount: Number(payAmount), note: payNote, paid_date: hariIni() }),
      })
      setInfo(res.lunas ? 'Pembayaran tersimpan, hutang LUNAS' : `Pembayaran tersimpan, sisa ${rupiah(res.sisa)}`)
      if (res.peringatan) setWarn(res.peringatan)
      setPayId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  function mulaiBatal(r) {
    setVoidId(r.id)
    setPayId(null)
    setAlasan('')
    resetPesan()
  }

  async function konfirmasiBatal() {
    if (!alasan.trim()) {
      setError('Alasan pembatalan wajib diisi')
      return
    }
    resetPesan()
    try {
      const res = await api(`/dana/void/${voidId}`, {
        method: 'POST',
        body: JSON.stringify({ reason: alasan }),
      })
      setInfo(res.message)
      setVoidId(null)
      muat()
    } catch (err) {
      setError(err.message)
    }
  }

  const payRow = rows.find((r) => r.id === payId)
  const voidRow = rows.find((r) => r.id === voidId)

  return (
    <div>
      <Pesan error={error} info={info} warn={warn && `${warn}. Pembayaran tetap tersimpan.`} />

      <FormHutang
        onMulai={resetPesan}
        onSukses={(res) => {
          setInfo(res.message)
          muat()
        }}
        onError={setError}
      />

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Daftar Hutang</h3>

        {payRow && (
          <KotakBatal
            danger={false}
            labelKonfirmasi="Bayar"
            onKonfirmasi={konfirmasiBayar}
            onTutup={() => setPayId(null)}
          >
            <div>
              Bayar hutang <b>{payRow.creditor_name}</b> (sisa {rupiah(payRow.sisa)}). Uang otomatis keluar dari
              kas back office.
            </div>
            <input
              type="number"
              min="1"
              value={payAmount}
              onChange={(e) => setPayAmount(e.target.value)}
              placeholder="Nominal pembayaran (Rp)"
              autoFocus
            />
            <input
              style={{ marginTop: 8 }}
              value={payNote}
              onChange={(e) => setPayNote(e.target.value)}
              placeholder="Catatan (opsional), contoh: cicilan 2"
            />
          </KotakBatal>
        )}

        {voidRow && (
          <KotakBatal onKonfirmasi={konfirmasiBatal} onTutup={() => setVoidId(null)}>
            <div>
              Batalkan hutang <b>{voidRow.creditor_name}</b> ({rupiah(voidRow.principal)}).
              {voidRow.dari_kas && ' Kas back office yang masuk ikut dibatalkan.'} Tulis alasannya:
            </div>
            <input value={alasan} onChange={(e) => setAlasan(e.target.value)} autoFocus />
          </KotakBatal>
        )}

        <TabelHutang rows={rows} bisaVoid={bisaVoid} onBayar={mulaiBayar} onBatal={mulaiBatal} />
      </div>
    </div>
  )
}
