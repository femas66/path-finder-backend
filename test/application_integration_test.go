package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestApplicationIntegration(t *testing.T) {
	viperConfig := config.NewViper()
	log := config.NewLogger()
	db := config.NewDatabase(viperConfig, log)
	validate := config.NewValidator()
	app := config.NewFiber(viperConfig)

	config.Bootstrap(&config.BootstrapConfig{
		DB:       db,
		App:      app,
		Log:      log,
		Validate: validate,
		Config:   viperConfig,
	})

	db.Exec("DELETE FROM applications")
	db.Exec("DELETE FROM jobs")
	db.Exec("DELETE FROM users")

	createAndLogin := func(id, name, email, password, role string, profilePic, cv, portfolio string) string {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		u := &entity.User{
			ID:             id,
			Name:           name,
			Email:          email,
			Password:       string(hashed),
			Role:           role,
			ProfilePicture: profilePic,
			CV:             cv,
			PortfolioURL:   portfolio,
		}
		require.NoError(t, db.Create(u).Error)

		loginBody := model.LoginUserRequest{
			Email:    email,
			Password: password,
		}
		loginBytes, _ := json.Marshal(loginBody)
		req := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(loginBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.LoginResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		return webResp.Data.Token
	}

	mitra1ID := "mitra-e2e-1-" + uuid.NewString()
	mitra2ID := "mitra-e2e-2-" + uuid.NewString()
	mhsIncompleteID := "mhs-e2e-inc-" + uuid.NewString()
	mhsCompleteID := "mhs-e2e-ok-" + uuid.NewString()
	mhsFreshID := "mhs-e2e-fresh-" + uuid.NewString()

	mitra1Token := createAndLogin(mitra1ID, "PT Solusi Cloud", "cloud@mitra.com", "pass123", entity.RoleMitra, "https://cloud.com/logo.png", "", "")
	mitra2Token := createAndLogin(mitra2ID, "PT Inovasi AI", "ai@mitra.com", "pass123", entity.RoleMitra, "https://ai.com/logo.png", "", "")
	mhsIncompleteToken := createAndLogin(mhsIncompleteID, "Dodi Belum Lengkap", "dodi@mhs.ac.id", "pass123", entity.RoleMahasiswa, "", "", "")
	mhsCompleteToken := createAndLogin(mhsCompleteID, "Maya Lengkap", "maya@mhs.ac.id", "pass123", entity.RoleMahasiswa, "https://pic.me/maya.png", "https://cv.me/maya.pdf", "https://portfolio.me/maya")
	mhsFreshToken := createAndLogin(mhsFreshID, "Fresh Mahasiswa", "fresh@mhs.ac.id", "pass123", entity.RoleMahasiswa, "https://pic.me/f.png", "https://cv.me/f.pdf", "https://portfolio.me/f")

	job1 := &entity.Job{
		ID:       "job-e2e-1-" + uuid.NewString(),
		MitraID:  &mitra1ID,
		Title:    "Golang Backend Intern",
		Category: "Backend Developer Intern",
		Company:  "PT Solusi Cloud",
	}
	require.NoError(t, db.Create(job1).Error)

	job2 := &entity.Job{
		ID:       "job-e2e-2-" + uuid.NewString(),
		MitraID:  &mitra1ID,
		Title:    "DevOps Intern",
		Category: "Network & IT Support Intern",
		Company:  "PT Solusi Cloud",
	}
	require.NoError(t, db.Create(job2).Error)

	var createdAppID string

	t.Run("Mahasiswa dengan profil belum lengkap gagal melamar (400)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/jobs/%s/apply", job1.ID), nil)
		req.Header.Set("Authorization", "Bearer "+mhsIncompleteToken)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		_ = json.Unmarshal(respBytes, &webResp)
		assert.False(t, webResp.Success)
		assert.Contains(t, webResp.Message, "Profil belum lengkap")
	})

	t.Run("Mahasiswa dengan profil lengkap berhasil melamar (201)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/jobs/%s/apply", job1.ID), nil)
		req.Header.Set("Authorization", "Bearer "+mhsCompleteToken)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.ApplicationResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))
		assert.True(t, webResp.Success)
		assert.NotEmpty(t, webResp.Data.ID)
		assert.Equal(t, entity.ApplicationStatusPending, webResp.Data.Status)
		assert.Equal(t, job1.ID, webResp.Data.JobID)
		assert.Equal(t, mhsCompleteID, webResp.Data.MahasiswaID)

		createdAppID = webResp.Data.ID
	})

	t.Run("Mahasiswa mencoba melamar ke lowongan lain saat masih pending ditolak (400)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/jobs/%s/apply", job2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+mhsCompleteToken)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		_ = json.Unmarshal(respBytes, &webResp)
		assert.False(t, webResp.Success)
		assert.Contains(t, webResp.Message, "sedang diproses")
	})

	t.Run("Mahasiswa melihat daftar lamarannya (200)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/mahasiswa/applications", nil)
		req.Header.Set("Authorization", "Bearer "+mhsCompleteToken)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[[]model.ApplicationResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))
		assert.True(t, webResp.Success)
		assert.Len(t, webResp.Data, 1)
		assert.Equal(t, createdAppID, webResp.Data[0].ID)
	})

	t.Run("Mitra melihat daftar pelamar pada lowongan mereka (200)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/mitra/applications", nil)
		req.Header.Set("Authorization", "Bearer "+mitra1Token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[[]model.ApplicationResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))
		assert.True(t, webResp.Success)
		assert.NotEmpty(t, webResp.Data)
		assert.Equal(t, createdAppID, webResp.Data[0].ID)
		assert.NotNil(t, webResp.Data[0].Mahasiswa)
		assert.Equal(t, "Maya Lengkap", webResp.Data[0].Mahasiswa.Name)
	})

	t.Run("Mahasiswa melamar ke lowongan yang tidak ada ditolak (404)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/jobs/non-existent-job-uuid/apply", nil)
		req.Header.Set("Authorization", "Bearer "+mhsFreshToken)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Non-mahasiswa (Mitra) mencoba melamar lowongan ditolak (403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/jobs/%s/apply", job1.ID), nil)
		req.Header.Set("Authorization", "Bearer "+mitra1Token)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Mitra lain dilarang mengubah status pelamar pada lowongan ini (403)", func(t *testing.T) {
		body, _ := json.Marshal(model.UpdateApplicationStatusRequest{Status: entity.ApplicationStatusRejected})
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/mitra/applications/%s/status", createdAppID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+mitra2Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Mitra pemilik lowongan menolak pelamar (200)", func(t *testing.T) {
		body, _ := json.Marshal(model.UpdateApplicationStatusRequest{Status: entity.ApplicationStatusRejected})
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/mitra/applications/%s/status", createdAppID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+mitra1Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.ApplicationResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))
		assert.True(t, webResp.Success)
		assert.Equal(t, entity.ApplicationStatusRejected, webResp.Data.Status)
	})

	var secondAppID string

	t.Run("Mahasiswa yang telah ditolak dapat mendaftar ke lowongan lain (201)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/jobs/%s/apply", job2.ID), nil)
		req.Header.Set("Authorization", "Bearer "+mhsCompleteToken)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.ApplicationResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))
		assert.True(t, webResp.Success)
		assert.Equal(t, entity.ApplicationStatusPending, webResp.Data.Status)
		assert.Equal(t, job2.ID, webResp.Data.JobID)

		secondAppID = webResp.Data.ID
	})

	t.Run("Mitra pemilik lowongan menerima pelamar pada lowongan kedua (200)", func(t *testing.T) {
		body, _ := json.Marshal(model.UpdateApplicationStatusRequest{Status: entity.ApplicationStatusAccepted})
		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/mitra/applications/%s/status", secondAppID), bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+mitra1Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.ApplicationResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))
		assert.True(t, webResp.Success)
		assert.Equal(t, entity.ApplicationStatusAccepted, webResp.Data.Status)
	})

	t.Run("Mahasiswa yang telah diterima tidak dapat melamar ke manapun lagi (400)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/jobs/%s/apply", job1.ID), nil)
		req.Header.Set("Authorization", "Bearer "+mhsCompleteToken)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		_ = json.Unmarshal(respBytes, &webResp)
		assert.False(t, webResp.Success)
		assert.Contains(t, webResp.Message, "telah diterima")
	})
}
