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

		// Verifikasi response time cepat dan non-blocking karena asynchronous goroutine
		assert.Less(t, elapsed, 1*time.Second, "Response time harus cepat dan non-blocking")

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

	t.Run("Verify Reset Code - Success", func(t *testing.T) {
		// Buat record OTP di database
		resetRecord := &entity.PasswordReset{
			ID:        uuid.New().String(),
			Email:     "femas@example.com",
			Code:      "654321",
			Attempts:  0,
			ExpiredAt: time.Now().Add(15 * time.Minute),
		}
		db.Create(resetRecord)

		reqBody := model.VerifyResetCodeRequest{
			Email: "femas@example.com",
			Code:  "654321",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/verify-reset-code", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.VerifyResetCodeResponse]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)
		assert.True(t, webResp.Success)
		assert.Equal(t, "Kode verifikasi valid", webResp.Message)
		assert.True(t, webResp.Data.Valid)
	})

	t.Run("Verify Reset Code - Wrong Code", func(t *testing.T) {
		reqBody := model.VerifyResetCodeRequest{
			Email: "femas@example.com",
			Code:  "000000",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/verify-reset-code", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		_ = json.Unmarshal(respBytes, &webResp)
		assert.False(t, webResp.Success)
		assert.Equal(t, "kode verifikasi salah", webResp.Message)
	})

	t.Run("Reset Password - Success and Login with New Password", func(t *testing.T) {
		// Record reset masih ada dari test sebelumnya (code: 654321)
		reqBody := model.ResetPasswordRequest{
			Email:           "femas@example.com",
			Code:            "654321",
			Password:        "newpassword123",
			ConfirmPassword: "newpassword123",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/reset-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		_ = json.Unmarshal(respBytes, &webResp)
		assert.True(t, webResp.Success)
		assert.Contains(t, webResp.Message, "Password berhasil diubah")

		// Verifikasi record reset terhapus
		var checkReset entity.PasswordReset
		err = db.Where("email = ?", "femas@example.com").Take(&checkReset).Error
		assert.Error(t, err)

		// Verifikasi login dengan password baru
		loginReq := model.LoginUserRequest{
			Email:    "femas@example.com",
			Password: "newpassword123",
		}
		loginBytes, _ := json.Marshal(loginReq)
		loginHttpReq := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(loginBytes))
		loginHttpReq.Header.Set("Content-Type", "application/json")

		loginResp, err := app.Test(loginHttpReq)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, loginResp.StatusCode)
	})

	t.Run("Reset Password - Mismatched Password", func(t *testing.T) {
		reqBody := model.ResetPasswordRequest{
			Email:           "femas@example.com",
			Code:            "654321",
			Password:        "newpassword123",
			ConfirmPassword: "differentpassword",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/reset-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}
