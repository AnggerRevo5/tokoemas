import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import Pesan from '../components/common/Pesan'
import FormInvestor from '../components/investor/FormInvestor'
import TabelInvestor from '../components/investor/TabelInvestor'
import DetailInvestor from '../components/investor/DetailInvestor'

export default function Investor() {
  const [rows, setRows] = useState([])
  const [terpilih, setTerpilih] = useState(null)
  const [detail, setDetail] = useState(null)
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')

  const muatDaftar = useCallback(() => {
    api('/investor')
      .then(setRows)
      .catch((e) => setError(e.message))
  }, [])

  const muatDetail = useCallback((id) => {
    api(`/investor/${id}`)
      .then(setDetail)
      .catch((e) => {
        setError(e.message)
        setTerpilih(null)
      })
  }, [])

  useEffect(() => {
    muatDaftar()
  }, [muatDaftar])

  function resetPesan() {
    setError('')
    setInfo('')
  }

  function buka(id) {
    resetPesan()
    setTerpilih(id)
    setDetail(null)
    muatDetail(id)
  }

  const tutup = useCallback(() => {
    setTerpilih(null)
    setDetail(null)
  }, [])

  function muatUlang() {
    muatDaftar()
    muatDetail(terpilih)
  }

  const investor = rows.find((r) => r.id === terpilih)

  return (
    <div>
      <Pesan error={error} info={info} />

      <FormInvestor
        onMulai={resetPesan}
        onSukses={(res) => {
          setInfo(res.message)
          muatDaftar()
        }}
        onError={setError}
      />

      <div className="card" style={{ marginTop: 16 }}>
        <h3>Daftar Investor</h3>
        <TabelInvestor rows={rows} terpilih={terpilih} onPilih={buka} />
      </div>

      {terpilih && (
        <DetailInvestor
          key={terpilih}
          nama={investor ? investor.name : ''}
          data={detail}
          onTutup={tutup}
          onSimpan={muatUlang}
          onPesan={setInfo}
        />
      )}
    </div>
  )
}
