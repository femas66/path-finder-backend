package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTestApp() (*gorm.DB, *http.Server, error) {
	return nil, nil, nil
}

func clearUsers(db *gorm.DB) {
	db.Exec("DELETE FROM users")
}

func TestUserRegistrationAndLogin(t *testing.T) {
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

	clearUsers(db)

	t.Run("Register Success", func(t *testing.T) {
		reqBody := model.RegisterUserRequest{
			Name:            "Femas Akbar",
			Email:           "femas@example.com",
			Password:        "password123",
			ConfirmPassword: "password123",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.UserResponse]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Registrasi berhasil", webResp.Message)
		assert.Equal(t, reqBody.Name, webResp.Data.Name)
		assert.Equal(t, reqBody.Email, webResp.Data.Email)
		assert.Equal(t, entity.RoleMahasiswa, webResp.Data.Role)
		assert.NotEmpty(t, webResp.Data.ID)
	})

	t.Run("Register Duplicate Email", func(t *testing.T) {
		reqBody := model.RegisterUserRequest{
			Name:            "Femas Duplikat",
			Email:           "femas@example.com",
			Password:        "password123",
			ConfirmPassword: "password123",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusConflict, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)

		assert.False(t, webResp.Success)
		assert.Equal(t, "email sudah terdaftar", webResp.Message)
	})

	t.Run("Register Password Mismatch", func(t *testing.T) {
		reqBody := model.RegisterUserRequest{
			Name:            "Budi Santoso",
			Email:           "budi@example.com",
			Password:        "password123",
			ConfirmPassword: "passwordBeda",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)

		assert.False(t, webResp.Success)
	})

	t.Run("Register Invalid Email", func(t *testing.T) {
		reqBody := model.RegisterUserRequest{
			Name:            "Budi Santoso",
			Email:           "bukan-email",
			Password:        "password123",
			ConfirmPassword: "password123",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/register", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Login Success", func(t *testing.T) {
		reqBody := model.LoginUserRequest{
			Email:    "femas@example.com",
			Password: "password123",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.LoginResponse]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)

		assert.True(t, webResp.Success)
		assert.Equal(t, "Login berhasil", webResp.Message)
		assert.NotEmpty(t, webResp.Data.Token)
		assert.Equal(t, "femas@example.com", webResp.Data.User.Email)
		assert.Equal(t, entity.RoleMahasiswa, webResp.Data.User.Role)
	})

	t.Run("Login Wrong Password", func(t *testing.T) {
		reqBody := model.LoginUserRequest{
			Email:    "femas@example.com",
			Password: "salahPassword",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)

		assert.False(t, webResp.Success)
		assert.Equal(t, "email atau password salah", webResp.Message)
	})

	t.Run("Login Email Not Found", func(t *testing.T) {
		reqBody := model.LoginUserRequest{
			Email:    "tidakada@example.com",
			Password: "password123",
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		err = json.Unmarshal(respBytes, &webResp)
		assert.NoError(t, err)

		assert.False(t, webResp.Success)
		assert.Equal(t, "email atau password salah", webResp.Message)
	})
}
