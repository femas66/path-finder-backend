package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestForgotPassword(t *testing.T) {
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

	// Bersihkan data sebelum test
	db.Exec("DELETE FROM password_resets")
	db.Exec("DELETE FROM users")

	// Buat user contoh
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	testUser := &entity.User{
		ID:       uuid.New().String(),
		Name:     "Femas Akbar",
		Email:    "femas@example.com",
		Password: string(hashedPassword),
		Role:     entity.RoleMahasiswa,
	}
	err := db.Create(testUser).Error
	assert.NoError(t, err)

	t.Run("Forgot Password - Success and Fast Response", func(t *testing.T) {
		reqBody := model.ForgotPasswordRequest{
			Email: "femas@example.com",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/forgot-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		startTime := time.Now()
		resp, err := app.Test(req)
		elapsed := time.Since(startTime)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verifikasi response time sangat cepat (< 100ms) karena asynchronous goroutine
		assert.Less(t, elapsed, 100*time.Millisecond, "Response time harus cepat dan non-blocking")

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.ForgotPasswordResponse]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Kode verifikasi telah dikirim ke email", webResp.Message)
		assert.Equal(t, "femas@example.com", webResp.Data.Email)

		// Verifikasi kode 6-digit tersimpan di tabel password_resets
		var resetRecord entity.PasswordReset
		err = db.Where("email = ?", "femas@example.com").Take(&resetRecord).Error
		assert.NoError(t, err)
		assert.Equal(t, 6, len(resetRecord.Code))
		assert.True(t, resetRecord.ExpiredAt.After(time.Now()))
	})

	t.Run("Forgot Password - User Not Found", func(t *testing.T) {
		reqBody := model.ForgotPasswordRequest{
			Email: "tidakada@example.com",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/forgot-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)

		assert.False(t, webResp.Success)
		assert.Equal(t, "email tidak ditemukan", webResp.Message)
	})

	t.Run("Forgot Password - Invalid Email Format", func(t *testing.T) {
		reqBody := model.ForgotPasswordRequest{
			Email: "bukan-format-email",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/forgot-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
