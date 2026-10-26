package middleware

import (
	"github.com/gofiber/fiber/v3"
)

func NewAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		// request := &model.VerifyUserRequest{Token: c.Get("Authorization", "NOT_FOUND")}
		return c.Next()
	}
}
