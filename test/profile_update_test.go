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

	mitraToken := createAndLogin("mitra-1", "Mitra Mandiri", "mitra@mandiri.com", "pass123", entity.RoleMitra)
	mahasiswaToken := createAndLogin("mhs-1", "Siti Mahasiswa", "siti@univ.ac.id", "pass123", entity.RoleMahasiswa)
	adminToken := createAndLogin("admin-1", "Admin Sistem", "admin@root.com", "pass123", entity.RoleAdmin)

	t.Run("Update Profile Mitra - Berhasil dengan format bahasa Indonesia via /api/mitra/profile", func(t *testing.T) {
		reqBody := map[string]string{
			"nama_mitra":   "PT Inovasi Digital Perkasa",
			"lokasi":       "Jakarta Selatan, Indonesia",
			"foto_profile": "https://storage.googleapis.com/logos/corp.png",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/mitra/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mitraToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.UserResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Profil mitra berhasil diperbarui", webResp.Message)
		assert.Equal(t, "PT Inovasi Digital Perkasa", webResp.Data.Name)
		assert.Equal(t, "Jakarta Selatan, Indonesia", webResp.Data.Location)
		assert.Equal(t, "https://storage.googleapis.com/logos/corp.png", webResp.Data.ProfilePicture)

		// Verifikasi juga di database
		var u entity.User
		db.First(&u, "id = ?", "mitra-1")
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
		assert.Equal(t, "Profil mitra berhasil diperbarui", webResp.Message)
		assert.Equal(t, "Mitra Mandiri Sejahtera", webResp.Data.Name)
		assert.Equal(t, "Bandung, Indonesia", webResp.Data.Location)
		assert.Equal(t, "https://storage.googleapis.com/logos/mitra.png", webResp.Data.ProfilePicture)
	})

	t.Run("Update Profile Mitra - Ditolak jika diakses oleh Mahasiswa (403)", func(t *testing.T) {
		reqBody := map[string]string{
			"nama_mitra": "Hacker Corp",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/mitra/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mahasiswaToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Update Profile Mitra - Ditolak jika diakses oleh Admin (403)", func(t *testing.T) {
		reqBody := map[string]string{
			"nama_mitra": "Admin Corp",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/mitra/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Update Profile Mahasiswa - Berhasil dengan format bahasa Indonesia", func(t *testing.T) {
		reqBody := map[string]string{
			"lokasi":          "Bandung, Indonesia",
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
		assert.Equal(t, "Bandung, Indonesia", webResp.Data.Location)
		assert.Equal(t, "https://storage.googleapis.com/avatars/siti.jpg", webResp.Data.ProfilePicture)
		assert.Equal(t, "https://storage.googleapis.com/resumes/cv_siti.pdf", webResp.Data.CV)
		assert.Equal(t, "https://siti.dev", webResp.Data.PortfolioURL)

		// Verifikasi juga di database
		var u entity.User
		db.First(&u, "id = ?", "mhs-1")
		assert.Equal(t, "Bandung, Indonesia", u.Location)
		assert.Equal(t, "https://storage.googleapis.com/avatars/siti.jpg", u.ProfilePicture)
		assert.Equal(t, "https://storage.googleapis.com/resumes/cv_siti.pdf", u.CV)
		assert.Equal(t, "https://siti.dev", u.PortfolioURL)
	})

	t.Run("Update Profile Mahasiswa - Ditolak jika diakses oleh Mitra (403)", func(t *testing.T) {
		reqBody := map[string]string{
			"cv": "https://hacker.com/cv.pdf",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/mahasiswa/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mitraToken))

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

	t.Run("Update Profile Mahasiswa - Ditolak jika URL menggunakan skema javascript atau tidak valid (400)", func(t *testing.T) {
		maliciousInputs := []map[string]string{
			{"cv": "javascript:alert(1)"},
			{"foto_profile": "javascript:void(0)"},
			{"link_portofolio": "javascript:alert(document.cookie)"},
			{"profile_picture": "data:text/html,<script>alert(1)</script>"},
			{"portfolio_url": "ftp://files.example.com"},
			{"cv": "not-a-valid-url"},
		}

		for _, input := range maliciousInputs {
			bodyBytes, _ := json.Marshal(input)
			req := httptest.NewRequest(http.MethodPut, "/api/mahasiswa/profile", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mahasiswaToken))

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		}
	})

	t.Run("Update Profile Mahasiswa - Diterima dengan URL http dan https yang valid (200)", func(t *testing.T) {
		validInput := map[string]string{
			"foto_profile":    "https://example.com/photo.png",
			"cv":              "https://example.com/cv.pdf",
			"link_portofolio": "http://portfolio.my-domain.id",
		}
		bodyBytes, _ := json.Marshal(validInput)
		req := httptest.NewRequest(http.MethodPut, "/api/mahasiswa/profile", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mahasiswaToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
