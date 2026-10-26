import { m } from 'motion/react'
import { BellRing } from 'lucide-react'

// Banner kiriman barang yang menunggu konfirmasi (isi berbeda untuk akun cabang dan owner/admin).
export default function NotifBanner({ notif, isCabang, onLihat }) {
  if (notif.menunggu <= 0 && notif.keluar <= 0) return null

  return (
    <m.div
      className="banner"
      initial={{ opacity: 0, y: -8 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: 'easeOut' }}
    >
      <div className="banner-isi">
        <span className="banner-ikon">
          <BellRing size={16} strokeWidth={1.8} aria-hidden="true" />
        </span>
        <div>
        {isCabang ? (
          <>
            {notif.menunggu > 0 && (
              <div>
                <b>{notif.menunggu}</b> barang dioper ke toko ini, menunggu konfirmasi kamu
                {notif.belum_sampai > 0 && <> ({notif.belum_sampai} ditandai belum diterima)</>}
              </div>
            )}
            {notif.keluar > 0 && (
              <div>
                <b>{notif.keluar}</b> barang sudah diantar, menunggu konfirmasi back office
              </div>
            )}
          </>
        ) : (
          <>
            {notif.menunggu > 0 && (
              <div>
                <b>{notif.menunggu}</b> barang dari toko menunggu konfirmasi
                {notif.belum_sampai > 0 && <> ({notif.belum_sampai} ditandai belum sampai)</>}
              </div>
            )}
            {notif.keluar > 0 && (
              <div>
                <b>{notif.keluar}</b> barang dioper ke toko, menunggu konfirmasi toko tujuan
              </div>
            )}
          </>
        )}
        </div>
      </div>
      <button type="button" className="mini primary" onClick={onLihat}>
        Lihat
      </button>
    </m.div>
  )
}
