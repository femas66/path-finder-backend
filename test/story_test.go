package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestStoryCreate(t *testing.T) {
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

	db.Exec("DELETE FROM stories")
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

	token := createAndLogin("mhs-story", "Siti Cerita", "siti.story@univ.ac.id", "pass123", entity.RoleMahasiswa)
	validContent := strings.Repeat("Saya suka membangun aplikasi mobile dan belajar hal baru setiap hari. ", 2)
	// Server melakukan trim sebelum menyimpan.
	expectedContent := strings.TrimSpace(validContent)

	t.Run("Kirim cerita berhasil", func(t *testing.T) {
		reqBody := model.CreateStoryRequest{Content: validContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/stories", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.StoryResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Cerita berhasil disimpan", webResp.Message)
		assert.NotEmpty(t, webResp.Data.ID)
		assert.Equal(t, "mhs-story", webResp.Data.UserID)
		assert.Equal(t, expectedContent, webResp.Data.Content)
		assert.Greater(t, webResp.Data.WordCount, 0)
		assert.Equal(t, entity.StoryAnalysisPending, webResp.Data.AnalysisStatus)
	})

	t.Run("Cerita terlalu pendek ditolak", func(t *testing.T) {
		reqBody := model.CreateStoryRequest{Content: "terlalu pendek"}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/stories", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.False(t, webResp.Success)
		assert.Contains(t, webResp.Message, "minimal 50 karakter")
	})

	t.Run("Cerita tanpa token ditolak", func(t *testing.T) {
		reqBody := model.CreateStoryRequest{Content: validContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/stories", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Ambil cerita terakhir milik sendiri", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stories/me", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.StoryResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "mhs-story", webResp.Data.UserID)
		assert.Equal(t, expectedContent, webResp.Data.Content)
	})

	t.Run("Role non-mahasiswa ditolak 403 Forbidden", func(t *testing.T) {
		adminToken := createAndLogin("admin-story", "Admin", "admin.story@univ.ac.id", "pass123", entity.RoleAdmin)
		mitraToken := createAndLogin("mitra-story", "Mitra", "mitra.story@univ.ac.id", "pass123", entity.RoleMitra)

		reqBody := model.CreateStoryRequest{Content: validContent}
		bodyBytes, _ := json.Marshal(reqBody)

		reqAdmin := httptest.NewRequest(http.MethodPost, "/api/stories", bytes.NewReader(bodyBytes))
		reqAdmin.Header.Set("Content-Type", "application/json")
		reqAdmin.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))
		respAdmin, err := app.Test(reqAdmin)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, respAdmin.StatusCode)

		reqMitra := httptest.NewRequest(http.MethodGet, "/api/stories/me", nil)
		reqMitra.Header.Set("Authorization", fmt.Sprintf("Bearer %s", mitraToken))
		respMitra, err := app.Test(reqMitra)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, respMitra.StatusCode)
	})
}
