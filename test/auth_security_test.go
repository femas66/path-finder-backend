package test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"haji-koding/path-finder/internal/config"
	controller "haji-koding/path-finder/internal/delivery/http"
	"haji-koding/path-finder/internal/delivery/http/middleware"
	"haji-koding/path-finder/internal/delivery/http/route"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestPublicRegistrationRejectsAdmin(t *testing.T) {
	// No DB: forbidden registration must fail before any persistence access.
	logger := config.NewLogger()
	userUseCase := usecase.NewUserUseCase(nil, logger, config.NewValidator(), nil, nil, nil, nil)
	app := config.NewFiber(viper.New())
	routes := route.RouteConfig{App: app, UserController: controller.NewUserController(userUseCase, logger)}
	routes.SetupGuestRoute()

	for _, path := range []string{"/api/users/register", "/api/auth/register"} {
		t.Run(path, func(t *testing.T) {
			body := []byte(`{"name":"Public Admin","email":"admin@example.com","password":"password123","confirm_password":"password123","role":"admin"}`)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestPublicRegistrationRoleValidation(t *testing.T) {
	for _, role := range []string{"", "mahasiswa", "mitra", "admin", "Admin", "invalid"} {
		t.Run("role="+role, func(t *testing.T) {
			request := &model.RegisterUserRequest{
				Name: "Test User", Email: "user@example.com", Password: "password123",
				ConfirmPassword: "password123", Role: role,
			}
			err := config.NewValidator().Struct(request)
			if role == "" || role == "mahasiswa" || role == "mitra" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAuthRejectsMissingSecret(t *testing.T) {
	for _, cfg := range []*viper.Viper{nil, viper.New()} {
		require.Panics(t, func() { middleware.NewAuth(cfg) })
	}
	cfg := viper.New()
	cfg.Set("jwt.secret", " \t\n")
	require.Panics(t, func() { middleware.NewAuth(cfg) })
}

func TestLoginRejectsMissingSecret(t *testing.T) {
	logger := config.NewLogger()
	userUseCase := usecase.NewUserUseCase(nil, logger, config.NewValidator(), viper.New(), nil, nil, nil)
	_, err := userUseCase.Login(context.Background(), &model.LoginUserRequest{
		Email: "user@example.com", Password: "password123",
	})
	var responseError *fiber.Error
	require.ErrorAs(t, err, &responseError)
	require.Equal(t, http.StatusInternalServerError, responseError.Code)
}

func TestAuthUsesConfiguredSecret(t *testing.T) {
	cfg := viper.New()
	cfg.Set("jwt.secret", "test-only-configured-secret")
	app := fiber.New()
	app.Get("/protected", middleware.NewAuth(cfg), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	for _, tc := range []struct {
		name   string
		secret string
		status int
	}{
		{"configured", "test-only-configured-secret", http.StatusOK},
		{"other", "test-only-other-secret", http.StatusUnauthorized},
		{"empty", "", http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"user_id": "user-1", "role": "mahasiswa", "exp": time.Now().Add(time.Minute).Unix(),
			})
			signed, err := token.SignedString([]byte(tc.secret))
			require.NoError(t, err)
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer "+signed)
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			require.Equal(t, tc.status, resp.StatusCode)
		})
	}
}
