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
`JobController`, `ApplicationController`, dan `RecommendationController`), memastikan setiap route yang aktif terdaftar
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
  dan lowongan mitra lain. Daftar default hanya `status: aktif`; gunakan
  `GET /api/jobs?status=ditutup` untuk lowongan ditutup. Detail berdasarkan ID
  tetap dapat dibaca saat ditutup.
- `GET /api/mitra/jobs` dan `POST /api/mitra/jobs`: daftar milik sendiri dan buat
  lowongan; khusus role `mitra`. Daftar milik sendiri tanpa filter status
  mencakup `aktif` dan `ditutup`.
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
harus cocok persis dengan enum berikut; kosong berarti tanpa filter.
`status` opsional, di-trim, dan case-sensitive: hanya `aktif` atau `ditutup`;
nilai lain menghasilkan `400`. Dihilangkan, string kosong, atau whitespace
memakai default endpoint: `aktif` untuk katalog, tanpa filter status untuk
daftar milik sendiri. Filter `search`, `category`, dan `status` digabung AND
pada count dan data. Urutan `created_at DESC, id ASC`; `paging` di tingkat
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
`status` opsional dengan enum `aktif`/`ditutup` (case-sensitive), di-trim sebelum
validasi. Dihilangkan, `null` (bind ke zero value string), `""`, atau whitespace
menjadi `aktif`; nilai lain menghasilkan `400`.

`company`, `source`, `mitra_id`, ID, dan timestamp dikelola server, bukan field
request. `company` memakai nama akun mitra saat create/update, `source` selalu
`mitra`, dan pemilik berasal dari identitas JWT; ownership tidak dapat dipindah.
Respons selalu menyertakan `status` (wajib, non-null, `aktif`/`ditutup`),
`mitra_id` (nullable), dan kedua daftar skill sebagai array. Create menghasilkan
`201`; GET/update/delete menghasilkan `200`.
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

### Tutup dan buka kembali

`status` pada PUT/PATCH opsional dan berbasis pointer: dihilangkan atau `null`
berarti tidak diubah. Nilai di-trim; `""` eksplisit atau whitespace ditolak
(`400`), bukan menjadi `aktif`. Hanya `aktif`/`ditutup` yang valid.

Gunakan route yang sama, `PATCH /api/mitra/jobs/{id}`, tanpa endpoint baru:

- Tutup: `{"status":"ditutup"}`.
- Buka kembali: `{"status":"aktif"}`.

Hanya pemilik dapat mengubah status. Mitra lain tetap mendapat `404`; aturan
role dan seed/tanpa pemilik tetap berlaku. PUT mendukung perubahan yang sama.

### Catatan migrasi

- Startup menjalankan GORM `AutoMigrate` untuk `jobs` dan `applications`.
- Kolom pemilik `mitra_id` pada `jobs` nullable dan berindeks. Pastikan migrasi
  kolom/index diterapkan pada database deployment; data seed lama tidak perlu
  diberi pemilik.
- Kolom `status` memiliki default database `aktif`, `NOT NULL`, dan index.
  Migrasi dari tabel tanpa kolom status mengisi row lama, termasuk seed, dengan
  `aktif`. Migrasi ulang mempertahankan `ditutup`, tidak membuka ulang lowongan.
- Relasi `applications` memiliki foreign key ke `jobs` dan `users` (`mahasiswa_id`) dengan cascade delete (`constraint:OnUpdate:CASCADE,OnDelete:CASCADE`).
- Lowongan seed/tanpa pemilik tetap `mitra_id: null`: terbaca di katalog, tidak
  muncul pada daftar milik mitra, dan read-only melalui API mitra. Batas input
  create/update tidak diberlakukan ulang pada data seed.
- Delete menghapus row katalog secara permanen.

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

## Recommendations: integrasi ML nyata

`POST /api/recommendations`, Bearer JWT, khusus `mahasiswa`; role lain `403`.
Tidak ada fallback data palsu, penyimpanan hasil, snapshot, atau route GET latest.
Tidak mengubah `analysis_status` cerita.

Body JSON wajib tepat satu key `consent_at`, string RFC3339 yang diberikan
secara eksplisit setelah persetujuan pengguna. Key duplikat/asing, `null`,
non-object, JSON tambahan, field hilang, atau timestamp invalid menghasilkan
`400`. Jangan kirim identitas, cerita, CV, profil manual, atau katalog dari klien.
Parser hanya memeriksa format timestamp; tidak memeriksa umur/waktu mendatang.
Consent tidak disimpan; cerita historis tidak memiliki consent tersimpan, sehingga
keberadaan cerita bukan bukti persetujuan. Jangan membuat timestamp otomatis
sebagai pengganti persetujuan eksplisit.

### Input terpercaya dan blocker data

- `student_id`: identitas JWT; `narrative`: `content` cerita milik sendiri terbaru,
  urutan `created_at DESC, id DESC`. Belum punya cerita: `404 Story not found`.
- CV profil saat ini berupa URL, bukan teks hasil ekstraksi. URL tidak dikirim
  sebagai teks; `cv_text: null`, `manual_profile: null`. Ekstraksi CV belum tersedia.
- Katalog: seluruh row DB `jobs` dengan `status=aktif`, urutan `id ASC`, tanpa
  pagination. Mapping field nyata: `ID` menjadi `job_id`, `Title` menjadi `title`,
  `Category` menjadi `category`, `Description` menjadi `description`,
  `RequiredSkills` menjadi `required_skills`, `OptionalSkills` menjadi
  `optional_skills`, `Source` menjadi `source`, `SourceURL` menjadi `source_url`,
  `CreatedAt.UTC()` menjadi `collected_at` berformat RFC3339Nano. Skill nil menjadi
  `[]`. `requirements_text` dan `is_synthetic` tidak tersedia di DB, tidak dikirim;
  default ML dipertahankan. Tidak ada penambahan field jobs atau nilai buatan.
- Sumber ML hanya `Glints`, `Dealls`, `Jobstreet`, `LinkedIn` (case-sensitive).
  Sumber `mitra` tidak diterima. Implementasi tidak menyaring row inkompatibel:
  satu row aktif tidak valid menggagalkan seluruh katalog dengan `422`.
- `source_url` boleh kosong pada CRUD jobs, tetapi wajib nonblank untuk ML.
  Katalog saat ini tidak dijamin memenuhi kontrak: sumber mitra atau URL kosong
  memblokir inferensi. Jangan mengganti sumber menjadi portal lain atau membuat URL.
- Maksimal 200 jobs; ID unik/nonblank, title nonblank, kategori resmi,
  `CreatedAt` nonzero, ID/title/category/source/source_url maksimal 2000 karakter,
  description maksimal 20000. Setiap daftar skill maksimal 200, entri nonblank
  maksimal 2000 karakter, tanpa duplikat case-insensitive. Semua row diperiksa;
  tidak ada pemotongan katalog otomatis. Katalog kosong dikirim sebagai `[]`.

### Respons dan kegagalan

`200`: `WebResponse` dengan `success: true`, `message: "Recommendation generated"`,
`data` langsung berisi payload ML, bukan envelope ML bersarang. `errors` dan
`paging` tidak disertakan. Schema presisi tersedia pada `RecommendationResponse`
dan `RecommendationItem` di [`openapi.json`](internal/apidocs/openapi.json).

- `student_id` sama dengan JWT; `generated_at` RFC3339; `schema_version: "1.0"`;
  `top_k: 3`, `items` array maksimal tiga, bukan jaminan tiga hasil.
- `model_version` provenance asli ML: segmen dipisahkan `;`, wajib prefiks persis
  `ner:`, `encoder:`, `ranker:` dengan suffix non-whitespace. Go tidak memaksakan
  urutan, tidak men-trim prefiks, mengizinkan segmen tambahan.
- `status`: `ok`, `parsial`, `profil_tidak_lengkap`. Status terakhir wajib
  `items: []`; `ok` wajib seluruh item `score_status: ok`.
- Item wajib seluruh key: `job_id`, `title`, `matching_score`, `cosine_similarity`,
  `readiness_score`, `matched_skills`, `missing_skills`, `score_status`, `rank`.
  ID unik dari katalog; title/entri skill nonblank, daftar skill non-null.
- `matching_score` dan `readiness_score`: integer 0–100 atau null;
  `cosine_similarity`: angka 0–1 atau null; `rank`: integer >=1 atau null.
  Null dipertahankan, bukan nol. Rank tidak diwajibkan berurutan atau <=3.
  `score_status=ok` wajib matching score non-null; `skor_belum_tersedia`,
  `persyaratan_belum_tersedia`, `profil_tanpa_keterampilan` wajib matching score
  null. `persyaratan_belum_tersedia` wajib `missing_skills: []`.

Status HTTP:

- `400`: body strict gagal atau `consent_at must be RFC3339`.
- `401`: JWT/identitas tidak valid; `403`: bukan mahasiswa.
- `404`: `Story not found`.
- `413`: body publik/payload internal >1 MiB atau upstream 413.
- `422`: `Catalog incompatible with ML contract` atau `Invalid recommendation input`
  (termasuk upstream 400).
- `500`: DB story/katalog gagal atau kegagalan internal controller.
- `502`: `Invalid ML response`; malformed, field hilang/asing, kontrak salah,
  gagal baca, atau respons >1 MiB.
- `503`: `ML service unavailable`; konfigurasi invalid/hilang, koneksi gagal,
  upstream non-200 selain 400/413, termasuk busy atau autentikasi internal gagal.
- `504`: `ML service timed out`.

Error dari recommendation controller menggunakan `success: false`, `message`
dan `errors` string sama. Berbeda dari error umum, `data` adalah struct zero value:
semua string `""`, `top_k: 0`, `items: null`, bukan `data: null`.
Error middleware dapat memakai `data: null`; error transport/framework dapat berbeda.

### Peluncuran dan konfigurasi

Petunjuk peluncuran model, artefak lokal, environment Python, keamanan jaringan,
dan smoke check: [`ml-service/README.md`](ml-service/README.md).
Go memanggil `POST /recommend` pada service internal, tanpa redirect/fallback.

Environment Go:

- `ML_SERVICE_URL`: origin `http`/`https` service, misalnya `http://127.0.0.1:8090`;
  tanpa credentials, query, fragment, atau path selain `/`.
- `ML_SERVICE_TOKEN`: secret internal yang sama dengan service Python, ASCII
  printable nonempty tanpa whitespace di awal/akhir. Bukan token JWT mahasiswa.
- `ML_REQUEST_TIMEOUT_SECONDS`: integer 1–120, default 60.

Konfigurasi hilang/invalid menonaktifkan inferensi (`503` ketika dipanggil), bukan
seluruh API. Validasi consent, story, dan katalog berlangsung sebelum pemanggilan
service sehingga dapat menghasilkan error lebih dahulu. Jangan mengekspos secret;
service ML tetap internal. Tidak ada klaim katalog/model deployment sudah siap.

## Batas cakupan

Kontrak mencakup endpoint aktif, termasuk delapan operasi jobs dan alias
auth/perusahaan. Dashboard saat ini hanya identitas login. Alur lupa password
mencakup pengiriman kode OTP, verifikasi kode, dan reset password beserta alias
`/api/auth/*`. Cerita dan `POST /api/recommendations` khusus role `mahasiswa`.
Rekomendasi memakai ML nyata; kesiapan bergantung konfigurasi, artefak, consent
 eksplisit, dan kompatibilitas katalog. Tidak ada persistence atau route latest.
