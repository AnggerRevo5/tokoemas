# Laporan Audit Keamanan — Pembukuan Toko Emas

Tanggal audit: 26 Oktober 2026
Cakupan: `backend/` (Go), `frontend/` (Vite), `docker-compose.yml`, Dockerfile, nginx, file konfigurasi/env, dan dependensi.

## Ringkasan

- **Stack:** frontend React 19 + Vite 8 (SPA tanpa router, navigasi lewat state). Backend Go 1.25 + Gin 1.12 + `database/sql` dengan driver `go-sql-driver/mysql` (tanpa ORM, semua query ditulis manual). Database MariaDB.
- **Cara serve di produksi:** hasil build Vite disajikan nginx (container `frontend`). nginx meneruskan `/api/*` ke backend. Publik masuk lewat Cloudflare Tunnel ke `127.0.0.1:8081`. Produksi **tidak** menjalankan `vite` dev server maupun `vite preview`.
- **Origin:** frontend dan API satu origin (`/api` di domain yang sama), jadi CORS tidak diperlukan dan memang tidak dipasang. Browser otomatis menolak request lintas origin.
- **Role:** `super_admin`, `owner` (back office), `cabang` (satu akun per toko, terikat `location_id`, misalnya Dinoyo dan Kepanjen), `pj` (penanggung jawab lebur/cuci/service). Role dan toko dibaca dari tabel `users` pada setiap request, bukan dari isian pengguna.
- **Hasil:** 3 temuan Tinggi, 9 Sedang, dan beberapa Rendah/Info. Tidak ada temuan Kritis: tidak ditemukan SQL injection, XSS sink, upload file, atau secret produksi yang ter-commit. Semua temuan Tinggi dan Sedang sudah diperbaiki.

## 1. Daftar temuan

| ID | Tingkat | Temuan | Lokasi | Status |
|---|---|---|---|---|
| T-01 | **Tinggi** | Role PJ bisa melampaui kewenangannya lewat API: membaca kas, saldo, mutasi stok, dan daftar barang semua lokasi; **menjual barang toko mana pun** (`POST /barang/jual` membuat catatan kas dan menandai barang terjual); dan mengantar barang toko ke back office (`POST /barang/kirim-bo`). Middleware `BatasiPJ` sudah ditulis tetapi tidak pernah dipasang, dan daftar izinnya juga keliru (tanpa `/cuci`, `/cucian`, `/locations`). | `routes/routes.go`, `middleware/roles.go`, `controllers/barang_controller.go`, `controllers/barang_transfer_controller.go`, `controllers/transaksi_controller.go`, `controllers/koreksi_controller.go` | Diperbaiki: `BatasiPJ` dipasang dengan daftar izin yang sesuai menu PJ (Lebur, Cuci, Service), ditambah pengecekan role eksplisit di `BarangJual` dan `BarangKirimBO` |
| T-02 | **Tinggi** | Token sesi (JWT 12 jam) disimpan di `localStorage`, sehingga bisa dicuri bila ada XSS. Tidak ada logout di server dan tidak ada pencabutan token: setelah password diganti, token lama tetap berlaku sampai habis. | `frontend/src/services/api.js`, `controllers/auth_controller.go`, `middleware/auth.go` | Diperbaiki: cookie `HttpOnly` + `Secure` + `SameSite=Strict`, sesi disimpan di tabel `user_sessions` (bisa dicabut), `POST /logout`, proteksi CSRF (lihat bagian 2) |
| T-03 | **Tinggi** | Endpoint login tanpa rate limit maupun lockout, sehingga password akun toko bisa ditebak tanpa batas. | `controllers/auth_controller.go` | Diperbaiki: maksimal 20 percobaan per IP per 10 menit; 5 kali gagal per username+IP dalam 15 menit → dikunci 15 menit |
| S-01 | Sedang | `POST /barang/selesai-cuci` membuat PJ bisa mengembalikan barang cucian langsung ke stok `tersedia`, **melewati konfirmasi owner**. Frontend tidak memakainya; alur resmi adalah `/cucian/selesai` → konfirmasi `/barang/terima-kembali`. | `routes/routes.go`, `controllers/cuci_controller.go` | Diperbaiki: route ditutup |
| S-02 | Sedang | Pesan error internal (teks error SQL, nama tabel/kolom) dikirim mentah ke client di 27 tempat. | `utils/response.go` dan 13 controller | Diperbaiki: client menerima pesan umum, detail hanya di log server. Pesan kesalahan isian tetap tampil apa adanya |
| S-03 | Sedang | Tidak ada validasi panjang teks dan rentang angka: catatan atau nama yang terlalu panjang memicu error database; nominal/berat bisa bernilai tidak masuk akal (misalnya Rp 10^15 atau 1 ton). | `models/*_dto.go` | Diperbaiki: tag `binding` di 130 field (teks ≤ ukuran kolom, nominal ≤ Rp 1 triliun, berat ≤ 100 kg, ID > 0, qty 0–10.000, biaya service ≥ 0, daftar ≤ 1000) |
| S-04 | Sedang | Tidak ada header keamanan, baik di API maupun di halaman (CSP, anti-clickjacking, nosniff, HSTS, Referrer-Policy, Permissions-Policy). | `main.go`, `frontend/nginx.conf` | Diperbaiki: middleware `HeaderKeamanan` (API) dan `frontend/security-headers.conf` (halaman) |
| S-05 | Sedang | Server HTTP tanpa timeout (rentan slowloris), tanpa batas ukuran body, panic dibalas tanpa format JSON, dan mode debug Gin hanya bergantung pada `GIN_MODE`. | `main.go` | Diperbaiki: `http.Server` dengan timeout, body maks 1 MB, recovery JSON, mode release otomatis saat `APP_ENV=production`, batas 600 request/menit per IP |
| S-06 | Sedang | Jejak audit tanpa IP, username, dan lokasi; perpindahan status barang tanpa nilai sebelum; login berhasil/gagal tidak tercatat. | `services/audit_service.go` | Diperbaiki: kolom baru di `audit_logs` dan tabel baru `security_events` (lihat bagian 2) |
| S-07 | Sedang | Parameter `jenis` di laporan bisa diulang (`jenis=kas,kas,kas,…` ribuan kali), sehingga satu request memicu ribuan query berat (DoS). | `controllers/laporan_controller.go` | Diperbaiki: tiap jenis hanya diproses sekali |
| S-08 | Sedang | Hash password memakai bcrypt cost 10 (di bawah rekomendasi 12). | `services/akun_service.go`, `services/user_service.go` | Diperbaiki: cost 12; hash lama diperbarui otomatis saat pemiliknya berhasil login (password tidak berubah) |
| S-09 | Sedang | Waktu respons login berbeda antara username yang tidak ada dan password salah, sehingga username yang terdaftar bisa ditebak. | `controllers/auth_controller.go` | Diperbaiki: bcrypt pembanding dijalankan juga saat username tidak ada; pesan tetap sama untuk semua alasan |
| R-01 | Rendah | Verifikasi JWT tidak mengunci algoritma secara eksplisit dan tanpa issuer/audience. Library sudah menolak `alg=none`, jadi tidak bisa dieksploitasi. | `middleware/auth.go` | Diperbaiki: hanya HS256, wajib `exp`, issuer, audience, dan ID sesi. Diuji di `middleware/keamanan_test.go` |
| R-02 | Rendah | DSN pengembangan dengan password `root` tertulis di kode (`config.go`) dan ada di riwayat git. | `backend/config/config.go` | Diperbaiki di kode (default tanpa password). **Perlu tindakan manual** bila password itu dipakai di MySQL mana pun (lihat bagian 3) |
| R-03 | Rendah | `quic-go` v0.59.0 rentan memory exhaustion (GO-2026-5676); dependensi tidak langsung dari Gin. | `backend/go.mod` | Diperbaiki: update ke v0.59.1 |
| R-04 | Rendah | `excelize` v2.11.0 rentan stack overflow saat membaca formula buatan penyerang (GO-2026-6688). | `backend/go.mod` | Diterima sebagai risiko: excelize hanya dipakai di `legacy_laporan.go` (tidak terhubung ke route) untuk **menulis** file, tidak pernah membaca spreadsheet dari luar. Perbaikannya baru ada sebagai versi pseudo yang belum rilis |
| R-05 | Rendah | Standard library Go 1.25.4 di laptop pengembangan punya beberapa kerentanan (crypto/tls, crypto/x509, net/http). | toolchain lokal | **Perlu tindakan manual**: update Go lokal. Image produksi `golang:1.25-alpine` memakai patch 1.25 terbaru saat di-build ulang |
| R-06 | Rendah | Kontainer belum diperketat. Backend sudah non-root dan multi-stage; nginx resmi menjalankan proses master sebagai root. | `docker-compose.yml` | Sebagian diperbaiki: backend `read_only`, `cap_drop: ALL`, `no-new-privileges`; frontend `no-new-privileges`. Opsional: ganti ke `nginxinc/nginx-unprivileged` (perlu mengubah port tunnel) |
| R-07 | Rendah | Panjang minimal password hanya 6 karakter. | `services/akun_service.go` | Diterima, karena menaikkannya mengubah aturan dan teks di form Akun. Rekomendasi: minimal 10 karakter, terutama untuk akun Super Admin/Owner |
| R-08 | Rendah | Nilai uang dan berat diproses sebagai `float64` di Go, padahal tersimpan sebagai `DECIMAL` di database. Penjumlahan di Go bisa selisih di bawah 1 sen. | backend | Diterima: tidak ditemukan bug integritas dan rumus tidak diubah |
| R-09 | Rendah | Pembatas login disimpan di memori, sehingga reset saat backend restart dan berlaku per instance. | `middleware/batas.go` | Diterima (hanya satu instance backend). Rekomendasi: tambah Rate Limiting Rule di Cloudflare untuk `/api/login` |
| I-01 | Info | File `controllers/legacy_*.go` berisi kode lama yang tidak terhubung ke route. | backend | Dibiarkan sesuai catatan di file; disesuaikan agar tetap terkompilasi. Rekomendasi: hapus |
| I-02 | Info | Tombol **Unduh Excel** di menu Laporan memanggil `/laporan/excel`, yang tidak ada di backend (respons 404). Ini bukan celah keamanan. | `frontend/src/pages/Laporan.jsx` | Tidak diubah (di luar cakupan); perlu diputuskan terpisah |
| I-03 | Info | Perintah `buatuser` menerima password sebagai argumen terminal, sehingga tercatat di riwayat shell server. | `services/user_service.go` | Rekomendasi: buat akun lewat menu Akun, atau hapus riwayat shell sesudahnya |

**Hal yang diperiksa dan tidak bermasalah:**
- SQL injection: semua nilai dari pengguna dikirim sebagai parameter `?`. Penggabungan string hanya untuk placeholder `IN (…)`, klausa tetap, dan nama tabel dari konstanta (sekarang dikunci dengan whitelist).
- IDOR dan isolasi toko: setiap aksi berdasarkan ID memeriksa lokasi dan role. Akun cabang hanya melihat dan mengubah barang/kas/service tokonya sendiri; lokasinya diambil dari database, bukan dari request.
- Alur status: semua perpindahan status mengecek status asal dan mengunci baris (`FOR UPDATE`). Barang tidak bisa lompat ke "diterima" tanpa back office atau toko tujuan.
- Mass assignment: semua handler memakai struct input khusus (DTO), tidak ada bind langsung ke model database.
- Frontend: tidak ada `dangerouslySetInnerHTML`/`innerHTML`, tidak ada variabel `VITE_*`, dan tidak ada file `.env` di folder frontend. Source map tidak dibuat dan diblokir di nginx.
- Docker: database tidak di-expose; backend hanya `expose` ke jaringan internal; frontend hanya di `127.0.0.1:8081`. Backend sudah multi-stage dan non-root.
- Riwayat git: `.env` tidak pernah ter-commit. File `db/migrations/002_users.sql` yang pernah ter-commit hanya berisi placeholder `<HASH_…>`, bukan hash asli. Pemeriksaan ini dilakukan sebelum Anda meminta saya berhenti menjalankan perintah git; setelah itu tidak ada perintah git yang saya jalankan.

## 2. Perubahan yang berdampak pada perilaku

| Perubahan | Dampak bagi pengguna |
|---|---|
| Sesi pindah dari `localStorage` ke cookie (`te_sesi` HttpOnly, Path `/api`; `te_csrf` dibaca frontend) | **Semua pengguna harus login ulang satu kali** setelah deploy. Token lama otomatis ditolak |
| Proteksi CSRF: setiap `POST/PUT` wajib membawa header `X-CSRF-Token` yang cocok dengan sesi | Tidak terasa bagi pengguna (frontend mengirimnya otomatis). Skrip/alat lain yang memanggil API harus login lewat `/login` dan mengirim header ini |
| Sesi berakhir setelah **12 jam** atau **4 jam tidak dipakai** (bisa diatur lewat `SESI_JAM` dan `SESI_IDLE_MENIT`) | Staf yang membiarkan aplikasi tidak dipakai lebih dari 4 jam harus login ulang |
| Logout mencabut sesi di server. Ganti password oleh Super Admin mengeluarkan akun itu dari perangkat lain. Akun dinonaktifkan langsung keluar dari semua perangkat | Akun toko yang dipakai di beberapa perangkat **tidak** ikut keluar saat satu perangkat logout |
| Lockout login: 5 kali salah per username dari IP yang sama → dikunci 15 menit; 20 percobaan per IP per 10 menit | Pesan "terlalu banyak percobaan login, coba lagi dalam N menit" (HTTP 429). Penguncian per IP, jadi orang luar tidak bisa mengunci akun toko |
| Akun PJ hanya bisa memanggil API Lebur, Cuci, Service, notifikasi, dan daftar lokasi | Tidak ada perubahan di layar PJ (menu PJ memang hanya itu) |
| `POST /barang/selesai-cuci` ditutup (404) | Tidak dipakai frontend |
| Validasi input baru | Isian yang melewati batas (misalnya catatan > 200 karakter) ditolak dengan "data tidak valid" |
| Error server dibalas dengan pesan umum | Detail error kini hanya ada di `docker compose logs backend` |
| Batas 600 request per menit per IP, body maksimal 1 MB | Jauh di atas pemakaian normal (input 300 barang sekaligus ±100 KB) |
| Cookie `Secure` di produksi | Login hanya berfungsi lewat **HTTPS** (domain Cloudflare). Membuka `http://127.0.0.1:8081` langsung di server tidak bisa login, kecuali `COOKIE_SECURE=false` |
| CSP di halaman | Bila nanti menambah skrip, font, atau gambar dari domain lain, tambahkan domainnya di `frontend/security-headers.conf` |
| **Migrasi database 019 (aditif)** | Tabel baru `user_sessions` dan `security_events`, serta kolom `username`, `location_id`, `ip` (NULL) di `audit_logs`. Tidak ada data lama yang diubah atau dihapus |

**Isi log audit yang baru:**
- `security_events`: login berhasil, login gagal (beserta alasan internal), login dibatasi, logout. Masing-masing dengan username, IP, user-agent, dan waktu.
- `audit_logs`: semua aksi kini menyimpan username, lokasi, dan IP pelaku. Perpindahan status barang (antar ke BO, konfirmasi BO, oper ke toko, konfirmasi toko, terima kembali) dan penjualan menyimpan nilai **sebelum** (status/lokasi tiap barang) dan **sesudah** (termasuk harga jual). Perubahan akun menyimpan role/lokasi/status lama dan baru.
- Password dan token tidak pernah dicatat.

### Cara deploy

1. **Backup database** `toko_emas`.
2. Jalankan migrasi `backend/migrations/019_keamanan.sql` lewat phpMyAdmin (database `toko_emas` → Import) atau lewat terminal (lihat `DEPLOYMENT.md`). Langkah ini bisa dilewati bila user `app` punya hak `CREATE`/`ALTER`, karena backend akan membuatnya sendiri saat start. Bila tabel sesi tetap tidak ada, backend berhenti dengan pesan yang menunjuk file migrasi ini.
3. (Disarankan) Ganti `JWT_SECRET` di `.env` dengan nilai baru: `openssl rand -hex 32`. Semua orang toh harus login ulang di deploy ini.
4. Jalankan `docker compose up -d --build`, lalu `docker compose logs backend`. Pastikan muncul `database terhubung` dan `server berjalan di :8080`, tanpa `peringatan: kolom audit_logs…`.
5. Uji lewat domain HTTPS: login sebagai owner, akun Dinoyo, akun Kepanjen, dan PJ; input barang; antar ke BO lalu konfirmasi; oper ke toko lalu konfirmasi; logout. Periksa baris baru di `security_events` dan `audit_logs`.
6. Cek `nginx -t` di container frontend (`docker compose exec frontend nginx -t`), karena konfigurasi nginx belum bisa saya uji di sini (lihat bagian 4).

## 3. Tindakan manual untuk Anda

**Secret**
- Tidak ditemukan secret produksi yang bocor ke repo. Rotasi `JWT_SECRET` tetap disarankan saat deploy ini (langkah 3 di atas).
- Password `rahasia123` sebelumnya tertulis di kode sebagai default pengembangan dan ada di riwayat git. **Bila password itu pernah dipakai untuk MySQL/MariaDB mana pun (laptop, server, phpMyAdmin), ganti.**
- Pastikan `.env` di server hanya bisa dibaca pemiliknya: `chmod 600 .env`.

**Variabel env baru (semuanya opsional, sudah ada di `.env.example`)**
- `SESI_JAM` (bawaan 12), `SESI_IDLE_MENIT` (bawaan 240).
- `TRUSTED_PROXIES`: CIDR proxy yang dipercaya untuk membaca IP asli. Bawaannya jaringan privat Docker; isi hanya bila jaringan Docker server memakai rentang lain.
- `COOKIE_SECURE=false`: hanya untuk uji lokal lewat http; jangan diisi di server.
- `COOKIE_PATH`: bawaan `/api`, ubah hanya bila prefix proxy berubah.

**Database**
- User database `app` sebaiknya bukan superuser: cukup `SELECT, INSERT, UPDATE, DELETE` (ditambah `CREATE, ALTER, INDEX, REFERENCES` bila ingin migrasi otomatis) pada `toko_emas.*`, tanpa `GRANT OPTION`, `FILE`, `SUPER`, atau akses ke database lain.
- Jalankan backup terjadwal yang disimpan di luar server dan uji proses pemulihannya.

**Cloudflare / server**
- Aktifkan **Always Use HTTPS** dan TLS minimum 1.2 di Cloudflare.
- Tambahkan Rate Limiting Rule untuk path `/api/login`, misalnya 10 request per menit per IP.
- Pertimbangkan **Cloudflare Access** (login email/OTP) di depan domain ini, karena aplikasi hanya dipakai staf internal. Ini lapisan perlindungan terkuat yang tersedia.
- Pastikan port 8081 dan port database tidak terbuka di firewall server (compose sudah mengikat ke `127.0.0.1`).
- Update Go di laptop pengembangan ke patch 1.25 terbaru, lalu jalankan `govulncheck ./...` lagi.
- Opsional: perbarui tag image `nginx:1.27-alpine`/`alpine:3.22` ke versi stable terbaru saat deploy berikutnya.

## 4. Konfirmasi dan verifikasi

**Alur bisnis tidak berubah.** Rumus perhitungan, struktur respons API yang dipakai frontend, dan tampilan tetap sama. Pengecualiannya: respons `/login` tidak lagi berisi `token` (frontend tidak membutuhkannya), dan endpoint `/barang/selesai-cuci` yang tidak dipakai kini ditutup. Pengguna yang sah tetap bisa melakukan semua yang biasa mereka lakukan; perubahannya terbatas pada yang tercantum di bagian 2.

**Hasil pemeriksaan otomatis**

| Pemeriksaan | Hasil |
|---|---|
| `go build ./...` | Lolos |
| `go vet ./...` | Lolos |
| `go test ./...` | Lolos. Test baru: `middleware/keamanan_test.go` (token `alg=none`/HS512/secret palsu/tanpa sesi/kedaluwarsa/audiens lain ditolak, CSRF terikat sesi, daftar izin PJ, lockout per username+IP, batas body, header keamanan) dan `models/validasi_test.go` (input normal diterima, 9 input tidak masuk akal ditolak) |
| `gofmt -l .` | Bersih |
| Frontend `npm run build` | Lolos, tanpa file `.map` di `dist` |
| Frontend `npm run lint` | 0 error, 18 warning (sama persis dengan sebelum audit; semuanya di hook lama yang tidak diubah) |
| `npm audit` | 0 kerentanan |
| `govulncheck ./...` | Sisa: excelize (R-04, diterima) dan standard library Go lokal (R-05, perlu update toolchain) |

**Yang belum diverifikasi.** Pemeriksaan berikut belum bisa saya jalankan:
- Alur login → transaksi → logout end-to-end terhadap database sungguhan. Saya tidak punya password akun aplikasi, dan tidak menjalankan migrasi pada database lokal Anda tanpa izin. Lakukan langkah uji nomor 5 di bagian 2 sebelum dipakai staf.
- `nginx -t` dan build image Docker, karena Docker Desktop di laptop sedang tidak berjalan. Lakukan langkah nomor 6.
