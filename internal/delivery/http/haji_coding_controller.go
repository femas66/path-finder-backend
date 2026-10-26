package http

import (
	"haji-koding/path-finder/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

type HajiCodingController struct {
	Log *logrus.Logger
}

func NewHajiCodingController(logger *logrus.Logger) *HajiCodingController {
	return &HajiCodingController{
		Log: logger,
	}
}

func (c *HajiCodingController) HaiBos(ctx fiber.Ctx) error {
	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[string]{
		Success: true,
		Data:    "HALO JURAGAN",
		Message: "juragansengon88",
	})
}
