# Menjalankan aplikasi di server dengan Cloudflare Tunnel

Konfigurasi ini menjalankan frontend React, API Go, dan MySQL dengan Docker Compose. Frontend dipublikasikan pada `127.0.0.1:8081`; port database dan API tidak dipublikasikan.

## 1. Siapkan server

Pasang Docker Engine dan Docker Compose plugin di server Linux, lalu salin/clone folder proyek ke server. Jalankan perintah berikut dari folder proyek:

```sh
cp .env.example .env
```

Edit `.env` dan ganti semua nilai contoh:

- `MYSQL_ROOT_PASSWORD` dan `DB_PASSWORD`: password kuat, gunakan karakter alfanumerik.
- `JWT_SECRET`: string acak minimal 32 karakter.

Jangan membagikan atau commit file `.env`.

## 2. Hubungkan domain melalui Cloudflare Tunnel

Tidak perlu membuat Tunnel atau token baru. Pada Tunnel yang sudah terhubung, buka route untuk domain aplikasi dan atur origin service sesuai cara `cloudflared` dijalankan:

Jika `cloudflared` berjalan langsung sebagai service di host server:

- **Service type:** `HTTP`
- **URL:** `http://127.0.0.1:8081`

Jika `cloudflared` berjalan di container Docker, `127.0.0.1:8081` adalah container Tunnel itu sendiri, bukan server. Periksa network Tunnel dan ID container frontend:

```sh
docker inspect cloudflared --format '{{range $name, $conf := .NetworkSettings.Networks}}{{println $name}}{{end}}'
docker compose ps -q frontend
```

Setelah memastikan network mana yang digunakan Tunnel, hubungkan frontend saja ke network itu—jangan hubungkan container Tunnel ke network default aplikasi yang juga berisi database. Origin route Tunnel menjadi `http://frontend:80`. Untuk konfigurasi permanen, deklarasikan network Tunnel sebagai external pada service `frontend` di Compose aplikasi.

Gunakan hostname yang memang sudah dikonfigurasi di route Tunnel. Mengubah origin pada route yang sudah ada akan mengarahkan hostname itu ke aplikasi ini. Domain harus dikelola Cloudflare. Tidak perlu membuka port inbound router/server atau memasang port forwarding.

## 3. Jalankan aplikasi

```sh
docker compose up -d --build
docker compose ps
```

Pembuatan schema dasar dijalankan otomatis saat volume database pertama kali dibuat. Untuk database yang sudah ada, jalankan migrasi tabel pengguna berikut sebelum membuat akun. Pastikan tabel `locations` sudah tersedia; migrasi aman dijalankan ulang dan tidak mengubah tabel `users` jika tabel itu sudah ada:

```sh
docker compose exec -T db sh -c 'mysql -uroot -p"$MYSQL_ROOT_PASSWORD" "$MYSQL_DATABASE"' < db/migrations/001_create_users.sql
```

Buat akun pertama setelah migrasi berhasil:

```sh
docker compose run --rm backend buatuser admin 'GANTI_DENGAN_PASSWORD_KUAT' owner
```

Untuk akun cabang gunakan role `cabang` dan tambahkan ID lokasi (`1` untuk Kepanjen atau `2` untuk Mall Dinoyo):

```sh
docker compose run --rm backend buatuser namauser 'GANTI_DENGAN_PASSWORD_KUAT' cabang 1
```

Sesudah Tunnel route disimpan dan aplikasi berjalan, buka domain tersebut melalui HTTPS.

## Operasi dan pemeliharaan

```sh
docker compose logs -f
docker compose pull
docker compose up -d --build
```

Data database disimpan pada volume Docker `db_data`; image/container dapat dibangun ulang tanpa menghapus volume itu. Jadwalkan backup MySQL yang disimpan di luar server, dan uji pemulihannya. Jangan menjalankan `docker compose down -v` kecuali memang ingin menghapus seluruh data database.

Untuk perubahan pada `db/init.sql` setelah database dibuat, volume yang sudah ada tidak menjalankan ulang file inisialisasi tersebut. Buat dan terapkan migrasi SQL secara terpisah.
