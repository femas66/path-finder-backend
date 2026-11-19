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

Catatan fixture saat ini: `TestOpenAPIDocumentation` belum memasang
`JobController` pada `RouteConfig`, sehingga delapan operasi jobs dilaporkan
sebagai route tidak terdaftar walaupun route produksi sudah ada. Untuk memulihkan
pemeriksaan kesetaraan, fixture perlu menyertakan `JobController`; jangan
menghapus operasi jobs dari kontrak untuk meloloskan test tersebut.

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

- Startup menjalankan GORM `AutoMigrate` untuk `jobs`; kolom pemilik `mitra_id`
  nullable dan berindeks. Pastikan migrasi kolom/index tersebut diterapkan pada
  database deployment; tidak perlu memberi pemilik pada data seed lama.
- Lowongan seed/tanpa pemilik tetap `mitra_id: null`: terbaca di katalog, tidak
  muncul pada daftar milik mitra, dan read-only melalui API mitra. Batas input
  create/update tidak diberlakukan ulang pada data seed.
- Delete menghapus row katalog secara permanen. Snapshot rekomendasi yang sudah
  tersimpan tidak diubah atau dihapus; `job_id` di snapshot dapat menunjuk row
  yang sudah tidak ada, sehingga GET katalog untuk ID tersebut mengembalikan
  `404`. Perubahan lowongan juga tidak menulis ulang snapshot lama.

## Batas cakupan

Kontrak mencakup endpoint aktif, termasuk delapan operasi jobs dan alias
auth/perusahaan. Dashboard saat ini hanya identitas login; lupa password hanya
mengirim OTP. Verifikasi OTP serta reset password belum didokumentasikan karena
belum ada route. Rekomendasi masih rule-based di Go, bukan integrasi layanan ML.
