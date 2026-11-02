package http

import (
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

type UserController struct {
	UseCase *usecase.UserUseCase
	Log     *logrus.Logger
}

func NewUserController(useCase *usecase.UserUseCase, logger *logrus.Logger) *UserController {
	return &UserController{
		UseCase: useCase,
		Log:     logger,
	}
}

func (c *UserController) Register(ctx fiber.Ctx) error {
	request := new(model.RegisterUserRequest)
	if err := ctx.Bind().Body(request); err != nil {
		c.Log.Warnf("Failed to parse register request body: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Permintaan tidak valid",
			Errors:  err.Error(),
		})
	}

	response, err := c.UseCase.Register(ctx.Context(), request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(model.WebResponse[*model.UserResponse]{
		Success: true,
		Message: "Registrasi berhasil",
		Data:    response,
	})
}

func (c *UserController) Login(ctx fiber.Ctx) error {
	request := new(model.LoginUserRequest)
	if err := ctx.Bind().Body(request); err != nil {
		c.Log.Warnf("Failed to parse login request body: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Permintaan tidak valid",
			Errors:  err.Error(),
		})
	}

	response, err := c.UseCase.Login(ctx.Context(), request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.LoginResponse]{
		Success: true,
		Message: "Login berhasil",
		Data:    response,
	})
}

func (c *UserController) ForgotPassword(ctx fiber.Ctx) error {
	request := new(model.ForgotPasswordRequest)
	if err := ctx.Bind().Body(request); err != nil {
		c.Log.Warnf("Failed to parse forgot password request body: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Permintaan tidak valid",
			Errors:  err.Error(),
		})
	}

	response, err := c.UseCase.ForgotPassword(ctx.Context(), request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.ForgotPasswordResponse]{
		Success: true,
		Message: "Kode verifikasi telah dikirim ke email",
		Data:    response,
	})
}

func (c *UserController) Current(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Unauthorized",
			Errors:  "unauthorized",
		})
	}

	response, err := c.UseCase.Current(ctx.Context(), userID)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.UserResponse]{
		Success: true,
		Message: "Berhasil mengambil data pengguna",
		Data:    response,
	})
}

func (c *UserController) AdminDashboard(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[fiber.Map]{
		Success: true,
		Message: "Selamat datang di Dashboard Admin",
		Data: fiber.Map{
			"user_id": ctx.Locals("user_id"),
			"role":    ctx.Locals("role"),
			"email":   ctx.Locals("email"),
		},
	})
}

func (c *UserController) MitraDashboard(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[fiber.Map]{
		Success: true,
		Message: "Selamat datang di Dashboard Mitra / Perusahaan",
		Data: fiber.Map{
			"user_id": ctx.Locals("user_id"),
			"role":    ctx.Locals("role"),
			"email":   ctx.Locals("email"),
		},
	})
}

func (c *UserController) MahasiswaDashboard(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[fiber.Map]{
		Success: true,
		Message: "Selamat datang di Dashboard Mahasiswa",
		Data: fiber.Map{
			"user_id": ctx.Locals("user_id"),
			"role":    ctx.Locals("role"),
			"email":   ctx.Locals("email"),
		},
	})
}

func (c *UserController) UpdateCompanyProfile(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Unauthorized",
			Errors:  "unauthorized",
		})
	}

	request := new(model.UpdateCompanyProfileRequest)
	if err := ctx.Bind().Body(request); err != nil {
		c.Log.Warnf("Failed to parse update company profile request body: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Permintaan tidak valid",
			Errors:  err.Error(),
		})
	}

	response, err := c.UseCase.UpdateCompanyProfile(ctx.Context(), userID, request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.UserResponse]{
		Success: true,
		Message: "Profil perusahaan berhasil diperbarui",
		Data:    response,
	})
}

func (c *UserController) UpdateMahasiswaProfile(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Unauthorized",
			Errors:  "unauthorized",
		})
	}

	request := new(model.UpdateMahasiswaProfileRequest)
	if err := ctx.Bind().Body(request); err != nil {
		c.Log.Warnf("Failed to parse update mahasiswa profile request body: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Permintaan tidak valid",
			Errors:  err.Error(),
		})
	}

	response, err := c.UseCase.UpdateMahasiswaProfile(ctx.Context(), userID, request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.UserResponse]{
		Success: true,
		Message: "Profil mahasiswa berhasil diperbarui",
		Data:    response,
	})
}
