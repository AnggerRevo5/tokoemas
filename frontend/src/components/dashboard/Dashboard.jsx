import { useState, useEffect } from 'react'
import { BellRing, Scale, Wallet, Wrench } from 'lucide-react'
import { api } from '../../services/api'
import { gram, rupiah } from '../../utils/format'
import { KerangkaHalaman } from '../common/Skeleton'
import KartuLokasi from './KartuLokasi'
import KartuRingkasan from './KartuRingkasan'

// format angka saat bergulir (nilai akhir sama dengan format yang sudah dipakai)
const formatRupiah = (v) => rupiah(Math.round(v))
const formatGram = (v) => String(gram(v))
const formatJumlah = (v) => String(Math.round(v))

// perluTindakan = angka menu yang sudah dimuat AppShell: { stok, service, cuci }
export default function Dashboard({ perluTindakan }) {
  const [saldo, setSaldo] = useState([])
  const [svc, setSvc] = useState({})
  const [error, setError] = useState('')

  useEffect(() => {
    api('/saldo').then(setSaldo).catch((err) => setError(err.message))
    api('/service/ringkas')
      .then((d) => setSvc(Object.fromEntries(d.map((x) => [x.location_id, x]))))
      .catch(() => {})
  }, [])

  // ringkasan tampilan: jumlah dari kartu per lokasi yang sudah ada
  const totalKas = saldo.reduce((s, x) => s + Number(x.saldo_kas || 0), 0)
  const totalGram = saldo.reduce((s, x) => s + Number(x.stok_gram || 0), 0)
  const totalPcs = saldo.reduce((s, x) => s + Number(x.stok_pcs || 0), 0)
  const totalService = saldo.reduce((s, x) => s + Number((svc[x.location_id] || {}).di_sini || 0), 0)
  const t = perluTindakan || {}
  const totalTindakan = (t.stok || 0) + (t.service || 0) + (t.cuci || 0)

  return (
    <div>
      {error && <div className="error" style={{ marginBottom: 16 }}>{error}</div>}

      {saldo.length === 0 && !error ? (
        <KerangkaHalaman kartu={4} />
      ) : (
        <>
          <div className="ringkasan">
            <KartuRingkasan urutan={0} ikon={Wallet} label="Saldo kas" nilai={totalKas} format={formatRupiah} sub={`Total ${saldo.length} lokasi`} />
            <KartuRingkasan urutan={1} ikon={Scale} label="Stok emas" nilai={totalGram} format={formatGram} satuan="gram" sub={`${totalPcs} pcs`} />
            <KartuRingkasan urutan={2} ikon={Wrench} label="Barang service" nilai={totalService} format={formatJumlah} satuan="barang" sub="Di semua lokasi" />
            <KartuRingkasan
              urutan={3}
              ikon={BellRing}
              label="Perlu tindakan"
              nilai={totalTindakan}
              format={formatJumlah}
              sub={`Stok ${t.stok || 0} · Service ${t.service || 0} · Cuci ${t.cuci || 0}`}
            />
          </div>

          <div className="judul-bagian">Per lokasi</div>
          <div className="grid">
            {saldo.map((s) => (
              <KartuLokasi key={s.location_id} saldo={s} service={svc[s.location_id]} />
            ))}
          </div>
        </>
      )}
    </div>
  )
}
