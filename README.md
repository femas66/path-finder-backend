# Path Finder Backend

Backend REST API untuk layanan Path Finder yang dibangun menggunakan Go (Fiber) dan PostgreSQL.

## Prasyarat

- Docker dan Docker Compose
- Go versi 1.27 atau lebih baru (opsional, jika ingin menjalankan aplikasi langsung di host)

## Konfigurasi Environment

1. Salin file template `.env.example` menjadi `.env`:

```bash
cp .env.example .env
```

2. Buka file `.env` dan sesuaikan nilainya sesuai kebutuhan lokal Anda:

```env
JWT_SECRET=your_secret_key_here
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password_here
DB_NAME=path_finder_db
```

## Cara Menjalankan Proyek

Pilih salah satu cara di bawah ini.

### Cara 1: Menggunakan Docker Compose (Direkomendasikan)

Perintah ini akan menyalakan container PostgreSQL dan container aplikasi backend sekaligus:

```bash
docker compose up --build
```

Untuk menghentikan container:

```bash
docker compose down
```

### Cara 2: Menjalankan Secara Manual di Local

Jika ingin menjalankan aplikasi Go langsung di mesin lokal tanpa container backend:

1. Nyalakan container PostgreSQL terlebih dahulu:

```bash
docker compose up -d postgres
```

2. Jalankan server aplikasi backend:

```bash
go run ./cmd/web
```

Catatan untuk pengguna Windows PowerShell: Anda juga dapat menggunakan skrip otomatis:

```powershell
.\start-backend.ps1
```

## Seeder Akun Awal (Opsional)

Untuk mengisi data awal akun (Admin, Mitra, dan Mahasiswa) ke database:

```bash
go run ./cmd/seeder
```

## Akses Layanan dan Dokumentasi

Setelah server berjalan, layanan dapat diakses melalui URL berikut:

- Base URL API: http://localhost:3000
- Dokumentasi Swagger UI: http://localhost:3000/docs
- Spesifikasi OpenAPI JSON: http://localhost:3000/openapi.json
