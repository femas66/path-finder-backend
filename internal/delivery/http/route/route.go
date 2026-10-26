package route

import (
	"haji-koding/path-finder/internal/delivery/http"

	"github.com/gofiber/fiber/v3"
)

type RouteConfig struct {
	App                  *fiber.App
	UserController       *http.UserController
	HajiCodingController *http.HajiCodingController
	AuthMiddleware       fiber.Handler
}

func (c *RouteConfig) Setup() {
	c.SetupGuestRoute()
	c.SetupAuthRoute()
}

func (c *RouteConfig) SetupGuestRoute() {
	if c.HajiCodingController != nil {
		c.App.Get("/", c.HajiCodingController.HaiBos)
	}

	if c.UserController != nil {
		c.App.Post("/api/users/register", c.UserController.Register)
		c.App.Post("/api/users/login", c.UserController.Login)
		c.App.Post("/api/auth/register", c.UserController.Register)
		c.App.Post("/api/auth/login", c.UserController.Login)
	}
}

func (c *RouteConfig) SetupAuthRoute() {

}
