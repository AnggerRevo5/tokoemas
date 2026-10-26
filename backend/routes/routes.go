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
	auth.POST("/barang/terima-kembali", controllers.BarangTerimaKembali(db))
	// /barang/selesai-cuci ditutup: langsung mengembalikan barang ke stok tanpa konfirmasi owner
	// (tidak dipakai frontend; alur yang berlaku adalah /cucian/selesai lalu /barang/terima-kembali).
	auth.POST("/barang/void/:id", controllers.BarangVoid(db))

	// Cuci
	auth.POST("/cuci", controllers.CuciMulai(db))
	auth.GET("/cuci", controllers.CuciList(db))
	auth.POST("/cuci/:id/kembali", controllers.CuciKembali(db))
	auth.POST("/cuci/:id/void", controllers.CuciVoid(db))
	auth.POST("/cuci/:id/kirim", controllers.CuciKirim(db))
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

	// Hutang dan dana
	auth.POST("/hutang", controllers.HutangCreate(db))
	auth.GET("/hutang", controllers.HutangList(db))
	auth.POST("/hutang/:id/bayar", controllers.HutangBayar(db))
	auth.POST("/dana/terima", controllers.HutangTerimaDana(db))
	auth.POST("/dana/void/:id", controllers.HutangVoid(db))

	// Laporan
	auth.GET("/laporan/labarugi", controllers.LabaRugi(db))
	auth.GET("/laporan/preview", controllers.LaporanPreview(db))
	auth.GET("/laporan/pdf", controllers.LaporanPDF(db))

	// Service
	auth.POST("/service", controllers.ServiceCreate(db))
	auth.GET("/service", controllers.ServiceList(db))
	auth.POST("/service/aksi", controllers.ServiceAksi(db))
	auth.GET("/service/notif", controllers.ServiceNotif(db))
	auth.GET("/service/ringkas", controllers.ServiceRingkas(db))
	auth.POST("/service/void/:id", controllers.ServiceVoid(db))
}
