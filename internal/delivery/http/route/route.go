package route

import (
	"haji-koding/path-finder/internal/delivery/http"

	"github.com/gofiber/fiber/v3"
)

type RouteConfig struct {
	App                  *fiber.App
	HajiCodingController *http.HajiCodingController
	AuthMiddleware       fiber.Handler
}

func (c *RouteConfig) Setup() {
	c.App.Get("/", c.HajiCodingController.HaiBos)
}

func (c *RouteConfig) SetupGuestRoute() {

}

func (c *RouteConfig) SetupAuthRoute() {

}
