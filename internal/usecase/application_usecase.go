package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/model/converter"
	"haji-koding/path-finder/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ApplicationUseCase struct {
	DB                    *gorm.DB
	Log                   *logrus.Logger
	Validate              *validator.Validate
	ApplicationRepository *repository.ApplicationRepository
	JobRepository         *repository.JobRepository
	UserRepository        *repository.UserRepository
}

func NewApplicationUseCase(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	applicationRepository *repository.ApplicationRepository,
	jobRepository *repository.JobRepository,
	userRepository *repository.UserRepository,
) *ApplicationUseCase {
	return &ApplicationUseCase{
		DB:                    db,
		Log:                   log,
		Validate:              validate,
		ApplicationRepository: applicationRepository,
		JobRepository:         jobRepository,
		UserRepository:        userRepository,
	}
}

func (c *ApplicationUseCase) Apply(ctx context.Context, mahasiswaID string, request *model.ApplyJobRequest) (*model.ApplicationResponse, error) {
	request.JobID = strings.TrimSpace(request.JobID)
	if err := c.Validate.Struct(request); err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	user := new(entity.User)
	if err := c.UserRepository.FindById(c.DB.WithContext(ctx), user, mahasiswaID); err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Pengguna tidak ditemukan")
	}

	if user.Role != entity.RoleMahasiswa {
		return nil, fiber.NewError(fiber.StatusForbidden, "Hanya mahasiswa yang dapat mendaftar magang")
	}

	if strings.TrimSpace(user.ProfilePicture) == "" || strings.TrimSpace(user.CV) == "" || strings.TrimSpace(user.PortfolioURL) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Profil belum lengkap. Wajib melengkapi foto profil, CV, dan link portofolio sebelum mendaftar lowongan")
	}

	job := new(entity.Job)
	if err := c.JobRepository.FindById(c.DB.WithContext(ctx), job, request.JobID); err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Lowongan tidak ditemukan")
	}

	if job.MitraID == nil || strings.TrimSpace(*job.MitraID) == "" {
		return nil, fiber.NewError(fiber.StatusBadRequest, "lowongan ini tidak menerima pendaftaran langsung melalui platform")
	}

	app := &entity.Application{
		ID:          uuid.NewString(),
		JobID:       job.ID,
		MahasiswaID: user.ID,
		Status:      entity.ApplicationStatusPending,
	}

	err := c.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		activeApp := new(entity.Application)
		err := c.ApplicationRepository.FindActiveByMahasiswaID(tx, activeApp, mahasiswaID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal memeriksa status lamaran")
		}
		if err == nil && activeApp.ID != "" {
			if activeApp.Status == entity.ApplicationStatusAccepted {
				return fiber.NewError(fiber.StatusBadRequest, "Anda telah diterima di lowongan magang dan tidak dapat mendaftar lagi")
			}
			if activeApp.Status == entity.ApplicationStatusPending {
				return fiber.NewError(fiber.StatusBadRequest, "Anda sudah memiliki pendaftaran magang yang sedang diproses. Harap menunggu hingga keputusan seleksi diberikan")
			}
		}

		if err := c.ApplicationRepository.Create(tx, app); err != nil {
			var pgErr *pgconn.PgError
			if (errors.As(err, &pgErr) && pgErr.Code == "23505") || strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "idx_unique_active_application") {
				return fiber.NewError(fiber.StatusBadRequest, "anda masih memiliki pendaftaran magang aktif yang sedang diproses")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "Gagal menyimpan pendaftaran magang")
		}

		return nil
	})

	if err != nil {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			return nil, fiberErr
		}
		var pgErr *pgconn.PgError
		if (errors.As(err, &pgErr) && pgErr.Code == "23505") || strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "idx_unique_active_application") {
			return nil, fiber.NewError(fiber.StatusBadRequest, "anda masih memiliki pendaftaran magang aktif yang sedang diproses")
		}
		return nil, err
	}

	app.Job = job
	app.Mahasiswa = user
	return converter.ApplicationToResponse(app), nil
}

func (c *ApplicationUseCase) ListByMahasiswa(ctx context.Context, mahasiswaID string) ([]model.ApplicationResponse, error) {
	var apps []entity.Application
	if err := c.ApplicationRepository.ListByMahasiswaID(c.DB.WithContext(ctx), &apps, mahasiswaID); err != nil {
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil daftar lamaran")
	}
	return converter.ApplicationsToResponses(apps), nil
}

func (c *ApplicationUseCase) ListByMitra(ctx context.Context, mitraID string, jobID string, page, size int) ([]model.ApplicationResponse, *model.PageMetadata, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	var apps []entity.Application
	paging, err := c.ApplicationRepository.ListByMitraID(c.DB.WithContext(ctx), &apps, mitraID, jobID, page, size)
	if err != nil {
		return nil, nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal mengambil daftar pelamar")
	}
	return converter.ApplicationsToResponses(apps), paging, nil
}

func (c *ApplicationUseCase) UpdateStatus(ctx context.Context, mitraID string, applicationID string, request *model.UpdateApplicationStatusRequest) (*model.ApplicationResponse, error) {
	if err := c.Validate.Struct(request); err != nil {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Status lamaran tidak valid, hanya boleh 'accepted' atau 'rejected'")
	}

	if !entity.IsValidApplicationStatus(request.Status) || request.Status == entity.ApplicationStatusPending {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Status lamaran tidak valid, hanya boleh 'accepted' atau 'rejected'")
	}

	app := new(entity.Application)
	if err := c.ApplicationRepository.FindByIDWithRelations(c.DB.WithContext(ctx), app, applicationID); err != nil {
		return nil, fiber.NewError(fiber.StatusNotFound, "Lamaran tidak ditemukan")
	}

	if app.Job == nil || app.Job.MitraID == nil || *app.Job.MitraID != mitraID {
		return nil, fiber.NewError(fiber.StatusForbidden, "Anda tidak memiliki izin untuk mengelola pelamar pada lowongan ini")
	}

	if app.Status != entity.ApplicationStatusPending {
		return nil, fiber.NewError(fiber.StatusBadRequest, "Status lamaran sudah final dan tidak dapat diubah lagi")
	}

	if err := c.ApplicationRepository.UpdateStatus(c.DB.WithContext(ctx), applicationID, request.Status); err != nil {
		if errors.Is(err, repository.ErrApplicationStatusNotPending) {
			return nil, fiber.NewError(fiber.StatusBadRequest, "Status lamaran sudah final dan tidak dapat diubah lagi")
		}
		return nil, fiber.NewError(fiber.StatusInternalServerError, "Gagal memperbarui status lamaran")
	}

	app.Status = request.Status
	app.UpdatedAt = time.Now()
	return converter.ApplicationToResponse(app), nil
}
