package middleware

import (
	"strings"

	"haji-koding/path-finder/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

func NewAuth(config *viper.Viper) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
				Success: false,
				Message: "Unauthorized: token tidak ditemukan",
				Errors:  "unauthorized",
			})
		}

		parts := strings.Split(authHeader, " ")
		tokenString := authHeader
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			tokenString = parts[1]
		}

		secret := "super-secret-path-finder-key-2026"
		if config != nil && config.GetString("jwt.secret") != "" {
			secret = config.GetString("jwt.secret")
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
				Success: false,
				Message: "Unauthorized: token tidak valid atau kedaluwarsa",
				Errors:  "unauthorized",
			})
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Locals("auth", claims)
		}

		return c.Next()
	}
}
