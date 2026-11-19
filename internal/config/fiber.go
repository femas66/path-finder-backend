package config

import (
	"errors"

	"haji-koding/path-finder/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func NewFiber(config *viper.Viper) *fiber.App {
	var app = fiber.New(fiber.Config{
		AppName: config.GetString("app.name"),
		ErrorHandler: func(ctx fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError

			var e *fiber.Error
			if errors.As(err, &e) && e != nil {
				code = e.Code
			}

			message := "terjadi kesalahan pada server"
			if code < 500 {
				message = err.Error()
			} else {
				logrus.WithError(err).Error("Internal server error occurred")
			}

			return ctx.Status(code).JSON(model.WebResponse[any]{
				Success: false,
				Message: message,
				Errors:  message,
			})
		},
	})

	return app
}
