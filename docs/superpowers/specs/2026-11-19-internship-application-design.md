# Spesifikasi Desain: Fitur Pendaftaran Magang (Internship Application)

**Tanggal**: 2026-11-19  
**Status**: Disetujui (Approved)  

---

## 1. Latar Belakang & Tujuan
Sistem membutuhkan fitur di mana mahasiswa dapat melamar ke lowongan magang yang tersedia. Namun, terdapat dua aturan bisnis utama yang harus dipenuhi:
1. **Kelengkapan Profil**: Mahasiswa wajib melengkapi foto profil, CV, dan link portofolio sebelum dapat melamar ke lowongan manapun.
2. **Aturan Lamaran Tunggal Aktif**:
   - Mahasiswa hanya boleh memiliki satu lamaran aktif dalam satu waktu (`pending`). Selama belum ada keputusan (ditolak atau diterima), mahasiswa tidak dapat melamar ke lowongan lain.
   - Jika lamaran ditolak (`rejected`), mahasiswa diperbolehkan melamar ke lowongan lain.
   - Jika lamaran diterima (`accepted`), mahasiswa tidak dapat melamar ke lowongan manapun lagi.
3. **Manajemen Pelamar oleh Mitra**: Mitra pemilik lowongan dapat melihat daftar pelamar dan mengubah status lamaran (`accepted` / `rejected`).

---

## 2. Model Data & Arsitektur

### 2.1 Entitas `Application`
File: `internal/entity/application_entity.go`

```go
package entity

import "time"

const (
	ApplicationStatusPending  = "pending"
	ApplicationStatusAccepted = "accepted"
	ApplicationStatusRejected = "rejected"
)

func IsValidApplicationStatus(status string) bool {
	return status == ApplicationStatusPending || status == ApplicationStatusAccepted || status == ApplicationStatusRejected
}

type Application struct {
	ID          string    `gorm:"column:id;primaryKey"`
	JobID       string    `gorm:"column:job_id;not null;index"`
	Job         *Job      `gorm:"foreignKey:JobID;references:ID"`
	MahasiswaID string    `gorm:"column:mahasiswa_id;not null;index"`
	Mahasiswa   *User     `gorm:"foreignKey:MahasiswaID;references:ID"`
	Status      string    `gorm:"column:status;not null;default:'pending';index"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (a *Application) TableName() string {
	return "applications"
}
```

### 2.2 Model DTO & Request/Response
File: `internal/model/application_model.go`

```go
package model

import "time"

type ApplyJobRequest struct {
	JobID string `json:"job_id" validate:"required"`
}

type UpdateApplicationStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=accepted rejected"`
}

type MahasiswaApplicant struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	NIM            string `json:"nim,omitempty"`
	Email          string `json:"email"`
	ProfilePicture string `json:"profile_picture"`
	CV             string `json:"cv"`
	PortfolioURL   string `json:"portfolio_url"`
	Location       string `json:"location,omitempty"`
}

type ApplicationResponse struct {
	ID          string              `json:"id"`
	JobID       string              `json:"job_id"`
	JobTitle    string              `json:"job_title,omitempty"`
	Company     string              `json:"company,omitempty"`
	MahasiswaID string              `json:"mahasiswa_id"`
	Mahasiswa   *MahasiswaApplicant `json:"mahasiswa,omitempty"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}
```

### 2.3 Converter
File: `internal/model/converter/application_converter.go`
- `ApplicationToResponse(app *entity.Application) *model.ApplicationResponse`
- `ApplicationsToResponses(apps []entity.Application) []model.ApplicationResponse`

---

## 3. Komponen & Lapisan Logika

### 3.1 Repository
File: `internal/repository/application_repository.go`
- `Create(db *gorm.DB, app *entity.Application) error`
- `FindByID(db *gorm.DB, app *entity.Application, id string) error`
- `FindByMahasiswaIDAndStatus(db *gorm.DB, mahasiswaID string, status string) (*entity.Application, error)`
- `FindActiveByMahasiswaID(db *gorm.DB, mahasiswaID string) (*entity.Application, error)`: Mencari lamaran mahasiswa yang berstatus `pending` atau `accepted`.
- `ListByMahasiswaID(db *gorm.DB, mahasiswaID string) ([]entity.Application, error)`
- `ListByMitraID(db *gorm.DB, mitraID string, jobID string) ([]entity.Application, error)`
- `UpdateStatus(db *gorm.DB, id string, status string) error`

### 3.2 Use Case
File: `internal/usecase/application_usecase.go`
Metode:
1. `Apply(ctx context.Context, mahasiswaID string, jobID string) (*model.ApplicationResponse, error)`
   - Cek apakah user ada dan memiliki `Role == entity.RoleMahasiswa`.
   - Cek kelengkapan profil:
     - `strings.TrimSpace(user.ProfilePicture) == ""`
     - `strings.TrimSpace(user.CV) == ""`
     - `strings.TrimSpace(user.PortfolioURL) == ""`
     - Jika salah satu kosong: `fiber.NewError(fiber.StatusBadRequest, "Profil belum lengkap. Wajib melengkapi foto profil, CV, dan link portofolio sebelum mendaftar lowongan")`
   - Cek apakah mahasiswa sudah memiliki lamaran dengan status `accepted`:
     - Jika ada: `fiber.NewError(fiber.StatusBadRequest, "Anda telah diterima di lowongan magang dan tidak dapat mendaftar lagi")`
   - Cek apakah mahasiswa memiliki lamaran dengan status `pending`:
     - Jika ada: `fiber.NewError(fiber.StatusBadRequest, "Anda sudah memiliki pendaftaran magang yang sedang diproses. Harap menunggu hingga keputusan seleksi diberikan")`
   - Cek lowongan `jobID`:
     - Jika tidak ditemukan: `fiber.NewError(fiber.StatusNotFound, "Lowongan tidak ditemukan")`
   - Simpan entitas `Application` baru dengan status `ApplicationStatusPending`.
   - Kembalikan `model.ApplicationResponse`.

2. `ListByMahasiswa(ctx context.Context, mahasiswaID string) ([]model.ApplicationResponse, error)`
   - Mengambil seluruh riwayat lamaran mahasiswa, diurutkan dari yang terbaru (`created_at DESC`).

3. `ListByMitra(ctx context.Context, mitraID string, jobID string) ([]model.ApplicationResponse, error)`
   - Mengambil daftar pelamar untuk lowongan yang dimiliki oleh mitra tersebut.

4. `UpdateStatus(ctx context.Context, mitraID string, applicationID string, request *model.UpdateApplicationStatusRequest) (*model.ApplicationResponse, error)`
   - Validasi payload request (`status` harus `accepted` atau `rejected`).
   - Cari lamaran berdasarkan `applicationID`. Jika tidak ada: `404 Lamaran tidak ditemukan`.
   - Ambil data lowongan terkait. Pastikan `job.MitraID != nil && *job.MitraID == mitraID`. Jika tidak cocok: `403 Anda tidak memiliki izin untuk mengelola pelamar pada lowongan ini`.
   - Update status lamaran ke database.
   - Kembalikan `model.ApplicationResponse` terbaru.

### 3.3 HTTP Controller
File: `internal/delivery/http/application_controller.go`
- `Apply(ctx fiber.Ctx) error`: Menangani `POST /api/jobs/:id/apply` dan `POST /api/mahasiswa/applications`.
- `ListMahasiswa(ctx fiber.Ctx) error`: Menangani `GET /api/mahasiswa/applications`.
- `ListMitra(ctx fiber.Ctx) error`: Menangani `GET /api/mitra/applications`.
- `UpdateStatus(ctx fiber.Ctx) error`: Menangani `PATCH /api/mitra/applications/:id/status`.

### 3.4 Registrasi Rute & Bootstrap
File: `internal/delivery/http/route/route.go` & `internal/config/app.go`
- Tambahkan `&entity.Application{}` pada `config.DB.AutoMigrate`.
- Inisialisasi `applicationRepository`, `applicationUseCase`, dan `applicationController`.
- Daftarkan endpoint ke Fiber:
  - Mahasiswa (`AuthMiddleware` + `MahasiswaMiddleware`):
    - `POST /api/jobs/:id/apply`
    - `POST /api/mahasiswa/applications`
    - `GET /api/mahasiswa/applications`
  - Mitra (`AuthMiddleware` + `MitraMiddleware`):
    - `GET /api/mitra/applications`
    - `PATCH /api/mitra/applications/:id/status`

---

## 4. Rencana Pengujian di Folder `@test`
Semua pengujian unit test diletakkan di folder `test/`:

1. `test/application_converter_test.go`:
   - Pengujian fungsi konversi entitas ke DTO response.
2. `test/application_usecase_test.go`:
   - Kasus profil tidak lengkap (foto profil kosong, CV kosong, atau link portofolio kosong).
   - Kasus pendaftaran pertama kali berhasil (status `pending`).
   - Kasus penolakan pendaftaran ganda saat masih ada lamaran `pending`.
   - Kasus pendaftaran baru diizinkan setelah lamaran sebelumnya `rejected`.
   - Kasus penolakan pendaftaran permanen setelah lamaran `accepted`.
   - Kasus lowongan tidak valid / tidak ditemukan.
   - Kasus mitra berhasil update status (`accepted` dan `rejected`).
   - Kasus mitra tidak berhak (bukan pemilik lowongan) ditolak (403).
   - Kasus status update tidak valid ditolak (400).

---

## 5. Standar Kode
- Menghilangkan komentar-komentar yang berlebihan (*minimal comments*).
- Menjaga konsistensi struktur dan penamaan sesuai kode eksisting.
