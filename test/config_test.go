package test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
)

func TestNewViper(t *testing.T) {
	viperConfig := config.NewViper()

	assert.NotNil(t, viperConfig)
	assert.Equal(t, "path-finder-backend", viperConfig.GetString("app.name"))
	assert.Equal(t, 3000, viperConfig.GetInt("web.port"))
	assert.NotEmpty(t, viperConfig.GetString("database.host"))
	assert.Equal(t, 5432, viperConfig.GetInt("database.port"))
}

func TestNewLogger(t *testing.T) {
	log := config.NewLogger()

	assert.NotNil(t, log)
}

func TestNewValidator(t *testing.T) {
	validate := config.NewValidator()

	assert.NotNil(t, validate)

	type SampleUser struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	validUser := SampleUser{
		Name:  "Femas",
		Email: "femas@example.com",
	}
	err := validate.Struct(validUser)
	assert.NoError(t, err)

	invalidUser := SampleUser{
		Name:  "",
		Email: "not-an-email",
	}
	err = validate.Struct(invalidUser)
	assert.Error(t, err)
}

func TestNewFiber(t *testing.T) {
	viperConfig := config.NewViper()
	app := config.NewFiber(viperConfig)

	assert.NotNil(t, app)
}

func TestFiberErrorHandlerSanitization(t *testing.T) {
	viperConfig := config.NewViper()
	app := config.NewFiber(viperConfig)

	app.Get("/test-internal-error", func(c fiber.Ctx) error {
		return errors.New("pq: relation \"users\" does not exist in SELECT * FROM sensitive_db")
	})

	app.Get("/test-bad-request", func(c fiber.Ctx) error {
		return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	})

	t.Run("Sanitasi Error 500 - Pesan SQL mentah tidak bocor ke publik", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test-internal-error", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		assert.NoError(t, json.Unmarshal(respBytes, &webResp))

		assert.False(t, webResp.Success)
		assert.Equal(t, "terjadi kesalahan pada server", webResp.Message)
		assert.Equal(t, "terjadi kesalahan pada server", webResp.Errors)
		assert.NotContains(t, string(respBytes), "sensitive_db")
		assert.NotContains(t, string(respBytes), "SELECT *")
	})

	t.Run("Error 400 - Pesan ramah klien tetap ditampilkan", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test-bad-request", nil)
		resp, err := app.Test(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[any]
		assert.NoError(t, json.Unmarshal(respBytes, &webResp))

		assert.False(t, webResp.Success)
		assert.Equal(t, "Permintaan tidak valid", webResp.Message)
	})
}
