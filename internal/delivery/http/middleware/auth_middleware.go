package middleware

import (
	"strings"

	"haji-koding/path-finder/internal/entity"
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
			if userID, ok := claims["user_id"].(string); ok {
				c.Locals("user_id", userID)
			}
			if role, ok := claims["role"].(string); ok {
				c.Locals("role", role)
			}
			if email, ok := claims["email"].(string); ok {
				c.Locals("email", email)
			}
		}

		return c.Next()
	}
}

// RequireRole memeriksa apakah user yang terautentikasi memiliki salah satu dari allowedRoles
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		roleVal := c.Locals("role")
		var currentRole string

		if roleStr, ok := roleVal.(string); ok && roleStr != "" {
			currentRole = roleStr
		} else if claims, ok := c.Locals("auth").(jwt.MapClaims); ok && claims != nil {
			if r, ok := claims["role"].(string); ok {
				currentRole = r
			}
		}

		if currentRole == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
				Success: false,
				Message: "Unauthorized: pengguna belum terautentikasi",
				Errors:  "unauthorized",
			})
		}

		for _, allowed := range allowedRoles {
			if strings.EqualFold(currentRole, allowed) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Forbidden: role tidak memiliki akses ke resource ini",
			Errors:  "forbidden",
		})
	}
}

// OnlyAdmin mengizinkan akses hanya untuk role admin
func OnlyAdmin() fiber.Handler {
	return RequireRole(entity.RoleAdmin)
}

// OnlyMitra mengizinkan akses untuk role mitra atau perusahaan
func OnlyMitra() fiber.Handler {
	return RequireRole(entity.RoleMitra, entity.RolePerusahaan)
}

// OnlyMahasiswa mengizinkan akses hanya untuk role mahasiswa
func OnlyMahasiswa() fiber.Handler {
	return RequireRole(entity.RoleMahasiswa)
}
