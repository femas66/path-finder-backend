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

func TestAdminMahasiswaCRUD(t *testing.T) {
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

	// Helper login
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

	adminToken := createAndLogin("admin-root", "Admin Utama", "admin@kampus.id", "admin123", entity.RoleAdmin)
	mahasiswaToken := createAndLogin("mhs-sample", "Budi Mahasiswa", "budi@kampus.id", "budi123", entity.RoleMahasiswa)

	var createdMahasiswaID string

	t.Run("Create Mahasiswa - Berhasil dan Default Password adalah NIM", func(t *testing.T) {
		reqBody := model.CreateMahasiswaRequest{
			Name:  "Ahmad Dahlan",
			Email: "ahmad@kampus.id",
			NIM:   "2201010055",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/mahasiswa", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.MahasiswaResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Ahmad Dahlan", webResp.Data.Name)
		assert.Equal(t, "ahmad@kampus.id", webResp.Data.Email)
		assert.Equal(t, "2201010055", webResp.Data.NIM)
		assert.Equal(t, entity.RoleMahasiswa, webResp.Data.Role)
		assert.NotEmpty(t, webResp.Data.ID)

		createdMahasiswaID = webResp.Data.ID

		// Verifikasi mahasiswa baru bisa login dengan default password = NIM
		loginReq := model.LoginUserRequest{
			Email:    "ahmad@kampus.id",
			Password: "2201010055", // Default password = NIM
		}
		loginBytes, _ := json.Marshal(loginReq)
		reqLogin := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(loginBytes))
		reqLogin.Header.Set("Content-Type", "application/json")

		respLogin, err := app.Test(reqLogin)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, respLogin.StatusCode)

		loginRespBytes, _ := io.ReadAll(respLogin.Body)
		var loginResp model.WebResponse[*model.LoginResponse]
		_ = json.Unmarshal(loginRespBytes, &loginResp)

		assert.True(t, loginResp.Success)
		assert.Equal(t, "2201010055", loginResp.Data.User.NIM)
		assert.Equal(t, entity.RoleMahasiswa, loginResp.Data.User.Role)
		assert.NotEmpty(t, loginResp.Data.Token)
	})

	t.Run("Create Mahasiswa - Ditolak jika Email duplikat (409)", func(t *testing.T) {
		reqBody := model.CreateMahasiswaRequest{
			Name:  "Ahmad Kloning",
			Email: "ahmad@kampus.id", // email sudah ada
			NIM:   "2201010099",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/mahasiswa", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("Create Mahasiswa - Ditolak jika NIM duplikat (409)", func(t *testing.T) {
		reqBody := model.CreateMahasiswaRequest{
			Name:  "Ahmad Lain",
			Email: "ahmad_lain@kampus.id",
			NIM:   "2201010055", // NIM sudah ada
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/admin/mahasiswa", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("Get Mahasiswa by ID - Berhasil", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/mahasiswa/%s", createdMahasiswaID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.MahasiswaResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, createdMahasiswaID, webResp.Data.ID)
		assert.Equal(t, "2201010055", webResp.Data.NIM)
	})

	t.Run("List Mahasiswa - Berhasil dengan Paging", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/mahasiswa?page=1&size=10&search=Ahmad", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[[]*model.MahasiswaResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.GreaterOrEqual(t, len(webResp.Data), 1)
		assert.NotNil(t, webResp.Paging)
		assert.Equal(t, 1, webResp.Paging.Page)
	})

	t.Run("Update Mahasiswa - Berhasil", func(t *testing.T) {
		reqBody := model.UpdateMahasiswaByAdminRequest{
			Name: "Ahmad Dahlan S.Kom",
			NIM:  "2201010056",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/mahasiswa/%s", createdMahasiswaID), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.MahasiswaResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Ahmad Dahlan S.Kom", webResp.Data.Name)
		assert.Equal(t, "2201010056", webResp.Data.NIM)
	})

	t.Run("Delete Mahasiswa - Berhasil", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/admin/mahasiswa/%s", createdMahasiswaID), nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verifikasi mahasiswa tidak ditemukan lagi
		reqCheck := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/admin/mahasiswa/%s", createdMahasiswaID), nil)
		reqCheck.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		respCheck, err := app.Test(reqCheck)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, respCheck.StatusCode)
	})

	t.Run("Non-Admin Ditolak mengakses /api/admin/mahasiswa (403)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/mahasiswa", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mahasiswaToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Tanpa Token Ditolak mengakses /api/admin/mahasiswa (401)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/mahasiswa", nil)

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
