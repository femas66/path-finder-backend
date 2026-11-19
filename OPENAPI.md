# Dokumentasi API

## Akses

Jalankan backend seperti biasa dengan PostgreSQL dan `JWT_SECRET` terpasang.
Port default 3000:

- Swagger UI: http://localhost:3000/docs
- OpenAPI 3.0.3 JSON: http://localhost:3000/openapi.json

Dokumentasi dapat diakses tanpa login. Endpoint bisnis terlindungi tetap
memerlukan JWT dan role sesuai dokumentasi.

Login melalui `POST /api/users/login` atau alias `/api/auth/login`, salin
`data.token`, lalu masukkan token saja (tanpa prefiks `Bearer`) ke tombol
**Authorize**. Swagger UI menambahkan header Bearer. Token tidak disimpan lintas
reload (`persistAuthorization: false`). Gunakan akun dan database pengujian:
**Try it out menjalankan request sungguhan, termasuk operasi ubah dan hapus.**

## Sumber kontrak

`internal/apidocs/openapi.json` merupakan sumber kontrak statis, bukan hasil
generator. Kontrak dan halaman UI di-embed ke binary Go; runtime tidak perlu
membaca file dari working directory. Tidak ada dependency Go baru.

Saat menambah atau mengubah API:

1. Perbarui route, request/response model, dan implementasi.
2. Perbarui paths, schemas, validasi, status respons, security, serta alias dalam
   `internal/apidocs/openapi.json`. Setiap operasi wajib memiliki operationId unik.
3. Jalankan test dokumentasi dan unit test:

   ```sh
   go test ./test -run '^TestOpenAPIDocumentation$' -count=1
   go test ./internal/...
   ```

Test dokumentasi tidak memerlukan DB atau secret: test mendaftarkan route tanpa
memanggil handler bisnis, memeriksa JSON dan kecocokan operasi terhadap route
aktif. Test ini tidak menggantikan validator formal OpenAPI atau test kontrak
respons bisnis. Endpoint dokumentasi dan HEAD otomatis tidak dihitung sebagai
operasi bisnis. Perubahan kontrak otomatis masuk binary setelah build ulang.

Pemeriksaan kesetaraan pada `TestOpenAPIDocumentation` memasang seluruh controller aktif
(`UserController`, `AdminMahasiswaController`, `HajiCodingController`, `StoryController`,
`JobController`, dan `ApplicationController`), memastikan setiap route yang aktif terdaftar
secara lengkap dan akurat di `openapi.json`.

## CDN dan akses jaringan

Swagger UI memakai `swagger-ui-dist@5.17.14` dari `cdn.jsdelivr.net`. Browser
memerlukan akses internet ke CDN untuk CSS/JavaScript. Script CDN berjalan dalam
origin dokumentasi; gunakan kebijakan deployment yang sesuai untuk dokumentasi
yang mengizinkan token. Untuk deployment offline atau larangan script pihak
ketiga, vendor aset UI terlebih dahulu. `validatorUrl: null` menonaktifkan
validator eksternal Swagger agar kontrak tidak dikirim ke layanan validator.
File `/openapi.json` sendiri tidak memerlukan CDN dan bisa diimpor ke Postman
atau alat OpenAPI lain.

## Jobs: katalog dan CRUD mitra

Kontrak lengkap, schema request/response, parameter, dan contoh error tersedia di
[Swagger UI](http://localhost:3000/docs) (tag **Jobs**) dan
[`internal/apidocs/openapi.json`](internal/apidocs/openapi.json).

- `GET /api/jobs` dan `GET /api/jobs/{id}`: katalog untuk semua role login
  (`admin`, `mitra`, `mahasiswa`), bukan endpoint publik. Termasuk lowongan seed
  dan lowongan mitra lain.
- `GET /api/mitra/jobs` dan `POST /api/mitra/jobs`: daftar milik sendiri dan buat
  lowongan; khusus role `mitra`.
- `GET`, `PUT`, `PATCH`, `DELETE /api/mitra/jobs/{id}`: khusus `mitra`, hanya
  lowongan dengan `mitra_id` akun login. ID tidak ada, milik mitra lain, atau
  seed/tanpa pemilik menghasilkan `404` dengan `lowongan tidak ditemukan`.
  Role lain mendapat `403`; admin tidak mewarisi akses mitra.

Kedua endpoint daftar memakai `page` (default `1`) dan `size` (default `10`,
minimal `1`, maksimal `100`). Default hanya berlaku bila parameter tidak ada.
Nilai kosong, non-integer, nol, negatif, overflow, `size > 100`, atau offset
melampaui rentang integer menghasilkan `400`, bukan dikoreksi otomatis.
`search` di-trim dan mencari substring literal case-insensitive pada `title`
atau `company`; `%`, `_`, dan backslash bukan wildcard. `category` di-trim dan
harus cocok persis dengan enum berikut; kosong berarti tanpa filter. Kedua
filter digabung AND. Urutan `created_at DESC, id ASC`; `paging` di tingkat
teratas berisi `page`, `size`, `total_item`, `total_page`. Hasil kosong tetap
`data: []`; tanpa hasil `total_page: 0`.

Delapan kategori resmi (case-sensitive):

- `Software Engineer Intern`
- `Backend Developer Intern`
- `Mobile Developer Intern`
- `Frontend Developer Intern`
- `Data Analyst Intern`
- `UI/UX Designer Intern`
- `QA Engineer Intern`
- `Network & IT Support Intern`

Contoh body `POST /api/mitra/jobs` (Bearer token mitra diperlukan):

```json
{
  "title": "Backend Developer Intern",
  "category": "Backend Developer Intern",
  "description": "Mengembangkan API backend bersama tim produk.",
  "location": "Bandung",
  "work_type": "Hybrid",
  "required_skills": ["Go", "SQL"],
  "optional_skills": ["Docker"],
  "source_url": "https://example.com/jobs/backend-intern"
}
```

Create wajib `title` (3–150 karakter), `category`, `description` (20–10000),
`location` (1–255), `work_type` (1–50, string bebas), dan `required_skills`
(1–50 entri). `optional_skills` maksimal 50 entri; setiap entri skill 1–100
karakter. String dan entri skill di-trim sebelum validasi; skill dideduplikasi
case-insensitive sesudah batas input diperiksa, mempertahankan urutan dan ejaan
pertama. Entri kosong/whitespace ditolak. `source_url` maksimal 1000 karakter,
berupa URL `http`/`https` dengan hostname atau string kosong.

`company`, `source`, `mitra_id`, ID, dan timestamp dikelola server, bukan field
request. `company` memakai nama akun mitra saat create/update, `source` selalu
`mitra`, dan pemilik berasal dari identitas JWT; ownership tidak dapat dipindah.
Respons selalu menyertakan `mitra_id` (nullable) dan kedua daftar skill sebagai
array. Create menghasilkan `201`; GET/update/delete menghasilkan `200`.
Delete mengembalikan `data: null`, bukan `204`.

**PUT dan PATCH sama-sama partial update berbasis pointer, bukan replacement.**
Field dihilangkan atau `null` berarti tidak diubah. `{}` valid, tetapi
`company`/`source` tetap disinkronkan server dan `updated_at` diperbarui.
String wajib kosong dan `required_skills: []` ditolak. Untuk menghapus nilai
opsional tanpa mengubah field lain, kirim body berikut ke salah satu metode:

```json
{
  "optional_skills": [],
  "source_url": ""
}
```

### Catatan migrasi dan snapshot

- Startup menjalankan GORM `AutoMigrate` untuk `jobs` dan `applications`.
- Kolom pemilik `mitra_id` pada `jobs` nullable dan berindeks.
- Relasi `applications` memiliki foreign key ke `jobs` dan `users` (`mahasiswa_id`) dengan cascade delete (`constraint:OnUpdate:CASCADE,OnDelete:CASCADE`).
- Lowongan seed/tanpa pemilik tetap `mitra_id: null`: terbaca di katalog, tidak
  muncul pada daftar milik mitra, dan read-only melalui API mitra. Batas input
  create/update tidak diberlakukan ulang pada data seed.
- Delete menghapus row katalog secara permanen. Snapshot rekomendasi yang sudah
  tersimpan tidak diubah atau dihapus; `job_id` di snapshot dapat menunjuk row
  yang sudah tidak ada, sehingga GET katalog untuk ID tersebut mengembalikan
  `404`. Perubahan lowongan juga tidak menulis ulang snapshot lama.

## Pendaftaran Magang (Applications)

Kontrak lengkap, schema request/response, parameter, dan contoh error tersedia di
[Swagger UI](http://localhost:3000/docs) (tag **Applications**) dan
[`internal/apidocs/openapi.json`](internal/apidocs/openapi.json).

### Aturan Bisnis & Validasi

1. **Kelengkapan Profil:**
   Sebelum mendaftar lowongan magang, mahasiswa wajib melengkapi profil pribadinya:
   `profile_picture` (foto profil), `cv` (link/berkas CV), dan `portfolio_url` (link portofolio).
   Jika salah satu atau lebih belum terisi / hanya berupa spasi kosong, server menolak
   pendaftaran dengan status `400 Bad Request`:
   `"profil belum lengkap: foto profil, CV, dan link portofolio wajib diisi sebelum mendaftar"`.

2. **Aturan Satu Lamaran Aktif (Single Active Application):**
   - Ketika mahasiswa sudah mendaftar dan statusnya masih `pending`, mahasiswa tidak
     dapat mendaftar ke lowongan lain mana pun. Permintaan pendaftaran baru ditolak
     dengan status `400 Bad Request`:
     `"anda masih memiliki pendaftaran magang aktif yang sedang diproses"`.
   - Jika status pendaftaran ditolak (`rejected`) oleh pihak mitra, mahasiswa
     diperbolehkan mendaftar kembali ke lowongan lain.
   - Jika status pendaftaran diterima (`accepted`) oleh mitra, mahasiswa tidak dapat
     mendaftar ke lowongan manapun lagi secara permanen dengan status `400 Bad Request`:
     `"anda telah diterima di program magang lain dan tidak dapat mendaftar lagi"`.

3. **Otorisasi & Keamanan Status:**
   - Hanya role `mahasiswa` yang dapat mengajukan pendaftaran magang (role lain menghasilkan `403 Forbidden`).
   - Hanya role `mitra` pemilik lowongan yang dapat melihat pelamar dan mengubah status pendaftaran
     (mitra lain menghasilkan `403 Forbidden`).
   - Terminal status guard: Status pendaftaran yang sudah final (`accepted` atau `rejected`)
     tidak dapat diubah kembali (`400 Bad Request` - `"status lamaran sudah final dan tidak dapat diubah lagi"`).

4. **Lowongan Tanpa Mitra (Seed/Katalog):**
   - Lowongan katalog/seed awal yang tidak terikat dengan akun mitra (`mitra_id` bernilai null atau kosong) tidak menerima pendaftaran langsung melalui platform (`400 Bad Request` - `"lowongan ini tidak menerima pendaftaran langsung melalui platform"`).

### Endpoint Mahasiswa

- `POST /api/jobs/{id}/apply`: Mendaftar lowongan dengan ID lowongan di path parameter URL.
- `POST /api/mahasiswa/applications`: Mendaftar lowongan menggunakan request body JSON:
  ```json
  {
    "job_id": "018f-uuid-lowongan"
  }
  ```
- `GET /api/mahasiswa/applications`: Mengambil seluruh riwayat pendaftaran magang milik mahasiswa
  yang sedang login beserta detail relasi lowongan yang dilamar.

### Endpoint Mitra

- `GET /api/mitra/applications`: Mengambil daftar pelamar untuk lowongan yang dimiliki oleh
  mitra yang sedang login, dilengkapi objek `mahasiswa` (`MahasiswaApplicant`), objek `job`, dan metadata `paging` (`PageMetadata`).
  Mendukung query parameter:
  - `job_id`: filter opsional berdasarkan ID lowongan.
  - `page`: nomor halaman (default `1`, integer positif).
  - `size`: jumlah data per halaman (default `10`, minimal `1`, maksimal `100`).
  Nilai `page` atau `size` yang tidak valid (non-integer, <= 0, atau size > 100) menghasilkan respons `400 Bad Request`.
- `PATCH /api/mitra/applications/{id}/status` & `PUT /api/mitra/applications/{id}/status`:
  Memperbarui status lamaran pelamar.
  Contoh body:
  ```json
  {
    "status": "accepted"
  }
  ```
  Nilai `status` yang diizinkan hanya `"accepted"` atau `"rejected"`.

---

## Stories: Cerita Diri & Pembaruan Cerita

Kontrak lengkap tersedia di [Swagger UI](http://localhost:3000/docs) (tag **Stories**) dan
[`internal/apidocs/openapi.json`](internal/apidocs/openapi.json).

- `POST /api/stories`: Mengirim cerita diri baru. Khusus role `mahasiswa`.
- `GET /api/stories/me`: Mengambil cerita diri terbaru milik mahasiswa yang login.
- `PUT /api/stories/{id}` & `PATCH /api/stories/{id}`:
  Memperbarui isi cerita jika mahasiswa melakukan kesalahan/typo dalam input cerita diri.
  - Path parameter `{id}` menerima ID cerita (UUID) atau keyword `"me"` untuk secara langsung
    memperbarui cerita terbaru pengguna.
  - Body request:
    ```json
    {
      "content": "Isi cerita diri hasil revisi yang mendeskripsikan minat, pengalaman, dan keahlian secara komprehensif..."
    }
    ```
  - Validasi: `content` di-trim secara otomatis; panjang minimal 50 karakter dan maksimal 2000 karakter
    (`400 Bad Request` jika tidak memenuhi).
  - Server otomatis menghitung ulang `word_count` dan mereset `analysis_status` menjadi `"pending"`
    agar dapat dianalisis ulang.
  - Otorisasi: Khusus role `mahasiswa`. Cerita milik pengguna lain atau ID tidak ditemukan menghasilkan
    `404 Not Found`.

## Batas cakupan

Kontrak mencakup endpoint aktif, termasuk delapan operasi jobs dan alias
auth/perusahaan. Dashboard saat ini hanya identitas login; lupa password hanya
mengirim OTP. Verifikasi OTP serta reset password belum didokumentasikan karena
belum ada route. Rekomendasi masih rule-based di Go, bukan integrasi layanan ML.
