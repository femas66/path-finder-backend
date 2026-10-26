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
