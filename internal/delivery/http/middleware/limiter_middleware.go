package middleware

import (
	"time"

	"haji-koding/path-finder/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// NewRateLimiter membuat middleware rate limit per alamat IP klien
// untuk mencegah brute-force tebak email maupun spamming kode OTP.
func NewRateLimiter(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
		KeyGenerator: func(c fiber.Ctx) string {
			ip := c.IP()
			if ip == "" {
				ip = "127.0.0.1"
			}
			return ip
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(model.WebResponse[any]{
				Success: false,
				Message: "Terlalu banyak permintaan. Silakan coba lagi beberapa saat.",
				Errors:  "rate limit exceeded",
			})
		},
	})
}
