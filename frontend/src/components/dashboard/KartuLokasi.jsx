import { rupiah } from '../../utils/format'

// rincian barang service yang jumlahnya lebih dari nol
const rincian = (s) =>
  [
    ['Di toko, menunggu diantar', s.di_toko],
    ['Menunggu dikirim ke penyervice', s.menunggu_servis],
    ['Sedang diservis', s.diproses],
    ['Kembali ke back office, menunggu konfirmasi', s.kembali_bo],
    ['Selesai, menunggu dioper', s.selesai],
    ['Siap diambil customer', s.siap_diambil],
  ].filter(([, n]) => n > 0)

// satu kartu ringkasan per lokasi: saldo kas, stok emas, dan barang service
export default function KartuLokasi({ saldo, service }) {
  return (
    <div className="card">
      <h3>{saldo.name}</h3>
      <div className="label">Saldo kas</div>
      <div className="big">{rupiah(saldo.saldo_kas)}</div>
      <div className="label">Stok emas</div>
      <div className="big">
        {saldo.stok_gram} gram <span className="muted">({saldo.stok_pcs} pcs)</span>
      </div>

      <div className="label">Barang service</div>
      <div className="big">
        {service ? service.di_sini : 0} <span className="muted">barang</span>
      </div>
      {service && rincian(service).map(([l, n]) => (
        <div className="row-line" key={l}>
          <span>{l}</span>
          <span>{n}</span>
        </div>
      ))}
      {service && service.masuk > 0 && <div className="muted">{service.masuk} barang sedang menuju sini</div>}
      {service && service.keluar > 0 && <div className="muted">{service.keluar} barang sedang dikirim keluar</div>}
    </div>
  )
}
