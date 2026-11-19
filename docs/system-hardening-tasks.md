# Rencana & Pembagian Tugas Perbaikan Sistem (System Hardening & Bug Fixes)

Dokumen ini membagi seluruh temuan bug, celah keamanan, race condition, dan edge cases ke dalam tugas-tugas kecil yang terisolasi, terukur, dan dapat dieksekusi secara bertahap menggunakan pendekatan Test-Driven Development (TDD).

---

## Ringkasan Prioritas Tugas

| No | Tugas | Prioritas | Dampak Utama |
|---|---|---|---|
| **1** | [Proteksi Pendaftaran Lowongan Seed (Tanpa Mitra)](#tugas-1-proteksi-pendaftaran-lowongan-seed-tanpa-mitra) | **P0 (Kritis)** | Mencegah akun mahasiswa terkunci permanen (*deadlock*) |
| **2** | [Proteksi Concurrency Pendaftaran Ganda (TOCTOU)](#tugas-2-proteksi-concurrency-pendaftaran-ganda-toctou) | **P0 (Kritis)** | Menjamin integritas data agar mahasiswa tidak bisa melamar ganda secara simultan |
| **3** | [Pembaruan Status Lamaran Atomik (Atomic Status Transition)](#tugas-3-pembaruan-status-lamaran-atomik-atomic-status-transition) | **P1 (Tinggi)** | Mencegah konflik penimpaan status akhir lamaran (*race condition*) |
| **4** | [Proteksi Penghapusan Lowongan Berpelamar Aktif](#tugas-4-proteksi-penghapusan-lowongan-berpelamar-aktif) | **P1 (Tinggi)** | Mencegah rusaknya riwayat lamaran dan kebocoran aturan mahasiswa yang sudah diterima |
| **5** | [Validasi dan Sanitasi URL Profil Mahasiswa](#tugas-5-validasi-dan-sanitasi-url-profil-mahasiswa) | **P1 (Tinggi)** | Menangkal serangan Stored XSS dan skema URI berbahaya |
| **6** | [Sanitasi Error Handler HTTP 500 (CWE-209)](#tugas-6-sanitasi-error-handler-http-500-cwe-209) | **P2 (Sedang)** | Mencegah kebocoran skema dan query internal database ke publik |
| **7** | [Paginasi pada Endpoint Daftar Pelamar Mitra](#tugas-7-paginasi-pada-endpoint-daftar-pelamar-mitra) | **P2 (Sedang)** | Mencegah lonjakan memori (*OOM/DoS*) saat jumlah pelamar besar |
| **8** | [Penyempurnaan Modul Cerita (Stories Edge Cases)](#tugas-8-penyempurnaan-modul-cerita-stories-edge-cases) | **P2 (Sedang)** | Case-insensitive alias, deterministic sorting, dan scoped update |

---

### Tugas 1: Proteksi Pendaftaran Lowongan Seed (Tanpa Mitra)

**Tujuan:**
Mencegah mahasiswa melamar pada lowongan awal/seed katalog yang tidak memiliki akun mitra (`mitra_id = nil`), karena lamaran tersebut tidak akan pernah bisa diproses oleh siapapun sehingga mengunci akun mahasiswa selamanya dalam status `pending`.

**File yang Terlibat:**
- Ubah: `internal/usecase/application_usecase.go`
- Uji: `test/application_usecase_test.go`, `test/application_integration_test.go`
- Dokumen: `OPENAPI.md`

**Langkah Pengerjaan:**
- [x] **Langkah 1.1:** Tambahkan pengecekan `if job.MitraID == nil || strings.TrimSpace(*job.MitraID) == ""` pada fungsi `Apply` di `internal/usecase/application_usecase.go`.
- [x] **Langkah 1.2:** Kembalikan respons error `400 Bad Request` dengan pesan: `"lowongan ini tidak menerima pendaftaran langsung melalui platform"`.
- [x] **Langkah 1.3:** Buat unit test pada `test/application_usecase_test.go` untuk memverifikasi mahasiswa gagal mendaftar ke lowongan tanpa mitra.
- [x] **Langkah 1.4:** Perbarui deskripsi validasi pada `OPENAPI.md`.

---

### Tugas 2: Proteksi Concurrency Pendaftaran Ganda (TOCTOU)

**Tujuan:**
Mencegah celah waktu antara pengecekan lamaran aktif (`FindActiveByMahasiswaID`) dan penyimpanan (`Create`) yang memungkinkan mahasiswa melamar ke beberapa lowongan sekaligus melalui request paralel/simultan.

**File yang Terlibat:**
- Ubah: `internal/config/app.go` (atau file migrasi database)
- Ubah: `internal/repository/application_repository.go`
- Ubah: `internal/usecase/application_usecase.go`
- Uji: `test/application_usecase_test.go`

**Langkah Pengerjaan:**
- [x] **Langkah 2.1:** Tambahkan pembuatan *partial unique index* pada startup/migrasi di PostgreSQL:
  ```sql
  CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_active_application 
  ON applications (mahasiswa_id) 
  WHERE status IN ('pending', 'accepted');
  ```
- [x] **Langkah 2.2:** Bungkus alur pendaftaran di `ApplicationUseCase.Apply` ke dalam database transaction (`db.Transaction`).
- [x] **Langkah 2.3:** Tangani error duplicate key PostgreSQL (`SQLSTATE 23505`) di usecase agar menghasilkan respons ramah pengguna `400 Bad Request` (`"anda masih memiliki pendaftaran magang aktif yang sedang diproses"`).
- [x] **Langkah 2.4:** Buat unit test konkurensi (menjalankan 5 goroutine `Apply` paralel untuk user yang sama) dan pastikan hanya tepat 1 lamaran yang berhasil (HTTP 201) dan 4 lainnya ditolak (HTTP 400).

---

### Tugas 3: Pembaruan Status Lamaran Atomik (Atomic Status Transition)

**Tujuan:**
Menghilangkan race condition pada pembaruan status lamaran agar status yang sudah final (`accepted` atau `rejected`) tidak dapat ditimpa oleh request bersamaan.

**File yang Terlibat:**
- Ubah: `internal/repository/application_repository.go`
- Ubah: `internal/usecase/application_usecase.go`
- Uji: `test/application_usecase_test.go`

**Langkah Pengerjaan:**
- [x] **Langkah 3.1:** Ubah method `UpdateStatus` pada `ApplicationRepository` agar menyertakan klausul kondisi SQL:
  ```go
  result := db.Model(&entity.Application{}).
      Where("id = ? AND status = ?", id, entity.ApplicationStatusPending).
      Update("status", status)
  ```
- [x] **Langkah 3.2:** Jika `result.RowsAffected == 0`, kembalikan sentinel error kustom (misal `ErrApplicationStatusNotPending`).
- [x] **Langkah 3.3:** Di `ApplicationUseCase.UpdateStatus`, tangani sentinel error tersebut dengan mengembalikan `400 Bad Request` (`"status lamaran sudah final dan tidak dapat diubah lagi"`).
- [x] **Langkah 3.4:** Buat unit test yang memanggil pembaruan status dua kali secara bersamaan dan pastikan request kedua gagal ditolak.

---

### Tugas 4: Proteksi Penghapusan Lowongan Berpelamar Aktif

**Tujuan:**
Mencegah terhapusnya data pelamar secara otomatis (*cascade delete*) saat mitra menghapus lowongan yang sedang berjalan atau sudah memiliki pelamar berstatus `accepted`.

**File yang Terlibat:**
- Ubah: `internal/usecase/job_usecase.go`
- Ubah: `internal/repository/job_repository.go`
- Uji: `test/job_test.go`

**Langkah Pengerjaan:**
- [x] **Langkah 4.1:** Buat method `CountActiveApplications(db *gorm.DB, jobID string) (int64, error)` pada repository untuk menghitung lamaran dengan status `pending` atau `accepted`.
- [x] **Langkah 4.2:** Di `JobUseCase.Delete`, sebelum menghapus lowongan, periksa apakah terdapat lamaran aktif.
- [x] **Langkah 4.3:** Jika ada lamaran aktif, tolak penghapusan dengan status `400 Bad Request`: `"lowongan tidak dapat dihapus karena masih memiliki pelamar aktif atau diterima"`.
- [x] **Langkah 4.4:** Buat pengujian di `test/job_test.go` untuk memvalidasi lowongan berpelamar ditolak saat dihapus.

---

### Tugas 5: Validasi dan Sanitasi URL Profil Mahasiswa

**Tujuan:**
Mencegah injeksi payload script (`javascript:...`) atau skema URL berbahaya pada foto profil, CV, dan link portofolio mahasiswa yang ditampilkan kepada pihak mitra.

**File yang Terlibat:**
- Ubah: `internal/model/user_model.go`
- Ubah: `internal/usecase/user_usecase.go`
- Uji: `test/profile_update_test.go`

**Langkah Pengerjaan:**
- [x] **Langkah 5.1:** Perbarui tag validasi pada struct `UpdateMahasiswaProfileRequest`:
  ```go
  ProfilePicture string `json:"profile_picture" validate:"omitempty,http_url,max=500"`
  FotoProfile    string `json:"foto_profile" validate:"omitempty,http_url,max=500"`
  CV             string `json:"cv" validate:"omitempty,http_url,max=500"`
  PortfolioURL   string `json:"portfolio_url" validate:"omitempty,http_url,max=500"`
  LinkPortofolio string `json:"link_portofolio" validate:"omitempty,http_url,max=500"`
  ```
- [x] **Langkah 5.2:** Daftarkan custom validator atau rule `http_url` (memastikan skema diawali `http://` atau `https://` dan memiliki host valid).
- [x] **Langkah 5.3:** Buat unit test pada `test/profile_update_test.go` yang menguji penolakan input `javascript:alert(1)` dan menerima `https://example.com/cv.pdf`.

---

### Tugas 6: Sanitasi Error Handler HTTP 500 (CWE-209)

**Tujuan:**
Mencegah kebocoran struktur query internal, nama tabel, atau kredensial database ke respons JSON client saat terjadi kesalahan tak terduga di server.

**File yang Terlibat:**
- Ubah: `internal/config/fiber.go`
- Uji: `test/config_test.go`

**Langkah Pengerjaan:**
- [x] **Langkah 6.1:** Ubah `ErrorHandler` di `internal/config/fiber.go`:
  Jika status error bernilai `500 InternalServerError`, jangan gunakan string mentah `err.Error()`, melainkan ganti dengan pesan generik:
  ```go
  message := "terjadi kesalahan pada server"
  if code < 500 {
      message = err.Error()
  }
  ```
- [x] **Langkah 6.2:** Pastikan detail error asli tetap dicatat (*logged*) ke Logrus logger untuk keperluan debugging internal.
- [x] **Langkah 6.3:** Uji respons error 500 untuk memastikan tidak ada string SQL mentah yang keluar di response payload.

---

### Tugas 7: Paginasi pada Endpoint Daftar Pelamar Mitra

**Tujuan:**
Mencegah konsumsi memori berlebih (*memory spike / Out Of Memory DoS*) pada endpoint `GET /api/mitra/applications` saat suatu lowongan menerima ratusan atau ribuan pelamar.

**File yang Terlibat:**
- Ubah: `internal/model/application_model.go`
- Ubah: `internal/repository/application_repository.go`
- Ubah: `internal/usecase/application_usecase.go`
- Ubah: `internal/delivery/http/application_controller.go`
- Ubah: `internal/apidocs/openapi.json`
- Ubah: `OPENAPI.md`
- Uji: `test/application_integration_test.go`, `test/openapi_test.go`

**Langkah Pengerjaan:**
- [x] **Langkah 7.1:** Tambahkan parameter query `page` (default 1) dan `size` (default 10, max 100) pada controller `ListMitra`.
- [x] **Langkah 7.2:** Perbarui `ListByMitraID` di repository untuk menerima `page` dan `size`, serta menjalankan `COUNT` total pelamar untuk mengisi `model.PageMetadata`.
- [x] **Langkah 7.3:** Sesuaikan return type usecase dan respons controller agar menyertakan `Paging: pageMetadata`.
- [x] **Langkah 7.4:** Perbarui spesifikasi di `internal/apidocs/openapi.json` dan dokumentasi di `OPENAPI.md`.
- [x] **Langkah 7.5:** Jalankan `go test ./test -run '^TestOpenAPIDocumentation$'` untuk memverifikasi kecocokan skema.

---

### Tugas 8: Penyempurnaan Modul Cerita (Stories Edge Cases)

**Tujuan:**
Menangani edge case teknis pada manipulasi cerita diri: penanganan alias case-insensitive, pengurutan deterministik, dan pembaruan kolom terisolasi.

**File yang Terlibat:**
- Ubah: `internal/usecase/story_usecase.go`
- Ubah: `internal/repository/story_repository.go`
- Uji: `test/story_test.go`

**Langkah Pengerjaan:**
- [x] **Langkah 8.1:** Gunakan `strings.EqualFold(storyID, "me")` pada `StoryUseCase.Update` agar endpoint menerima variasi seperti `/api/stories/ME` dan `/api/stories/Me`.
- [x] **Langkah 8.2:** Ubah query pengurutan pada `FindLatestByUserID` di `StoryRepository` menjadi `ORDER BY created_at DESC, id DESC` agar selalu deterministik.
- [x] **Langkah 8.3:** Ganti pemanggilan `db.Save(story)` pada update cerita menjadi pembaruan terarah (`Select("content", "word_count", "analysis_status", "updated_at").Updates(...)`) agar tidak menulis ulang `created_at` atau `user_id`.
- [x] **Langkah 8.4:** Tambahkan unit test di `test/story_test.go` yang memverifikasi request `/api/stories/ME` berhasil dengan status 200.
