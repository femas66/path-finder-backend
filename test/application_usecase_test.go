package test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/repository"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplicationUseCase(t *testing.T) {
	viperConfig := config.NewViper()
	log := config.NewLogger()
	db := config.NewDatabase(viperConfig, log)
	validate := config.NewValidator()

	require.NoError(t, db.AutoMigrate(&entity.User{}, &entity.Job{}, &entity.Application{}))
	require.NoError(t, db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_active_application 
		ON applications (mahasiswa_id) 
		WHERE status IN ('pending', 'accepted');
	`).Error)

	cleanDB := func() {
		db.Exec("DELETE FROM applications")
		db.Exec("DELETE FROM jobs")
		db.Exec("DELETE FROM users")
	}
	cleanDB()
	t.Cleanup(cleanDB)

	userRepo := repository.NewUserRepository(log)
	jobRepo := repository.NewJobRepository(log)
	appRepo := repository.NewApplicationRepository(log)

	appUseCase := usecase.NewApplicationUseCase(db, log, validate, appRepo, jobRepo, userRepo)

	mitraID := "mitra-" + uuid.NewString()
	otherMitraID := "other-mitra-" + uuid.NewString()

	require.NoError(t, db.Create(&entity.User{
		ID:       mitraID,
		Name:     "PT Maju Bersama",
		Email:    "mitra@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleMitra,
	}).Error)

	require.NoError(t, db.Create(&entity.User{
		ID:       otherMitraID,
		Name:     "PT Lain",
		Email:    "othermitra@example.com",
		Password: "hashedpassword",
		Role:     entity.RoleMitra,
	}).Error)

	job1 := &entity.Job{
		ID:       "job-" + uuid.NewString(),
		MitraID:  &mitraID,
		Title:    "Go Backend Intern",
		Category: "Backend Developer Intern",
		Company:  "PT Maju Bersama",
	}
	require.NoError(t, db.Create(job1).Error)

	job2 := &entity.Job{
		ID:       "job-" + uuid.NewString(),
		MitraID:  &mitraID,
		Title:    "Frontend Intern",
		Category: "Frontend Developer Intern",
		Company:  "PT Maju Bersama",
	}
	require.NoError(t, db.Create(job2).Error)

	t.Run("Apply gagal jika profil mahasiswa belum lengkap", func(t *testing.T) {
		mhsIncomplete := &entity.User{
			ID:             "mhs-inc-" + uuid.NewString(),
			Name:           "Budi Incomplete",
			Email:          "budi.inc@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "",
			CV:             "https://example.com/cv.pdf",
			PortfolioURL:   "https://portfolio.com",
		}
		require.NoError(t, db.Create(mhsIncomplete).Error)

		_, err := appUseCase.Apply(context.Background(), mhsIncomplete.ID, &model.ApplyJobRequest{JobID: job1.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Profil belum lengkap")

		mhsIncomplete.ProfilePicture = "https://example.com/pic.png"
		mhsIncomplete.CV = "   "
		db.Save(mhsIncomplete)
		_, err = appUseCase.Apply(context.Background(), mhsIncomplete.ID, &model.ApplyJobRequest{JobID: job1.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Profil belum lengkap")

		mhsIncomplete.CV = "https://example.com/cv.pdf"
		mhsIncomplete.PortfolioURL = ""
		db.Save(mhsIncomplete)
		_, err = appUseCase.Apply(context.Background(), mhsIncomplete.ID, &model.ApplyJobRequest{JobID: job1.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Profil belum lengkap")
	})

	t.Run("Apply berhasil jika profil lengkap dan belum ada lamaran aktif", func(t *testing.T) {
		mhs := &entity.User{
			ID:             "mhs-ok-" + uuid.NewString(),
			Name:           "Siti Mahasiswa",
			Email:          "siti@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/siti.png",
			CV:             "https://example.com/siti-cv.pdf",
			PortfolioURL:   "https://siti.dev",
		}
		require.NoError(t, db.Create(mhs).Error)

		resp, err := appUseCase.Apply(context.Background(), mhs.ID, &model.ApplyJobRequest{JobID: job1.ID})
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.ID)
		assert.Equal(t, entity.ApplicationStatusPending, resp.Status)
		assert.Equal(t, job1.ID, resp.JobID)
		assert.Equal(t, mhs.ID, resp.MahasiswaID)
	})

	t.Run("Apply gagal ke lowongan lain jika masih memiliki lamaran pending", func(t *testing.T) {
		mhsPending := &entity.User{
			ID:             "mhs-pnd-" + uuid.NewString(),
			Name:           "Rian Pending",
			Email:          "rian@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/rian.png",
			CV:             "https://example.com/rian-cv.pdf",
			PortfolioURL:   "https://rian.dev",
		}
		require.NoError(t, db.Create(mhsPending).Error)

		_, err := appUseCase.Apply(context.Background(), mhsPending.ID, &model.ApplyJobRequest{JobID: job1.ID})
		require.NoError(t, err)

		_, err = appUseCase.Apply(context.Background(), mhsPending.ID, &model.ApplyJobRequest{JobID: job2.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sedang diproses")
	})

	t.Run("Apply berhasil ke lowongan lain jika lamaran sebelumnya ditolak", func(t *testing.T) {
		mhsRej := &entity.User{
			ID:             "mhs-rej-" + uuid.NewString(),
			Name:           "Dewi Ditolak",
			Email:          "dewi@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/dewi.png",
			CV:             "https://example.com/dewi-cv.pdf",
			PortfolioURL:   "https://dewi.dev",
		}
		require.NoError(t, db.Create(mhsRej).Error)

		appResp, err := appUseCase.Apply(context.Background(), mhsRej.ID, &model.ApplyJobRequest{JobID: job1.ID})
		require.NoError(t, err)

		_, err = appUseCase.UpdateStatus(context.Background(), mitraID, appResp.ID, &model.UpdateApplicationStatusRequest{
			Status: entity.ApplicationStatusRejected,
		})
		require.NoError(t, err)

		newApp, err := appUseCase.Apply(context.Background(), mhsRej.ID, &model.ApplyJobRequest{JobID: job2.ID})
		assert.NoError(t, err)
		assert.Equal(t, entity.ApplicationStatusPending, newApp.Status)
		assert.Equal(t, job2.ID, newApp.JobID)
	})

	t.Run("Apply gagal jika mahasiswa telah diterima di salah satu lowongan", func(t *testing.T) {
		mhsAcc := &entity.User{
			ID:             "mhs-acc-" + uuid.NewString(),
			Name:           "Andi Diterima",
			Email:          "andi@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/andi.png",
			CV:             "https://example.com/andi-cv.pdf",
			PortfolioURL:   "https://andi.dev",
		}
		require.NoError(t, db.Create(mhsAcc).Error)

		appResp, err := appUseCase.Apply(context.Background(), mhsAcc.ID, &model.ApplyJobRequest{JobID: job1.ID})
		require.NoError(t, err)

		_, err = appUseCase.UpdateStatus(context.Background(), mitraID, appResp.ID, &model.UpdateApplicationStatusRequest{
			Status: entity.ApplicationStatusAccepted,
		})
		require.NoError(t, err)

		_, err = appUseCase.Apply(context.Background(), mhsAcc.ID, &model.ApplyJobRequest{JobID: job2.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "telah diterima")
	})

	t.Run("Mitra bukan pemilik lowongan dilarang mengubah status pelamar", func(t *testing.T) {
		mhsTest := &entity.User{
			ID:             "mhs-tst-" + uuid.NewString(),
			Name:           "Test Mhs",
			Email:          "testmhs@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/t.png",
			CV:             "https://example.com/t.pdf",
			PortfolioURL:   "https://t.dev",
		}
		require.NoError(t, db.Create(mhsTest).Error)

		appResp, err := appUseCase.Apply(context.Background(), mhsTest.ID, &model.ApplyJobRequest{JobID: job1.ID})
		require.NoError(t, err)

		_, err = appUseCase.UpdateStatus(context.Background(), otherMitraID, appResp.ID, &model.UpdateApplicationStatusRequest{
			Status: entity.ApplicationStatusAccepted,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "izin")
	})

	t.Run("UpdateStatus gagal jika status tidak valid", func(t *testing.T) {
		mhsTest := &entity.User{
			ID:             "mhs-inv-" + uuid.NewString(),
			Name:           "Inv Mhs",
			Email:          "invmhs@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/i.png",
			CV:             "https://example.com/i.pdf",
			PortfolioURL:   "https://i.dev",
		}
		require.NoError(t, db.Create(mhsTest).Error)

		appResp, err := appUseCase.Apply(context.Background(), mhsTest.ID, &model.ApplyJobRequest{JobID: job1.ID})
		require.NoError(t, err)

		_, err = appUseCase.UpdateStatus(context.Background(), mitraID, appResp.ID, &model.UpdateApplicationStatusRequest{
			Status: "invalid_status",
		})
		assert.Error(t, err)
	})

	t.Run("Apply gagal jika lowongan tidak ditemukan (404)", func(t *testing.T) {
		mhsValid := &entity.User{
			ID:             "mhs-404-" + uuid.NewString(),
			Name:           "Valid Mhs",
			Email:          "valid404@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/v.png",
			CV:             "https://example.com/v.pdf",
			PortfolioURL:   "https://v.dev",
		}
		require.NoError(t, db.Create(mhsValid).Error)

		_, err := appUseCase.Apply(context.Background(), mhsValid.ID, &model.ApplyJobRequest{JobID: "non-existent-job-id"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tidak ditemukan")
	})

	t.Run("Apply gagal jika user bukan role mahasiswa (403)", func(t *testing.T) {
		_, err := appUseCase.Apply(context.Background(), mitraID, &model.ApplyJobRequest{JobID: job1.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Hanya mahasiswa")
	})

	t.Run("UpdateStatus gagal jika status sudah final (bukan pending)", func(t *testing.T) {
		mhsFinal := &entity.User{
			ID:             "mhs-fnl-" + uuid.NewString(),
			Name:           "Final Mhs",
			Email:          "final@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/f.png",
			CV:             "https://example.com/f.pdf",
			PortfolioURL:   "https://f.dev",
		}
		require.NoError(t, db.Create(mhsFinal).Error)

		appResp, err := appUseCase.Apply(context.Background(), mhsFinal.ID, &model.ApplyJobRequest{JobID: job1.ID})
		require.NoError(t, err)

		_, err = appUseCase.UpdateStatus(context.Background(), mitraID, appResp.ID, &model.UpdateApplicationStatusRequest{
			Status: entity.ApplicationStatusAccepted,
		})
		require.NoError(t, err)

		_, err = appUseCase.UpdateStatus(context.Background(), mitraID, appResp.ID, &model.UpdateApplicationStatusRequest{
			Status: entity.ApplicationStatusRejected,
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "final")
	})

	t.Run("Apply gagal ke lowongan seed tanpa mitra (400)", func(t *testing.T) {
		seedJob := &entity.Job{
			ID:       "seed-job-" + uuid.NewString(),
			MitraID:  nil,
			Title:    "Catalog Seed Job",
			Category: "Backend Developer Intern",
			Company:  "Catalog Company",
		}
		require.NoError(t, db.Create(seedJob).Error)

		mhs := &entity.User{
			ID:             "mhs-seed-" + uuid.NewString(),
			Name:           "Seed Mhs",
			Email:          "seedmhs@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/s.png",
			CV:             "https://example.com/s.pdf",
			PortfolioURL:   "https://s.dev",
		}
		require.NoError(t, db.Create(mhs).Error)

		_, err := appUseCase.Apply(context.Background(), mhs.ID, &model.ApplyJobRequest{JobID: seedJob.ID})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tidak menerima pendaftaran langsung")
	})

	t.Run("Proteksi Concurrency TOCTOU - 5 goroutine Apply paralel hanya 1 yang berhasil", func(t *testing.T) {
		mhsConcurrent := &entity.User{
			ID:             "mhs-cnc-" + uuid.NewString(),
			Name:           "Concurrency Mhs",
			Email:          "cnc@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/c.png",
			CV:             "https://example.com/c.pdf",
			PortfolioURL:   "https://c.dev",
		}
		require.NoError(t, db.Create(mhsConcurrent).Error)

		var testJobs []*entity.Job
		for i := 0; i < 5; i++ {
			j := &entity.Job{
				ID:       fmt.Sprintf("job-cnc-%d-%s", i, uuid.NewString()),
				MitraID:  &mitraID,
				Title:    fmt.Sprintf("Concurrent Job %d", i),
				Category: "Backend Developer Intern",
				Company:  "PT Maju Bersama",
			}
			require.NoError(t, db.Create(j).Error)
			testJobs = append(testJobs, j)
		}

		type result struct {
			resp *model.ApplicationResponse
			err  error
		}
		results := make(chan result, 5)

		var wg sync.WaitGroup
		wg.Add(5)
		for i := 0; i < 5; i++ {
			jobID := testJobs[i].ID
			go func() {
				defer wg.Done()
				r, err := appUseCase.Apply(context.Background(), mhsConcurrent.ID, &model.ApplyJobRequest{JobID: jobID})
				results <- result{resp: r, err: err}
			}()
		}
		wg.Wait()
		close(results)

		var successCount int
		var failCount int
		for res := range results {
			if res.err == nil {
				successCount++
				assert.NotEmpty(t, res.resp.ID)
			} else {
				failCount++
				var fiberErr *fiber.Error
				if errors.As(res.err, &fiberErr) {
					assert.Equal(t, fiber.StatusBadRequest, fiberErr.Code)
				}
			}
		}

		assert.Equal(t, 1, successCount, "Tepat 1 pendaftaran yang harus berhasil")
		assert.Equal(t, 4, failCount, "4 pendaftaran lainnya harus ditolak")
	})

	t.Run("Proteksi Atomic Status Transition - pembaruan status bersamaan hanya 1 yang berhasil", func(t *testing.T) {
		mhsAtomic := &entity.User{
			ID:             "mhs-atm-" + uuid.NewString(),
			Name:           "Atomic Mhs",
			Email:          "atm@example.com",
			Password:       "hashedpassword",
			Role:           entity.RoleMahasiswa,
			ProfilePicture: "https://example.com/a.png",
			CV:             "https://example.com/a.pdf",
			PortfolioURL:   "https://a.dev",
		}
		require.NoError(t, db.Create(mhsAtomic).Error)

		appResp, err := appUseCase.Apply(context.Background(), mhsAtomic.ID, &model.ApplyJobRequest{JobID: job1.ID})
		require.NoError(t, err)

		type updateResult struct {
			status string
			err    error
		}
		results := make(chan updateResult, 2)
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			_, err := appUseCase.UpdateStatus(context.Background(), mitraID, appResp.ID, &model.UpdateApplicationStatusRequest{
				Status: entity.ApplicationStatusAccepted,
			})
			results <- updateResult{status: entity.ApplicationStatusAccepted, err: err}
		}()

		go func() {
			defer wg.Done()
			_, err := appUseCase.UpdateStatus(context.Background(), mitraID, appResp.ID, &model.UpdateApplicationStatusRequest{
				Status: entity.ApplicationStatusRejected,
			})
			results <- updateResult{status: entity.ApplicationStatusRejected, err: err}
		}()

		wg.Wait()
		close(results)

		var successCount int
		var failCount int
		for res := range results {
			if res.err == nil {
				successCount++
			} else {
				failCount++
				assert.Contains(t, res.err.Error(), "final")
			}
		}

		assert.Equal(t, 1, successCount, "Tepat 1 pembaruan status yang berhasil")
		assert.Equal(t, 1, failCount, "Pembaruan kedua harus ditolak karena status sudah final")
	})
}
