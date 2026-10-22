// Banner kiriman barang yang menunggu konfirmasi (isi berbeda untuk akun cabang dan owner/admin).
export default function NotifBanner({ notif, isCabang, onLihat }) {
  if (notif.menunggu <= 0 && notif.keluar <= 0) return null

  return (
    <div className="banner">
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
      <button type="button" className="mini primary" onClick={onLihat}>
        Lihat
      </button>
    </div>
  )
}
