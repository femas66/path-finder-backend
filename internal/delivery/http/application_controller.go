package http

import (
	"strconv"

	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

type ApplicationController struct {
	UseCase *usecase.ApplicationUseCase
	Log     *logrus.Logger
}

func NewApplicationController(useCase *usecase.ApplicationUseCase, log *logrus.Logger) *ApplicationController {
	return &ApplicationController{
		UseCase: useCase,
		Log:     log,
	}
}

func applicationUserID(ctx fiber.Ctx) (string, error) {
	id, ok := ctx.Locals("user_id").(string)
	if !ok || id == "" {
		return "", fiber.ErrUnauthorized
	}
	return id, nil
}

func (c *ApplicationController) Apply(ctx fiber.Ctx) error {
	userID, err := applicationUserID(ctx)
	if err != nil {
		return err
	}

	request := new(model.ApplyJobRequest)
	if jobID := ctx.Params("id"); jobID != "" {
		request.JobID = jobID
	} else {
		if err := ctx.Bind().Body(request); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
		}
	}

	response, err := c.UseCase.Apply(ctx.Context(), userID, request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(model.WebResponse[*model.ApplicationResponse]{
		Success: true,
		Message: "Pendaftaran magang berhasil dikirim",
		Data:    response,
	})
}

func (c *ApplicationController) ListMahasiswa(ctx fiber.Ctx) error {
	userID, err := applicationUserID(ctx)
	if err != nil {
		return err
	}

	response, err := c.UseCase.ListByMahasiswa(ctx.Context(), userID)
	if err != nil {
		return err
	}

	return ctx.JSON(model.WebResponse[[]model.ApplicationResponse]{
		Success: true,
		Message: "Daftar lamaran berhasil diambil",
		Data:    response,
	})
}

func (c *ApplicationController) ListMitra(ctx fiber.Ctx) error {
	mitraID, err := applicationUserID(ctx)
	if err != nil {
		return err
	}

	page := 1
	size := 10
	if ctx.Request().URI().QueryArgs().Has("page") {
		v, err := strconv.Atoi(ctx.Query("page"))
		if err != nil || v <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "page dan size harus bilangan positif; size maksimal 100")
		}
		page = v
	}
	if ctx.Request().URI().QueryArgs().Has("size") {
		v, err := strconv.Atoi(ctx.Query("size"))
		if err != nil || v <= 0 || v > 100 {
			return fiber.NewError(fiber.StatusBadRequest, "page dan size harus bilangan positif; size maksimal 100")
		}
		size = v
	}

	jobID := ctx.Query("job_id")
	response, paging, err := c.UseCase.ListByMitra(ctx.Context(), mitraID, jobID, page, size)
	if err != nil {
		return err
	}

	return ctx.JSON(model.WebResponse[[]model.ApplicationResponse]{
		Success: true,
		Message: "Daftar pelamar berhasil diambil",
		Data:    response,
		Paging:  paging,
	})
}

func (c *ApplicationController) UpdateStatus(ctx fiber.Ctx) error {
	mitraID, err := applicationUserID(ctx)
	if err != nil {
		return err
	}

	applicationID := ctx.Params("id")
	request := new(model.UpdateApplicationStatusRequest)
	if err := ctx.Bind().Body(request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Permintaan tidak valid")
	}

	response, err := c.UseCase.UpdateStatus(ctx.Context(), mitraID, applicationID, request)
	if err != nil {
		return err
	}

	return ctx.JSON(model.WebResponse[*model.ApplicationResponse]{
		Success: true,
		Message: "Status lamaran berhasil diperbarui",
		Data:    response,
	})
}
