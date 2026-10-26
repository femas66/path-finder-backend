package config

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
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

			err = ctx.Status(code).SendFile(fmt.Sprintf("./%d.html", code))
			if err != nil {
				return ctx.Status(code).JSON(fiber.Map{
					"success": false,
					"error":   err.Error(),
				})
			}

			return nil
		},
	})

	return app
}
