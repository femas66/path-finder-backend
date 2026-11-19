# Konfigurasi keamanan autentikasi

## Registrasi publik

`POST /api/users/register` dan alias `POST /api/auth/register` hanya menerima
role `mahasiswa` atau `mitra`. Jika role tidak diisi, default tetap `mahasiswa`.
Permintaan role `admin` ditolak dengan HTTP 400 sebelum mengakses database.
Akun admin harus diprovisikan lewat jalur internal yang dikendalikan operator,
bukan registrasi publik. Seeder yang ada hanya untuk data pengujian; jangan
jalankan dengan akun/password bawaannya pada produksi.

## Secret JWT wajib

Backend tidak lagi menyediakan secret bawaan. Set `JWT_SECRET` pada environment
proses; nilainya mengoverride `jwt.secret` dalam `config.json`. Alternatif untuk
lingkungan lokal: isi `jwt.secret` pada konfigurasi privat, jangan commit secret.
Startup ditolak jika secret kosong atau hanya whitespace.

Buat secret acak dengan `openssl rand -hex 32`, kemudian masukkan hasilnya ke
secret manager atau environment runtime. Docker Compose meneruskan `JWT_SECRET`
dan menolak startup jika variabel tersebut belum diisi.

Mengganti secret membuat token yang ditandatangani secret lama tidak berlaku;
pengguna perlu login ulang. Rotasi wajib jika sebelumnya memakai secret bawaan
yang tersimpan dalam repo. Periksa akun admin yang sudah ada: pembatasan
registrasi tidak menghapus akun yang terlanjur dibuat.

## Verifikasi tanpa database

```sh
go test ./internal/config ./internal/authconfig ./internal/usecase
go test ./test -run 'TestPublicRegistration|TestAuthRejectsMissingSecret|TestAuthUsesConfiguredSecret' -count=1
```

Test integrasi lain di `test/` memakai PostgreSQL dan dapat menghapus data uji.
Jalankan hanya dengan database khusus pengujian dan `JWT_SECRET` terpasang.
