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

func TestRoleMiddlewareAndLogin(t *testing.T) {
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

	// Bersihkan data user sebelum test
	db.Exec("DELETE FROM users")

	// Helper untuk membuat user dan mendapatkan JWT token via endpoint Login
	createAndLoginUser := func(id, name, email, password, role string) string {
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
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.LoginResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, role, webResp.Data.User.Role)
		assert.NotEmpty(t, webResp.Data.Token)

		return webResp.Data.Token
	}

	// 1. Setup user untuk masing-masing role dan verifikasi login
	adminToken := createAndLoginUser("user-admin-1", "Admin Utama", "admin@pathfinder.com", "admin123", entity.RoleAdmin)
	mitraToken := createAndLoginUser("user-mitra-1", "PT Teknologi Nusantara", "mitra@tech.com", "mitra123", entity.RoleMitra)
	perusahaanToken := createAndLoginUser("user-perusahaan-1", "PT Maju Bersama", "hrd@maju.com", "perusahaan123", entity.RolePerusahaan)
	mahasiswaToken := createAndLoginUser("user-mhs-1", "Budi Mahasiswa", "budi@kampus.ac.id", "mhs12345", entity.RoleMahasiswa)

	// Helper function untuk request GET terproteksi
	testProtectedEndpoint := func(url string, token string) *http.Response {
		req := httptest.NewRequest(http.MethodGet, url, nil)
		if token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}
		resp, err := app.Test(req)
		assert.NoError(t, err)
		return resp
	}

	t.Run("Endpoint Current User - Dapat diakses oleh semua role", func(t *testing.T) {
		tokens := []struct {
			role  string
			token string
			email string
		}{
			{"admin", adminToken, "admin@pathfinder.com"},
			{"mitra", mitraToken, "mitra@tech.com"},
			{"perusahaan", perusahaanToken, "hrd@maju.com"},
			{"mahasiswa", mahasiswaToken, "budi@kampus.ac.id"},
		}

		for _, item := range tokens {
			resp := testProtectedEndpoint("/api/users/current", item.token)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			respBytes, _ := io.ReadAll(resp.Body)
			var webResp model.WebResponse[*model.UserResponse]
			_ = json.Unmarshal(respBytes, &webResp)

			assert.True(t, webResp.Success)
			assert.Equal(t, item.email, webResp.Data.Email)
			assert.Equal(t, item.role, webResp.Data.Role)
		}
	})

	t.Run("Admin Middleware - /api/admin/dashboard", func(t *testing.T) {
		// Admin berhasil (200)
		resp := testProtectedEndpoint("/api/admin/dashboard", adminToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Mitra ditolak (403 Forbidden)
		resp = testProtectedEndpoint("/api/admin/dashboard", mitraToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Perusahaan ditolak (403 Forbidden)
		resp = testProtectedEndpoint("/api/admin/dashboard", perusahaanToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Mahasiswa ditolak (403 Forbidden)
		resp = testProtectedEndpoint("/api/admin/dashboard", mahasiswaToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Tanpa token (401 Unauthorized)
		resp = testProtectedEndpoint("/api/admin/dashboard", "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Mitra/Perusahaan Middleware - /api/mitra/dashboard", func(t *testing.T) {
		// Mitra berhasil (200)
		resp := testProtectedEndpoint("/api/mitra/dashboard", mitraToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Perusahaan juga berhasil (200)
		resp = testProtectedEndpoint("/api/mitra/dashboard", perusahaanToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Admin ditolak (403 Forbidden)
		resp = testProtectedEndpoint("/api/mitra/dashboard", adminToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Mahasiswa ditolak (403 Forbidden)
		resp = testProtectedEndpoint("/api/mitra/dashboard", mahasiswaToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Tanpa token (401 Unauthorized)
		resp = testProtectedEndpoint("/api/mitra/dashboard", "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Mahasiswa Middleware - /api/mahasiswa/dashboard", func(t *testing.T) {
		// Mahasiswa berhasil (200)
		resp := testProtectedEndpoint("/api/mahasiswa/dashboard", mahasiswaToken)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Admin ditolak (403 Forbidden)
		resp = testProtectedEndpoint("/api/mahasiswa/dashboard", adminToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Mitra ditolak (403 Forbidden)
		resp = testProtectedEndpoint("/api/mahasiswa/dashboard", mitraToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Perusahaan ditolak (403 Forbidden)
		resp = testProtectedEndpoint("/api/mahasiswa/dashboard", perusahaanToken)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)

		// Tanpa token (401 Unauthorized)
		resp = testProtectedEndpoint("/api/mahasiswa/dashboard", "")
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Register dengan Spesifik Role", func(t *testing.T) {
		reqBody := model.RegisterUserRequest{
			Name:            "Mitra Baru",
			Email:           "mitrabaru@company.com",
			Password:        "password123",
			ConfirmPassword: "password123",
			Role:            entity.RoleMitra,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.UserResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, entity.RoleMitra, webResp.Data.Role)
	})
}
