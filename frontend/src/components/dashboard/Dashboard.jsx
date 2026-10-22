import { useState, useEffect } from 'react'
import { api } from '../../services/api'
import KartuLokasi from './KartuLokasi'

export default function Dashboard() {
  const [saldo, setSaldo] = useState([])
  const [svc, setSvc] = useState({})
  const [error, setError] = useState('')

  useEffect(() => {
    api('/saldo').then(setSaldo).catch((err) => setError(err.message))
    api('/service/ringkas')
      .then((d) => setSvc(Object.fromEntries(d.map((x) => [x.location_id, x]))))
      .catch(() => {})
  }, [])

  return (
    <div>
      {error && <div className="error">{error}</div>}
      <div className="grid">
        {saldo.map((s) => (
          <KartuLokasi key={s.location_id} saldo={s} service={svc[s.location_id]} />
        ))}
      </div>
    </div>
  )
}
