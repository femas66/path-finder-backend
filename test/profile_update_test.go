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

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestProfileUpdate(t *testing.T) {
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

	db.Exec("DELETE FROM users")

	createAndLogin := func(id, name, email, password, role string) string {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		u := &entity.User{
			ID:       id,
			Name:     name,
			Email:    email,
			Password: string(hashed),
			Role:     role,
		}
		err := db.Create(u).Error
		assert.NoError(t, err)

		loginBody := model.LoginUserRequest{
			Email:    email,
			Password: password,
		}
		loginBytes, _ := json.Marshal(loginBody)
		req := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(loginBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.LoginResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		return webResp.Data.Token
	}

	perusahaanToken := createAndLogin("comp-1", "PT Inovasi Digital", "corp@inovasi.com", "pass123", entity.RolePerusahaan)
	mitraToken := createAndLogin("mitra-1", "Mitra Mandiri", "mitra@mandiri.com", "pass123", entity.RoleMitra)
	mahasiswaToken := createAndLogin("mhs-1", "Siti Mahasiswa", "siti@univ.ac.id", "pass123", entity.RoleMahasiswa)
	adminToken := createAndLogin("admin-1", "Admin Sistem", "admin@root.com", "pass123", entity.RoleAdmin)

	t.Run("Update Profile Perusahaan - Berhasil dengan format bahasa Indonesia", func(t *testing.T) {
		reqBody := map[string]string{
			"nama_perusahaan": "PT Inovasi Digital Perkasa",
			"lokasi":          "Jakarta Selatan, Indonesia",
			"foto_profile":    "https://storage.googleapis.com/logos/corp.png",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/perusahaan/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", perusahaanToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.UserResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Profil perusahaan berhasil diperbarui", webResp.Message)
		assert.Equal(t, "PT Inovasi Digital Perkasa", webResp.Data.Name)
		assert.Equal(t, "Jakarta Selatan, Indonesia", webResp.Data.Location)
		assert.Equal(t, "https://storage.googleapis.com/logos/corp.png", webResp.Data.ProfilePicture)

		// Verifikasi juga di database
		var u entity.User
		db.First(&u, "id = ?", "comp-1")
		assert.Equal(t, "PT Inovasi Digital Perkasa", u.Name)
		assert.Equal(t, "Jakarta Selatan, Indonesia", u.Location)
		assert.Equal(t, "https://storage.googleapis.com/logos/corp.png", u.ProfilePicture)
	})

	t.Run("Update Profile Mitra - Berhasil dengan format bahasa Inggris via /api/mitra/profile", func(t *testing.T) {
		reqBody := map[string]string{
			"name":            "Mitra Mandiri Sejahtera",
			"location":        "Bandung, Indonesia",
			"profile_picture": "https://storage.googleapis.com/logos/mitra.png",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPatch, "/api/mitra/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mitraToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.UserResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Mitra Mandiri Sejahtera", webResp.Data.Name)
		assert.Equal(t, "Bandung, Indonesia", webResp.Data.Location)
		assert.Equal(t, "https://storage.googleapis.com/logos/mitra.png", webResp.Data.ProfilePicture)
	})

	t.Run("Update Profile Perusahaan - Ditolak jika diakses oleh Mahasiswa (403)", func(t *testing.T) {
		reqBody := map[string]string{
			"nama_perusahaan": "Hacker Corp",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/perusahaan/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mahasiswaToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Update Profile Perusahaan - Ditolak jika diakses oleh Admin (403)", func(t *testing.T) {
		reqBody := map[string]string{
			"nama_perusahaan": "Admin Corp",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/perusahaan/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Update Profile Mahasiswa - Berhasil dengan format bahasa Indonesia", func(t *testing.T) {
		reqBody := map[string]string{
			"foto_profile":    "https://storage.googleapis.com/avatars/siti.jpg",
			"cv":              "https://storage.googleapis.com/resumes/cv_siti.pdf",
			"link_portofolio": "https://siti.dev",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/mahasiswa/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mahasiswaToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.UserResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Profil mahasiswa berhasil diperbarui", webResp.Message)
		assert.Equal(t, "https://storage.googleapis.com/avatars/siti.jpg", webResp.Data.ProfilePicture)
		assert.Equal(t, "https://storage.googleapis.com/resumes/cv_siti.pdf", webResp.Data.CV)
		assert.Equal(t, "https://siti.dev", webResp.Data.PortfolioURL)

		// Verifikasi juga di database
		var u entity.User
		db.First(&u, "id = ?", "mhs-1")
		assert.Equal(t, "https://storage.googleapis.com/avatars/siti.jpg", u.ProfilePicture)
		assert.Equal(t, "https://storage.googleapis.com/resumes/cv_siti.pdf", u.CV)
		assert.Equal(t, "https://siti.dev", u.PortfolioURL)
	})

	t.Run("Update Profile Mahasiswa - Ditolak jika diakses oleh Perusahaan (403)", func(t *testing.T) {
		reqBody := map[string]string{
			"cv": "https://hacker.com/cv.pdf",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/mahasiswa/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", perusahaanToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Update Profile Mahasiswa - Ditolak jika diakses tanpa token (401)", func(t *testing.T) {
		reqBody := map[string]string{
			"cv": "https://hacker.com/cv.pdf",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/mahasiswa/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
