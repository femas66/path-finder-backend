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

	cleanDB := func() {
		db.Exec("DELETE FROM applications")
		db.Exec("DELETE FROM jobs")
		db.Exec("DELETE FROM stories")
		db.Exec("DELETE FROM password_resets")
		db.Exec("DELETE FROM users")
	}
	cleanDB()
	t.Cleanup(cleanDB)

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

	var createdStoryID string

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

		createdStoryID = webResp.Data.ID
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

	updatedContent := "Saya telah merevisi cerita saya untuk memperjelas minat karir di backend engineering dan cloud infrastructure."
	t.Run("Update cerita berhasil dengan ID - PUT", func(t *testing.T) {
		reqBody := model.UpdateStoryRequest{Content: updatedContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/stories/%s", createdStoryID), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.StoryResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Cerita berhasil diperbarui", webResp.Message)
		assert.Equal(t, createdStoryID, webResp.Data.ID)
		assert.Equal(t, updatedContent, webResp.Data.Content)
		assert.Equal(t, entity.StoryAnalysisPending, webResp.Data.AnalysisStatus)
	})

	t.Run("Update cerita berhasil dengan ID - PATCH", func(t *testing.T) {
		patchContent := "Revisi patch: Minat kuat pada Go, microservices, distributed systems, dan PostgreSQL optimization."
		reqBody := model.UpdateStoryRequest{Content: patchContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/stories/%s", createdStoryID), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.StoryResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, patchContent, webResp.Data.Content)
	})

	t.Run("Update cerita berhasil dengan alias me", func(t *testing.T) {
		meContent := "Update menggunakan alias me untuk memperbarui cerita terbaru pengguna secara langsung dan praktis."
		reqBody := model.UpdateStoryRequest{Content: meContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/stories/me", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.StoryResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		assert.True(t, webResp.Success)
		assert.Equal(t, meContent, webResp.Data.Content)
	})

	t.Run("Update cerita berhasil dengan alias case-insensitive ME dan Me (200)", func(t *testing.T) {
		for _, alias := range []string{"ME", "Me"} {
			aliasContent := fmt.Sprintf("Update menggunakan alias case-insensitive %s untuk memperbarui cerita pengguna secara fleksibel.", alias)
			reqBody := model.UpdateStoryRequest{Content: aliasContent}
			bodyBytes, _ := json.Marshal(reqBody)

			req := httptest.NewRequest(http.MethodPut, "/api/stories/"+alias, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			respBytes, _ := io.ReadAll(resp.Body)
			var webResp model.WebResponse[*model.StoryResponse]
			_ = json.Unmarshal(respBytes, &webResp)

			assert.True(t, webResp.Success)
			assert.Equal(t, aliasContent, webResp.Data.Content)
		}
	})

	t.Run("Update cerita terlalu pendek ditolak (400)", func(t *testing.T) {
		reqBody := model.UpdateStoryRequest{Content: "pendek"}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/stories/%s", createdStoryID), bytes.NewReader(bodyBytes))
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

	t.Run("Update cerita tidak ditemukan (404)", func(t *testing.T) {
		reqBody := model.UpdateStoryRequest{Content: updatedContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, "/api/stories/non-existent-id", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Update cerita milik user lain ditolak (404)", func(t *testing.T) {
		otherMhsToken := createAndLogin("mhs-other", "Other Mhs", "other.mhs@univ.ac.id", "pass123", entity.RoleMahasiswa)
		reqBody := model.UpdateStoryRequest{Content: updatedContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/stories/%s", createdStoryID), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", otherMhsToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Update cerita role non-mahasiswa ditolak (403)", func(t *testing.T) {
		adminToken := createAndLogin("admin-story-2", "Admin2", "admin2.story@univ.ac.id", "pass123", entity.RoleAdmin)
		reqBody := model.UpdateStoryRequest{Content: updatedContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/stories/%s", createdStoryID), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", adminToken))

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("Update cerita tanpa token ditolak (401)", func(t *testing.T) {
		reqBody := model.UpdateStoryRequest{Content: updatedContent}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/stories/%s", createdStoryID), bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}
