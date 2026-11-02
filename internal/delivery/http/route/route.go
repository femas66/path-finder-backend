package route

import (
	"haji-koding/path-finder/internal/delivery/http"

	"github.com/gofiber/fiber/v3"
)

type RouteConfig struct {
	App                      *fiber.App
	UserController           *http.UserController
	AdminMahasiswaController *http.AdminMahasiswaController
	HajiCodingController     *http.HajiCodingController
	AuthMiddleware           fiber.Handler
	AdminMiddleware          fiber.Handler
	MitraMiddleware          fiber.Handler
	MahasiswaMiddleware      fiber.Handler
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
		c.App.Post("/api/users/forgot-password", c.UserController.ForgotPassword)
		c.App.Post("/api/auth/register", c.UserController.Register)
		c.App.Post("/api/auth/login", c.UserController.Login)
		c.App.Post("/api/auth/forgot-password", c.UserController.ForgotPassword)
	}
}

func (c *RouteConfig) SetupAuthRoute() {
	if c.UserController != nil && c.AuthMiddleware != nil {
		// Endpoint user terautentikasi (semua role)
		c.App.Get("/api/users/current", c.AuthMiddleware, c.UserController.Current)
		c.App.Get("/api/users/profile", c.AuthMiddleware, c.UserController.Current)
		c.App.Get("/api/auth/me", c.AuthMiddleware, c.UserController.Current)

		// Endpoint khusus role Admin
		if c.AdminMiddleware != nil {
			c.App.Get("/api/admin/dashboard", c.AuthMiddleware, c.AdminMiddleware, c.UserController.AdminDashboard)

			if c.AdminMahasiswaController != nil {
				c.App.Post("/api/admin/mahasiswa", c.AuthMiddleware, c.AdminMiddleware, c.AdminMahasiswaController.Create)
				c.App.Get("/api/admin/mahasiswa", c.AuthMiddleware, c.AdminMiddleware, c.AdminMahasiswaController.List)
				c.App.Get("/api/admin/mahasiswa/:id", c.AuthMiddleware, c.AdminMiddleware, c.AdminMahasiswaController.Get)
				c.App.Put("/api/admin/mahasiswa/:id", c.AuthMiddleware, c.AdminMiddleware, c.AdminMahasiswaController.Update)
				c.App.Patch("/api/admin/mahasiswa/:id", c.AuthMiddleware, c.AdminMiddleware, c.AdminMahasiswaController.Update)
				c.App.Delete("/api/admin/mahasiswa/:id", c.AuthMiddleware, c.AdminMiddleware, c.AdminMahasiswaController.Delete)
			}
		}

		// Endpoint khusus role Mitra / Perusahaan
		if c.MitraMiddleware != nil {
			c.App.Get("/api/mitra/dashboard", c.AuthMiddleware, c.MitraMiddleware, c.UserController.MitraDashboard)
			c.App.Get("/api/perusahaan/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.Current)
			c.App.Get("/api/mitra/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.Current)
			c.App.Put("/api/perusahaan/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.UpdateCompanyProfile)
			c.App.Patch("/api/perusahaan/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.UpdateCompanyProfile)
			c.App.Put("/api/mitra/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.UpdateCompanyProfile)
			c.App.Patch("/api/mitra/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.UpdateCompanyProfile)
		}

		// Endpoint khusus role Mahasiswa
		if c.MahasiswaMiddleware != nil {
			c.App.Get("/api/mahasiswa/dashboard", c.AuthMiddleware, c.MahasiswaMiddleware, c.UserController.MahasiswaDashboard)
			c.App.Get("/api/mahasiswa/profile", c.AuthMiddleware, c.MahasiswaMiddleware, c.UserController.Current)
			c.App.Put("/api/mahasiswa/profile", c.AuthMiddleware, c.MahasiswaMiddleware, c.UserController.UpdateMahasiswaProfile)
			c.App.Patch("/api/mahasiswa/profile", c.AuthMiddleware, c.MahasiswaMiddleware, c.UserController.UpdateMahasiswaProfile)
		}
	}
}
