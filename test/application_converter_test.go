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

	t.Run("ApplicationToResponse with full relation", func(t *testing.T) {
		res := converter.ApplicationToResponse(app)
		assert.Equal(t, "app-1", res.ID)
		assert.Equal(t, "job-1", res.JobID)
		assert.Equal(t, "Backend Intern", res.JobTitle)
		assert.Equal(t, "Tech Corp", res.Company)
		assert.Equal(t, "mhs-1", res.MahasiswaID)
		assert.Equal(t, entity.ApplicationStatusPending, res.Status)
		assert.Equal(t, now, res.CreatedAt)
		assert.Equal(t, now, res.UpdatedAt)
		assert.NotNil(t, res.Mahasiswa)
		assert.Equal(t, "Budi", res.Mahasiswa.Name)
		assert.Equal(t, "https://example.com/p.png", res.Mahasiswa.ProfilePicture)
		assert.Equal(t, "https://example.com/cv.pdf", res.Mahasiswa.CV)
		assert.Equal(t, "https://portfolio.me", res.Mahasiswa.PortfolioURL)
	})

	t.Run("ApplicationToResponse with nil relation", func(t *testing.T) {
		appNil := &entity.Application{
			ID:          "app-2",
			JobID:       "job-2",
			MahasiswaID: "mhs-2",
			Status:      entity.ApplicationStatusAccepted,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		res := converter.ApplicationToResponse(appNil)
		assert.Equal(t, "app-2", res.ID)
		assert.Equal(t, "", res.JobTitle)
		assert.Equal(t, "", res.Company)
		assert.Nil(t, res.Mahasiswa)
	})

	t.Run("ApplicationsToResponses slice", func(t *testing.T) {
		apps := []entity.Application{*app}
		responses := converter.ApplicationsToResponses(apps)
		assert.Len(t, responses, 1)
		assert.Equal(t, "app-1", responses[0].ID)
	})
}
