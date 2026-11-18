package http

import (
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

type RecommendationController struct {
	UseCase *usecase.RecommendationUseCase
	Log     *logrus.Logger
}

func NewRecommendationController(useCase *usecase.RecommendationUseCase, logger *logrus.Logger) *RecommendationController {
	return &RecommendationController{
		UseCase: useCase,
		Log:     logger,
	}
}

// Create menjalankan analisis cerita dan mengembalikan rekomendasi.
func (c *RecommendationController) Create(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Unauthorized: pengguna belum terautentikasi",
			Errors:  "unauthorized",
		})
	}

	request := new(model.CreateRecommendationRequest)

	// Body bersifat opsional: tanpa body, server memakai cerita terakhir.
	if len(ctx.Body()) > 0 {
		if err := ctx.Bind().Body(request); err != nil {
			c.Log.Warnf("Failed to parse recommendation request body: %v", err)
			return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
				Success: false,
				Message: "Permintaan tidak valid",
				Errors:  err.Error(),
			})
		}
	}

	response, err := c.UseCase.Generate(ctx.Context(), userID, request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.RecommendationResponse]{
		Success: true,
		Message: "Rekomendasi berhasil dibuat",
		Data:    response,
	})
}

// Latest mengembalikan rekomendasi terakhir milik pengguna.
func (c *RecommendationController) Latest(ctx fiber.Ctx) error {
	userID, ok := ctx.Locals("user_id").(string)
	if !ok || userID == "" {
		return ctx.Status(fiber.StatusUnauthorized).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Unauthorized: pengguna belum terautentikasi",
			Errors:  "unauthorized",
		})
	}

	response, err := c.UseCase.Latest(ctx.Context(), userID)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.RecommendationResponse]{
		Success: true,
		Message: "Rekomendasi berhasil diambil",
		Data:    response,
	})
}
