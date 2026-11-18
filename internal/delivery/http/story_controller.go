package http

import (
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

type StoryController struct {
	UseCase *usecase.StoryUseCase
	Log     *logrus.Logger
}

func NewStoryController(useCase *usecase.StoryUseCase, logger *logrus.Logger) *StoryController {
	return &StoryController{
		UseCase: useCase,
		Log:     logger,
	}
}

func (c *StoryController) Create(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Unauthorized: pengguna belum terautentikasi",
			Errors:  "unauthorized",
		})
	}

	request := new(model.CreateStoryRequest)
	if err := ctx.Bind().Body(request); err != nil {
		c.Log.Warnf("Failed to parse create story request body: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Permintaan tidak valid",
			Errors:  err.Error(),
		})
	}

	response, err := c.UseCase.Create(ctx.Context(), userID, request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(model.WebResponse[*model.StoryResponse]{
		Success: true,
		Message: "Cerita berhasil disimpan",
		Data:    response,
	})
}

func (c *StoryController) Current(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Unauthorized: pengguna belum terautentikasi",
			Errors:  "unauthorized",
		})
	}

	response, err := c.UseCase.Current(ctx.Context(), userID)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.StoryResponse]{
		Success: true,
		Message: "Cerita berhasil diambil",
		Data:    response,
	})
}
