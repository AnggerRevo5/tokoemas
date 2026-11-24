import { useState, useEffect, useCallback } from 'react'
import { api } from '../services/api'
import { STATUS_MASUK, STATUS_KELUAR, LABEL_STATUS } from '../constants/talangan'
import Pesan from '../components/common/Pesan'
import TabelTalangan from '../components/talangan/TabelTalangan'
import FormTukar from '../components/talangan/FormTukar'
import DaftarTukar from '../components/talangan/DaftarTukar'
import PiutangBP from '../components/talangan/PiutangBP'
import Lipat from '../components/common/Lipat'

// Menu Talangan: barang milik BP lain yang dibeli toko (MASUK), barang kita yang dibeli BP (KELUAR),
// penukaran per cabang BP, dan piutang. Barang MASUK diinput petugas toko lewat Stok > Tambah barang masuk > Barang cabang.
export default function Talangan({ me }) {
  const lengkap = me.role !== 'cabang'
  const bisaTukar = me.role === 'pj' || me.role === 'super_admin'
  const tabAda = lengkap ? ['masuk', 'keluar', 'tukar'] : ['masuk']
  const NAMA_TAB = { masuk: 'Masuk (barang BP)', keluar: 'Keluar (barang kita)', tukar: 'Penukaran' }

  const [tab, setTab] = useState('masuk')
  const [bpList, setBpList] = useState([])
  const [bp, setBp] = useState('')
  const [status, setStatus] = useState('')
  const [rows, setRows] = useState([])
  const [tukar, setTukar] = useState([])
  const [piutang, setPiutang] = useState([])
  const [error, setError] = useState('')
  const [info, setInfo] = useState('')

  const reset = useCallback(() => {
    setError('')
    setInfo('')
  }, [])
  // callback stabil: anak memakainya sebagai dependensi efek, jangan dibuat ulang tiap render
  const galat = useCallback((m) => {
    setInfo('')
    setError(m)
  }, [])
  const kabar = useCallback((m) => {
    setError('')
    setInfo(m)
  }, [])

  const muat = useCallback(() => {
    api('/talangan/bp').then(setBpList).catch((e) => setError(e.message))
    const q = new URLSearchParams()
    if (tab === 'masuk' || tab === 'keluar') {
      q.set('arah', tab === 'masuk' ? 'MASUK' : 'KELUAR')
      if (bp) q.set('bp_id', bp)
      if (status) q.set('status', status)
      api(`/talangan?${q}`).then(setRows).catch((e) => setError(e.message))
    }
    if (tab === 'tukar') {
      api(`/talangan/tukar${bp ? `?bp_id=${bp}` : ''}`).then(setTukar).catch((e) => setError(e.message))
      api('/talangan/piutang').then(setPiutang).catch((e) => setError(e.message))
    }
  }, [tab, bp, status])

  useEffect(() => {
    muat()
  }, [muat])

  const pilihTab = (t) => {
    reset()
    setTab(t)
    setStatus('')
  }
  const daftarStatus = tab === 'keluar' ? STATUS_KELUAR : STATUS_MASUK

  return (
    <div>
      <Pesan error={error} info={info} />

      <div className="tab-mini" style={{ marginTop: 0 }}>
        {tabAda.map((t) => (
          <button key={t} type="button" className={tab === t ? 'aktif' : ''} onClick={() => pilihTab(t)}>{NAMA_TAB[t]}</button>
        ))}
      </div>

      {(tab === 'masuk' || tab === 'keluar' || tab === 'tukar') && (
        <div className="card" style={{ marginBottom: 16 }}>
          <div className="form-grid">
            <div>
              <label>Cabang BP</label>
              <select value={bp} onChange={(e) => setBp(e.target.value)}>
                <option value="">Semua</option>
                {bpList.map((b) => <option key={b.id} value={b.id}>{b.nama}</option>)}
              </select>
            </div>
            {tab !== 'tukar' && (
              <div>
                <label>Status</label>
                <select value={status} onChange={(e) => setStatus(e.target.value)}>
                  <option value="">Semua</option>
                  {daftarStatus.map((s) => <option key={s} value={s}>{LABEL_STATUS[s]}</option>)}
                </select>
              </div>
            )}
          </div>
        </div>
      )}

      {tab === 'masuk' && !lengkap && (
        <p className="muted" style={{ marginBottom: 12 }}>
          Untuk mencatat barang baru, buka menu Stok, pilih Tambah barang masuk, lalu asal barang &quot;Barang cabang BP (talangan)&quot;.
        </p>
      )}

      {(tab === 'masuk' || tab === 'keluar') && (
        <div className="card">
          <TabelTalangan rows={rows} me={me} onUbah={muat} onError={galat} onInfo={kabar} />
        </div>
      )}

      {tab === 'tukar' && (
        <>
          {bisaTukar && (
            <Lipat judul="Buat penukaran dengan cabang BP">
              <FormTukar bpList={bpList} onMulai={reset} onSelesai={muat} onError={galat} onInfo={kabar} />
            </Lipat>
          )}
          <Lipat judul="Cabang BP dan daftar penukaran">
            <div className="card">
              <PiutangBP rows={piutang} me={me} onUbah={muat} onError={galat} onInfo={kabar} />
              <h3 style={{ marginTop: 20 }}>Daftar penukaran</h3>
              <DaftarTukar rows={tukar} me={me} onUbah={muat} onError={galat} onInfo={kabar} />
            </div>
          </Lipat>
        </>
      )}
    </div>
  )
}
