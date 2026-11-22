package route

import (
	"time"

	"haji-koding/path-finder/internal/apidocs"
	"haji-koding/path-finder/internal/delivery/http"
	"haji-koding/path-finder/internal/delivery/http/middleware"

	"github.com/gofiber/fiber/v3"
)

type RouteConfig struct {
	App                      *fiber.App
	UserController           *http.UserController
	AdminMahasiswaController *http.AdminMahasiswaController
	HajiCodingController     *http.HajiCodingController
	StoryController          *http.StoryController
	JobController            *http.JobController
	ApplicationController    *http.ApplicationController
	AuthMiddleware           fiber.Handler
	AdminMiddleware          fiber.Handler
	MitraMiddleware          fiber.Handler
	MahasiswaMiddleware      fiber.Handler
}

func (c *RouteConfig) Setup() {
	apidocs.Register(c.App)
	c.SetupGuestRoute()
	c.SetupAuthRoute()
}

func (c *RouteConfig) SetupGuestRoute() {
	if c.HajiCodingController != nil {
		c.App.Get("/", c.HajiCodingController.HaiBos)
	}

	if c.UserController != nil {
		limiterMiddleware := middleware.NewRateLimiter(10, 1*time.Minute)

		c.App.Post("/api/users/register", c.UserController.Register)
		c.App.Post("/api/users/login", c.UserController.Login)
		c.App.Post("/api/users/forgot-password", limiterMiddleware, c.UserController.ForgotPassword)
		c.App.Post("/api/users/verify-reset-code", limiterMiddleware, c.UserController.VerifyResetCode)
		c.App.Post("/api/users/reset-password", limiterMiddleware, c.UserController.ResetPassword)

		c.App.Post("/api/auth/register", c.UserController.Register)
		c.App.Post("/api/auth/login", c.UserController.Login)
		c.App.Post("/api/auth/forgot-password", limiterMiddleware, c.UserController.ForgotPassword)
		c.App.Post("/api/auth/verify-reset-code", limiterMiddleware, c.UserController.VerifyResetCode)
		c.App.Post("/api/auth/reset-password", limiterMiddleware, c.UserController.ResetPassword)
	}
}

func (c *RouteConfig) SetupAuthRoute() {
	if c.JobController != nil && c.AuthMiddleware != nil {
		c.App.Get("/api/jobs", c.AuthMiddleware, c.JobController.List)
		c.App.Get("/api/jobs/:id", c.AuthMiddleware, c.JobController.Get)
		if c.MitraMiddleware != nil {
			c.App.Post("/api/mitra/jobs", c.AuthMiddleware, c.MitraMiddleware, c.JobController.Create)
			c.App.Get("/api/mitra/jobs", c.AuthMiddleware, c.MitraMiddleware, c.JobController.ListOwn)
			c.App.Get("/api/mitra/jobs/:id", c.AuthMiddleware, c.MitraMiddleware, c.JobController.GetOwn)
			c.App.Put("/api/mitra/jobs/:id", c.AuthMiddleware, c.MitraMiddleware, c.JobController.Update)
			c.App.Patch("/api/mitra/jobs/:id", c.AuthMiddleware, c.MitraMiddleware, c.JobController.Update)
			c.App.Delete("/api/mitra/jobs/:id", c.AuthMiddleware, c.MitraMiddleware, c.JobController.Delete)
		}
	}
	if c.UserController != nil && c.AuthMiddleware != nil {
		// Endpoint user terautentikasi (semua role)
		c.App.Get("/api/users/current", c.AuthMiddleware, c.UserController.Current)
		c.App.Get("/api/users/profile", c.AuthMiddleware, c.UserController.Current)
		c.App.Get("/api/auth/me", c.AuthMiddleware, c.UserController.Current)
		c.App.Patch("/api/users/fcm-token", c.AuthMiddleware, c.UserController.UpdateFCMToken)

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

		// Endpoint khusus role Mitra
		if c.MitraMiddleware != nil {
			c.App.Get("/api/mitra/dashboard", c.AuthMiddleware, c.MitraMiddleware, c.UserController.MitraDashboard)
			c.App.Get("/api/mitra/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.Current)
			c.App.Put("/api/mitra/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.UpdateMitraProfile)
			c.App.Patch("/api/mitra/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.UpdateMitraProfile)

			// Alias endpoint perusahaan diarahkan ke mitra
			c.App.Get("/api/perusahaan/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.Current)
			c.App.Put("/api/perusahaan/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.UpdateMitraProfile)
			c.App.Patch("/api/perusahaan/profile", c.AuthMiddleware, c.MitraMiddleware, c.UserController.UpdateMitraProfile)

			if c.ApplicationController != nil {
				c.App.Get("/api/mitra/applications", c.AuthMiddleware, c.MitraMiddleware, c.ApplicationController.ListMitra)
				c.App.Patch("/api/mitra/applications/:id/status", c.AuthMiddleware, c.MitraMiddleware, c.ApplicationController.UpdateStatus)
				c.App.Put("/api/mitra/applications/:id/status", c.AuthMiddleware, c.MitraMiddleware, c.ApplicationController.UpdateStatus)
			}
		}

		// Endpoint khusus role Mahasiswa
		if c.MahasiswaMiddleware != nil {
			c.App.Get("/api/mahasiswa/dashboard", c.AuthMiddleware, c.MahasiswaMiddleware, c.UserController.MahasiswaDashboard)
			c.App.Get("/api/mahasiswa/profile", c.AuthMiddleware, c.MahasiswaMiddleware, c.UserController.Current)
			c.App.Put("/api/mahasiswa/profile", c.AuthMiddleware, c.MahasiswaMiddleware, c.UserController.UpdateMahasiswaProfile)
			c.App.Patch("/api/mahasiswa/profile", c.AuthMiddleware, c.MahasiswaMiddleware, c.UserController.UpdateMahasiswaProfile)

			if c.ApplicationController != nil {
				c.App.Post("/api/jobs/:id/apply", c.AuthMiddleware, c.MahasiswaMiddleware, c.ApplicationController.Apply)
				c.App.Post("/api/mahasiswa/applications", c.AuthMiddleware, c.MahasiswaMiddleware, c.ApplicationController.Apply)
				c.App.Get("/api/mahasiswa/applications", c.AuthMiddleware, c.MahasiswaMiddleware, c.ApplicationController.ListMahasiswa)
			}

			// Endpoint cerita diri (khusus role Mahasiswa)
			if c.StoryController != nil {
				c.App.Post("/api/stories", c.AuthMiddleware, c.MahasiswaMiddleware, c.StoryController.Create)
				c.App.Get("/api/stories/me", c.AuthMiddleware, c.MahasiswaMiddleware, c.StoryController.Current)
				c.App.Put("/api/stories/:id", c.AuthMiddleware, c.MahasiswaMiddleware, c.StoryController.Update)
				c.App.Patch("/api/stories/:id", c.AuthMiddleware, c.MahasiswaMiddleware, c.StoryController.Update)
			}
		}
	}
}
