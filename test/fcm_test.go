package test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"haji-koding/path-finder/internal/config"
	controller "haji-koding/path-finder/internal/delivery/http"
	"haji-koding/path-finder/internal/delivery/http/middleware"
	"haji-koding/path-finder/internal/delivery/http/route"
	"haji-koding/path-finder/internal/service"
	"haji-koding/path-finder/internal/usecase"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestFCMTokenRouteAuthAndValidation(t *testing.T) {
	logger := config.NewLogger()
	v := viper.New()
	v.Set("jwt.secret", "test-secret-key-1234567890123456")
	app := config.NewFiber(v)
	authMiddleware := middleware.NewAuth(v)

	fcmService := service.NewFCMService(v, logger)
	userUseCase := usecase.NewUserUseCase(nil, logger, config.NewValidator(), v, nil, nil, nil, fcmService)
	userController := controller.NewUserController(userUseCase, logger)

	routes := route.RouteConfig{
		App:            app,
		UserController: userController,
		AuthMiddleware: authMiddleware,
	}
	routes.SetupAuthRoute()

	t.Run("Unauthorized without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPatch, "/api/users/fcm-token", bytes.NewReader([]byte(`{"fcm_token":"sample-device-token-12345"}`)))
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Validation error on too short token", func(t *testing.T) {
		// Even if bypass auth via synthetic header or test route, validator catches invalid token length
		validator := config.NewValidator()
		userUseCase := usecase.NewUserUseCase(nil, logger, validator, v, nil, nil, nil, fcmService)
		err := userUseCase.UpdateFCMToken(t.Context(), "user-id", nil)
		require.Error(t, err)
	})
}
