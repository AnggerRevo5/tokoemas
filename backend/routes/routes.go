package routes

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"toko-emas/controllers"
	"toko-emas/middleware"
)

// Setup mendaftarkan semua route aplikasi.
func Setup(r *gin.Engine, db *sql.DB) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.POST("/login", controllers.Login(db))

	// Semua route di grup ini wajib login (cookie sesi + token CSRF untuk request yang mengubah data).
	// Akun PJ dibatasi ke menu Lebur, Cuci, dan Service.
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware(db), middleware.BatasiPJ())
	auth.GET("/me", controllers.Me)
	auth.POST("/logout", controllers.Logout(db))
	auth.GET("/locations", controllers.ListLokasi(db))

	// Akun dan cabang (Super Admin)
	auth.GET("/akun", controllers.AkunList(db))
	auth.POST("/akun", controllers.AkunCreate(db))
	auth.PUT("/akun/:id", controllers.AkunUpdate(db))
	auth.POST("/akun/:id/password", controllers.AkunPassword(db))
	auth.POST("/lokasi", controllers.LokasiCreate(db))

	// Transaksi kas dan stok
	auth.POST("/kas", controllers.KasCreate(db))
	auth.GET("/kas", controllers.KasList(db))
	auth.GET("/stok/riwayat", controllers.RiwayatStok(db))
	auth.POST("/stok", controllers.StokCreate(db))
	auth.GET("/stok", controllers.StokList(db))
	auth.POST("/kas/:id/void", controllers.VoidTransaksi(db, "cash_transactions"))
	auth.POST("/stok/:id/void", controllers.VoidTransaksi(db, "stock_movements"))
	auth.GET("/saldo", controllers.Saldo(db))

	// Barang
	auth.POST("/barang", controllers.BarangCreate(db))
	auth.GET("/barang", controllers.BarangList(db))
	auth.POST("/barang/jual", controllers.BarangJual(db))
	auth.POST("/barang/kirim-bo", controllers.BarangKirimBO(db))
	auth.POST("/barang/terima", controllers.BarangTerimaBO(db))
	auth.GET("/barang/notif", controllers.BarangNotif(db))
	auth.POST("/barang/oper", controllers.BarangOper(db))
	auth.POST("/barang/terima-toko", controllers.BarangTerimaToko(db))
	auth.POST("/barang/serah-kurir", controllers.BarangSerahKurir(db))
	auth.POST("/barang/batal-kirim", controllers.BarangBatalKirim(db))
	auth.POST("/barang/tarik-bp", controllers.BarangTarikBP(db))
	auth.POST("/barang/terima-kembali", controllers.BarangTerimaKembali(db))
	// /barang/selesai-cuci ditutup: langsung mengembalikan barang ke stok tanpa konfirmasi owner
	// (tidak dipakai frontend; alur yang berlaku adalah /cucian/selesai lalu /barang/terima-kembali).
	auth.POST("/barang/void/:id", controllers.BarangVoid(db))
	auth.PUT("/barang/:id", controllers.BarangEdit(db))

	// Pembatalan dan koreksi penjualan dari Stok (cabang paling lama 7 hari, selebihnya Super Admin atau Owner)
	auth.POST("/jual-toko/:id/batal", controllers.JualBatal(db))
	auth.POST("/jual-toko/:id/koreksi", controllers.JualKoreksi(db))

	// Baki perhiasan
	auth.GET("/baki", controllers.BakiList(db))
	auth.POST("/baki", controllers.BakiCreate(db))
	auth.POST("/baki/pindah", controllers.BakiPindah(db))
	auth.POST("/baki/:id/hapus", controllers.BakiHapus(db))
	auth.POST("/baki/:id/nama", controllers.BakiRename(db))

	// Cuci
	auth.POST("/cuci", controllers.CuciMulai(db))
	auth.GET("/cuci", controllers.CuciList(db))
	auth.POST("/cuci/:id/kembali", controllers.CuciKembali(db))
	auth.POST("/cuci/:id/void", controllers.CuciVoid(db))
	auth.POST("/cuci/:id/kirim", controllers.CuciKirim(db))
	auth.POST("/cucian/kirim", controllers.CuciKirimTerpilih(db))
	auth.GET("/cucian/barang", controllers.CucianBarang(db))
	auth.POST("/cucian/selesai", controllers.CuciSelesaiBarang(db))
	auth.GET("/cucian/notif", controllers.CucianNotif(db))

	// Lebur
	auth.POST("/lebur/:id/kembali", controllers.LeburKembali(db))
	auth.POST("/lebur", controllers.LeburMulai(db))
	auth.GET("/lebur", controllers.LeburList(db))
	auth.POST("/lebur/:id/leburan", controllers.LeburCatatLeburan(db))
	auth.POST("/lebur/:id/cukim", controllers.LeburCatatCukim(db))
	auth.POST("/lebur/:id/undo", controllers.LeburUndo(db))

	// Penjualan
	auth.POST("/penjualan/batal/:id", controllers.PenjualanBatal(db))
	auth.GET("/penjualan/siap", controllers.PenjualanSiap(db))
	auth.POST("/penjualan", controllers.PenjualanPenadah(db))
	auth.GET("/penjualan", controllers.PenjualanList(db))
	auth.POST("/penjualan/void/:id", controllers.PenjualanVoid(db))
	auth.POST("/penjualan/:id/konfirmasi", controllers.PenjualanKonfirmasi(db))
	auth.PUT("/penjualan/:id", controllers.PenjualanEdit(db))

	// Talangan dan tukar barang dengan BP (toko menginput, owner memantau dan menyetujui)
	auth.GET("/talangan", controllers.TalanganList(db))
	auth.POST("/talangan", controllers.TalanganCreate(db))
	auth.PUT("/talangan/:id", controllers.TalanganEdit(db))
	auth.POST("/talangan/:id/konfirmasi", controllers.TalanganKonfirmasi(db))
	auth.POST("/talangan/:id/batal", controllers.TalanganBatal(db))
	auth.GET("/talangan/:id/riwayat", controllers.TalanganRiwayat(db))
	auth.GET("/tukar/:id/riwayat", controllers.TukarRiwayat(db))
	auth.GET("/tukar", controllers.TukarList(db))
	auth.POST("/tukar", controllers.TukarAjukan(db))
	auth.POST("/tukar/:id/setujui", controllers.TukarSetujui(db))
	auth.POST("/tukar/:id/batal", controllers.TukarBatal(db))

	// Gadai logam mulia: dana masuk, cicilan bulanan, tebus lebih cepat, barang kembali atau tidak ditebus
	auth.GET("/gadai", controllers.GadaiList(db))
	auth.POST("/gadai", controllers.GadaiCreate(db))
	auth.GET("/gadai/:id", controllers.GadaiDetail(db))
	auth.POST("/gadai/:id/bayar", controllers.GadaiBayar(db, false))
	auth.POST("/gadai/:id/tebus", controllers.GadaiBayar(db, true))
	auth.POST("/gadai/:id/kembali", controllers.GadaiAkhir(db, true))
	auth.POST("/gadai/:id/tidak-ditebus", controllers.GadaiAkhir(db, false))
	auth.POST("/gadai/:id/batal", controllers.GadaiBatal(db))
	auth.POST("/gadai-bayar/:id/void", controllers.GadaiBayarVoid(db))

	// Hutang dan dana
	auth.POST("/hutang", controllers.HutangCreate(db))
	auth.GET("/hutang", controllers.HutangList(db))
	auth.POST("/hutang/:id/bayar", controllers.HutangBayar(db))
	auth.POST("/dana/terima", controllers.HutangTerimaDana(db))
	auth.POST("/dana/void/:id", controllers.HutangVoid(db))

	// Investor: setoran pokok, bunga 1% per bulan, penarikan (Owner dan Super Admin)
	auth.GET("/investor", controllers.InvestorList(db))
	auth.POST("/investor", controllers.InvestorCreate(db))
	auth.GET("/investor/:id", controllers.InvestorDetail(db))
	auth.POST("/investor/:id/gerak", controllers.InvestorGerak(db))
	auth.POST("/investor/:id/ambil", controllers.InvestorAmbil(db))
	auth.POST("/investor-gerak/:id/void", controllers.InvestorGerakVoid(db))

	// Laporan
	auth.GET("/laporan/labarugi", controllers.LabaRugi(db))
	auth.GET("/laporan/preview", controllers.LaporanPreview(db))
	auth.GET("/laporan/pdf", controllers.LaporanPDF(db))
	auth.GET("/laporan/excel", controllers.LegacyLaporanExcel(db))

	// Service
	auth.POST("/service", controllers.ServiceCreate(db))
	auth.GET("/service", controllers.ServiceList(db))
	auth.POST("/service/aksi", controllers.ServiceAksi(db))
	auth.GET("/service/notif", controllers.ServiceNotif(db))
	auth.GET("/service/ringkas", controllers.ServiceRingkas(db))
	auth.POST("/service/void/:id", controllers.ServiceVoid(db))
}
