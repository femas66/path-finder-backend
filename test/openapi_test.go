package test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	controller "haji-koding/path-finder/internal/delivery/http"
	"haji-koding/path-finder/internal/delivery/http/route"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func TestOpenAPIDocumentation(t *testing.T) {
	app := fiber.New()
	pass := func(c fiber.Ctx) error { return c.Next() }
	// Register every route without a DB; handlers are not invoked by this test.
	routes := route.RouteConfig{
		App: app, UserController: &controller.UserController{},
		AdminMahasiswaController: &controller.AdminMahasiswaController{},
		HajiCodingController:     &controller.HajiCodingController{},
		StoryController:          &controller.StoryController{},
		AuthMiddleware:           pass, AdminMiddleware: pass, MitraMiddleware: pass, MahasiswaMiddleware: pass,
	}
	routes.Setup()

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "application/json")
	var spec struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]map[string]struct {
			OperationID string                     `json:"operationId"`
			Responses   map[string]json.RawMessage `json:"responses"`
		} `json:"paths"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&spec))
	require.Equal(t, "3.0.3", spec.OpenAPI)

	operations := make(map[string]bool)
	ids := make(map[string]bool)
	for path, item := range spec.Paths {
		for method, op := range item {
			if method == "parameters" || method == "summary" || method == "description" {
				continue
			}
			require.NotEmpty(t, op.OperationID, "%s %s", method, path)
			require.False(t, ids[op.OperationID], "duplicate operationId: %s", op.OperationID)
			ids[op.OperationID] = true
			require.NotEmpty(t, op.Responses)
			operations[strings.ToUpper(method)+" "+path] = true
		}
	}

	for _, r := range app.GetRoutes() {
		if r.Method == http.MethodHead || r.Path == "/docs" || r.Path == "/openapi.json" {
			continue
		}
		path := strings.ReplaceAll(r.Path, ":id", "{id}")
		key := r.Method + " " + path
		require.True(t, operations[key], "route missing in OpenAPI: %s", key)
		delete(operations, key)
	}
	require.Empty(t, operations, "OpenAPI documents unregistered routes")

	ui, err := app.Test(httptest.NewRequest(http.MethodGet, "/docs", nil))
	require.NoError(t, err)
	defer ui.Body.Close()
	require.Equal(t, http.StatusOK, ui.StatusCode)
	require.Contains(t, ui.Header.Get("Content-Type"), "text/html")
	html, err := io.ReadAll(ui.Body)
	require.NoError(t, err)
	require.Contains(t, string(html), "url: '/openapi.json'")
	require.Contains(t, string(html), "persistAuthorization: false")
}
