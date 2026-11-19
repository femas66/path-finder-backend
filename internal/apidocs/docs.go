// Package apidocs serves the embedded OpenAPI contract and Swagger UI.
package apidocs

import (
	_ "embed"

	"github.com/gofiber/fiber/v3"
)

//go:embed openapi.json
var specification []byte

//go:embed index.html
var swaggerUI string

// Register keeps documentation independent of the process working directory.
func Register(app *fiber.App) {
	app.Get("/openapi.json", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/json; charset=utf-8")
		return c.Send(specification)
	})
	app.Get("/docs", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.SendString(swaggerUI)
	})
}
