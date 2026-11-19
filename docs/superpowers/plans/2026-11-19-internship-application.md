# Internship Application Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Membangun fitur pendaftaran magang mahasiswa ke lowongan yang tersedia dengan validasi kelengkapan profil (foto profil, CV, portofolio) serta aturan status pendaftaran (hanya satu lamaran aktif hingga ditolak, dan jika diterima tidak dapat mendaftar lagi).

**Architecture:** Clean Architecture bertingkat (Entity -> Model & Converter -> Repository -> UseCase -> Controller -> Route & Bootstrap) dengan pemisahan domain `Application` mandiri.

**Tech Stack:** Go 1.27, GORM (PostgreSQL/SQLite), Fiber v3, Logrus, Go-Playground Validator v10, Testify.

**Spec:** `docs/superpowers/specs/2026-11-19-internship-application-design.md`

## Global Constraints
- Kode harus minim komentar (kurangi komentar-komentar berlebihan).
- Mengikuti gaya penulisan dan konvensi Clean Architecture yang sudah ada di repositori.
- Seluruh unit test ditempatkan di folder `test/`.
- Validasi profil wajib: `ProfilePicture != ""`, `CV != ""`, `PortfolioURL != ""` (string tidak boleh kosong atau hanya spasi).
- Invarian status: `accepted` (tidak bisa melamar selamanya), `pending` (tidak bisa melamar ke tempat lain sampai ada keputusan), `rejected` (bisa melamar ke tempat lain).

## Review Focus
- Mahasiswa dengan spasi saja pada profil ("   ") dianggap belum melengkapi profil (harus ditolak).
- Mahasiswa yang mencoba melamar ke lowongan yang tidak ada mengembalikan 404 Not Found.
- Mahasiswa role non-mahasiswa (misal mitra/admin) yang mengakses endpoint apply ditolak oleh middleware/usecase.
- Mitra yang mencoba mengupdate status lamaran lowongan milik mitra lain ditolak 403 Forbidden.
- Update status hanya menerima nilai "accepted" atau "rejected" (nilai lain ditolak 400 Bad Request).

---

### Task 1: Entity, Model DTO, & Converter
**Files:**
- Create: `internal/entity/application_entity.go`
- Create: `internal/model/application_model.go`
- Create: `internal/model/converter/application_converter.go`
- Test: `test/application_converter_test.go`

**Interfaces:**
- Consumes: `entity.Job`, `entity.User`
- Produces: `entity.Application`, `model.ApplyJobRequest`, `model.UpdateApplicationStatusRequest`, `model.ApplicationResponse`, `converter.ApplicationToResponse`, `converter.ApplicationsToResponses`

- [ ] **Step 1: Tulis unit test converter yang gagal di `test/application_converter_test.go`**
```go
package test

import (
	"testing"
	"time"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model/converter"

	"github.com/stretchr/testify/assert"
)

func TestApplicationConverter(t *testing.T) {
	now := time.Now()
	app := &entity.Application{
		ID:          "app-1",
		JobID:       "job-1",
		MahasiswaID: "mhs-1",
		Status:      entity.ApplicationStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
		Job: &entity.Job{
			ID:      "job-1",
			Title:   "Backend Intern",
			Company: "Tech Corp",
		},
		Mahasiswa: &entity.User{
			ID:             "mhs-1",
			Name:           "Budi",
			Email:          "budi@example.com",
			ProfilePicture: "https://example.com/p.png",
			CV:             "https://example.com/cv.pdf",
			PortfolioURL:   "https://portfolio.me",
		},
	}

	res := converter.ApplicationToResponse(app)
	assert.Equal(t, "app-1", res.ID)
	assert.Equal(t, "job-1", res.JobID)
	assert.Equal(t, "Backend Intern", res.JobTitle)
	assert.Equal(t, "Tech Corp", res.Company)
	assert.Equal(t, "mhs-1", res.MahasiswaID)
	assert.Equal(t, entity.ApplicationStatusPending, res.Status)
	assert.NotNil(t, res.Mahasiswa)
	assert.Equal(t, "Budi", res.Mahasiswa.Name)
}
```

- [ ] **Step 2: Jalankan test untuk memverifikasi kegagalan kompilasi**
Run: `go test -v ./test/application_converter_test.go`
Expected: FAIL (undefined: entity.Application)

- [ ] **Step 3: Implementasikan entity, model, dan converter**
- Buat `internal/entity/application_entity.go` dengan konstanta status (`pending`, `accepted`, `rejected`) dan struct `Application`.
- Buat `internal/model/application_model.go` dengan `ApplyJobRequest`, `UpdateApplicationStatusRequest`, `MahasiswaApplicant`, dan `ApplicationResponse`.
- Buat `internal/model/converter/application_converter.go` dengan fungsi konversi `ApplicationToResponse` dan `ApplicationsToResponses`.

- [ ] **Step 4: Jalankan test dan pastikan lolos**
Run: `go test -v ./test/application_converter_test.go`
Expected: PASS

- [ ] **Step 5: Commit perubahan**
```bash
git add internal/entity/application_entity.go internal/model/application_model.go internal/model/converter/application_converter.go test/application_converter_test.go
git commit -m "feat: add application entity model and converter with tests"
```

---

### Task 2: Application Repository
**Files:**
- Create: `internal/repository/application_repository.go`
- Modify: `internal/repository/repository.go` (jika ada interface agregator)

**Interfaces:**
- Consumes: `entity.Application`, `*gorm.DB`, `*logrus.Logger`
- Produces:
  - `NewApplicationRepository(log *logrus.Logger) *ApplicationRepository`
  - `Create(db *gorm.DB, app *entity.Application) error`
  - `FindByID(db *gorm.DB, id string) (*entity.Application, error)`
  - `FindActiveByMahasiswaID(db *gorm.DB, mahasiswaID string) (*entity.Application, error)`
  - `ListByMahasiswaID(db *gorm.DB, mahasiswaID string) ([]entity.Application, error)`
  - `ListByMitraID(db *gorm.DB, mitraID string, jobID string) ([]entity.Application, error)`
  - `UpdateStatus(db *gorm.DB, id string, status string) error`

- [ ] **Step 1: Implementasikan `ApplicationRepository` di `internal/repository/application_repository.go`**
Mengikuti pola repository di `internal/repository/job_repository.go` dengan method:
- `Create`: menyimpan record baru ke tabel `applications`.
- `FindByID`: mencari berdasarkan primary key dengan Preload `Job` dan `Mahasiswa`.
- `FindActiveByMahasiswaID`: mencari record mahasiswa dengan status `pending` atau `accepted` (diurutkan created_at desc, limit 1).
- `ListByMahasiswaID`: mengambil semua lamaran mahasiswa dengan Preload `Job`.
- `ListByMitraID`: mengambil semua lamaran pada lowongan yang dimiliki oleh mitra (join `jobs` on `jobs.id = applications.job_id` where `jobs.mitra_id = ?`, optional filter `job_id`).
- `UpdateStatus`: memperbarui kolom `status` dan `updated_at`.

- [ ] **Step 2: Verifikasi build package repository**
Run: `go build ./internal/repository/...`
Expected: SUCCESS

- [ ] **Step 3: Commit perubahan**
```bash
git add internal/repository/application_repository.go
git commit -m "feat: implement application repository"
```

---

### Task 3: Application UseCase & Business Logic (dengan Unit Test Komprehensif)
**Files:**
- Create: `internal/usecase/application_usecase.go`
- Create: `test/application_usecase_test.go`

**Interfaces:**
- Consumes: `*gorm.DB`, `*logrus.Logger`, `*validator.Validate`, `*repository.ApplicationRepository`, `*repository.JobRepository`, `*repository.UserRepository`
- Produces:
  - `NewApplicationUseCase(...) *ApplicationUseCase`
  - `Apply(ctx context.Context, mahasiswaID string, request *model.ApplyJobRequest) (*model.ApplicationResponse, error)`
  - `ListByMahasiswa(ctx context.Context, mahasiswaID string) ([]model.ApplicationResponse, error)`
  - `ListByMitra(ctx context.Context, mitraID string, jobID string) ([]model.ApplicationResponse, error)`
  - `UpdateStatus(ctx context.Context, mitraID string, applicationID string, request *model.UpdateApplicationStatusRequest) (*model.ApplicationResponse, error)`

- [ ] **Step 1: Tulis unit test usecase di `test/application_usecase_test.go`**
Menguji secara menyeluruh:
1. Mahasiswa dengan profil belum lengkap (foto/CV/portofolio kosong atau spasi) ditolak 400.
2. Mahasiswa dengan profil lengkap berhasil apply jika belum ada lamaran aktif (status `pending`).
3. Mahasiswa yang memiliki lamaran `pending` ditolak saat apply ke lowongan lain (400).
4. Mahasiswa yang lamarannya telah `rejected` diizinkan apply ke lowongan lain (status jadi `pending`).
5. Mahasiswa yang lamarannya telah `accepted` ditolak permanen untuk mendaftar lagi (400).
6. Mitra pemilik lowongan berhasil update status ke `accepted` atau `rejected`.
7. Mitra bukan pemilik lowongan ditolak 403 Forbidden.
8. Status selain `accepted`/`rejected` ditolak 400 Bad Request.

- [ ] **Step 2: Jalankan test usecase dan verifikasi gagal (RED)**
Run: `go test -v ./test/application_usecase_test.go`
Expected: FAIL (undefined: usecase.NewApplicationUseCase)

- [ ] **Step 3: Implementasikan `ApplicationUseCase` di `internal/usecase/application_usecase.go` (GREEN)**
Terapkan logika bisnis:
- Validasi data user profil (`ProfilePicture`, `CV`, `PortfolioURL`).
- Validasi status lamaran mahasiswa (cek `FindActiveByMahasiswaID`):
  - Jika ada dan `status == "accepted"`: return `fiber.NewError(fiber.StatusBadRequest, "Anda telah diterima di lowongan magang dan tidak dapat mendaftar lagi")`.
  - Jika ada dan `status == "pending"`: return `fiber.NewError(fiber.StatusBadRequest, "Anda sudah memiliki pendaftaran magang yang sedang diproses. Harap menunggu hingga keputusan seleksi diberikan")`.
- Cek ketersediaan lowongan via `jobRepository`.
- Simpan lamaran baru dengan UUID dan status `pending`.
- `UpdateStatus`: verifikasi status enum, kepemilikan mitra terhadap lowongan, update status di repository.

- [ ] **Step 4: Jalankan test usecase dan pastikan seluruh skenario lulus (PASS)**
Run: `go test -v ./test/application_usecase_test.go`
Expected: PASS untuk semua subtest.

- [ ] **Step 5: Commit perubahan**
```bash
git add internal/usecase/application_usecase.go test/application_usecase_test.go
git commit -m "feat: implement application usecase with comprehensive business logic tests"
```

---

### Task 4: HTTP Controller, Routing, & Bootstrap
**Files:**
- Create: `internal/delivery/http/application_controller.go`
- Modify: `internal/delivery/http/route/route.go`
- Modify: `internal/config/app.go`

**Interfaces:**
- Consumes: `*usecase.ApplicationUseCase`, `*logrus.Logger`
- Produces: `*http.ApplicationController`, route registrations

- [ ] **Step 1: Implementasikan `ApplicationController` di `internal/delivery/http/application_controller.go`**
- Method `Apply(ctx fiber.Ctx)`:
  - Ambil `user_id` dari `ctx.Locals("user_id")`.
  - Ambil `job_id` dari param URL (`:id`) atau body JSON jika dikirim via body.
  - Panggil `useCase.Apply`.
  - Return HTTP 201 Created dengan `model.WebResponse`.
- Method `ListMahasiswa(ctx fiber.Ctx)`:
  - Ambil `user_id` dari `ctx.Locals("user_id")`.
  - Panggil `useCase.ListByMahasiswa`.
  - Return HTTP 200 OK dengan `model.WebResponse`.
- Method `ListMitra(ctx fiber.Ctx)`:
  - Ambil `mitra_id` dari `ctx.Locals("user_id")`.
  - Query param `job_id` opsional.
  - Panggil `useCase.ListByMitra`.
  - Return HTTP 200 OK dengan `model.WebResponse`.
- Method `UpdateStatus(ctx fiber.Ctx)`:
  - Ambil `mitra_id` dari `ctx.Locals("user_id")`.
  - Param `id` dari URL.
  - Body `UpdateApplicationStatusRequest`.
  - Panggil `useCase.UpdateStatus`.
  - Return HTTP 200 OK dengan `model.WebResponse`.

- [ ] **Step 2: Update `internal/delivery/http/route/route.go`**
- Tambahkan field `ApplicationController *http.ApplicationController` di struct `RouteConfig`.
- Di `SetupAuthRoute()`:
  - Rute Mahasiswa (`AuthMiddleware` + `MahasiswaMiddleware`):
    - `c.App.Post("/api/jobs/:id/apply", c.AuthMiddleware, c.MahasiswaMiddleware, c.ApplicationController.Apply)`
    - `c.App.Post("/api/mahasiswa/applications", c.AuthMiddleware, c.MahasiswaMiddleware, c.ApplicationController.Apply)`
    - `c.App.Get("/api/mahasiswa/applications", c.AuthMiddleware, c.MahasiswaMiddleware, c.ApplicationController.ListMahasiswa)`
  - Rute Mitra (`AuthMiddleware` + `MitraMiddleware`):
    - `c.App.Get("/api/mitra/applications", c.AuthMiddleware, c.MitraMiddleware, c.ApplicationController.ListMitra)`
    - `c.App.Patch("/api/mitra/applications/:id/status", c.AuthMiddleware, c.MitraMiddleware, c.ApplicationController.UpdateStatus)`
    - `c.App.Put("/api/mitra/applications/:id/status", c.AuthMiddleware, c.MitraMiddleware, c.ApplicationController.UpdateStatus)`

- [ ] **Step 3: Update `internal/config/app.go`**
- Tambahkan `&entity.Application{}` ke `config.DB.AutoMigrate(...)`.
- Inisialisasi `applicationRepository := repository.NewApplicationRepository(config.Log)`.
- Inisialisasi `applicationUseCase := usecase.NewApplicationUseCase(config.DB, config.Log, config.Validate, applicationRepository, jobRepository, userRepository)`.
- Inisialisasi `applicationController := http.NewApplicationController(applicationUseCase, config.Log)`.
- Masukkan `ApplicationController: applicationController` ke `route.RouteConfig`.

- [ ] **Step 4: Verifikasi build seluruh project**
Run: `go build ./cmd/web/...`
Expected: SUCCESS

- [ ] **Step 5: Commit perubahan**
```bash
git add internal/delivery/http/application_controller.go internal/delivery/http/route/route.go internal/config/app.go
git commit -m "feat: wire application controller, routes, and bootstrap"
```

---

### Task 5: End-to-End HTTP Integration Test di `@test`
**Files:**
- Create: `test/application_integration_test.go`

**Interfaces:**
- Consumes: Fiber `app.Test`, JWT token generator, Database
- Produces: Verifikasi E2E HTTP endpoints

- [ ] **Step 1: Tulis HTTP test komprehensif di `test/application_integration_test.go`**
- Skenario E2E:
  1. Mahasiswa profil tidak lengkap coba POST `/api/jobs/:id/apply` -> HTTP 400 dengan pesan profil belum lengkap.
  2. Mahasiswa lengkapi profil (foto, CV, portofolio) lalu POST `/api/jobs/:id/apply` -> HTTP 201 Created.
  3. Mahasiswa coba daftar ke lowongan lain saat masih pending -> HTTP 400 ("sedang diproses").
  4. Mitra login, lihat daftar pelamar via GET `/api/mitra/applications` -> data pelamar muncul.
  5. Mitra lain coba update status lamaran tersebut -> HTTP 403 Forbidden.
  6. Mitra pemilik tolak lamaran (PATCH `/api/mitra/applications/:id/status` dengan `status: rejected`) -> HTTP 200 OK.
  7. Mahasiswa yang sudah ditolak coba daftar ke lowongan lain -> HTTP 201 Created (berhasil).
  8. Mitra terima lamaran baru tersebut (`status: accepted`) -> HTTP 200 OK.
  9. Mahasiswa yang sudah diterima coba daftar lagi ke lowongan manapun -> HTTP 400 ("telah diterima di lowongan magang").

- [ ] **Step 2: Jalankan integration test dan verifikasi semua skenario lolos**
Run: `go test -v ./test/application_integration_test.go`
Expected: PASS

- [ ] **Step 3: Jalankan seluruh test suite di `@test`**
Run: `go test -v ./test/...`
Expected: PASS

- [ ] **Step 4: Commit perubahan**
```bash
git add test/application_integration_test.go
git commit -m "test: add comprehensive E2E integration test for internship applications"
```
