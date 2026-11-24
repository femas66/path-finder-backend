package http

import (
	"strconv"

	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

type JobController struct {
	UseCase *usecase.JobUseCase
	Log     *logrus.Logger
}

func NewJobController(useCase *usecase.JobUseCase, log *logrus.Logger) *JobController {
	return &JobController{UseCase: useCase, Log: log}
}

func jobUserID(ctx fiber.Ctx) (string, error) {
	id, ok := ctx.Locals("user_id").(string)
	if !ok || id == "" {
		return "", fiber.ErrUnauthorized
	}
	return id, nil
}

func (c *JobController) Create(ctx fiber.Ctx) error {
	id, err := jobUserID(ctx)
	if err != nil {
		return err
	}
	request := new(model.CreateJobRequest)
	if err := ctx.Bind().Body(request); err != nil {
		return fiber.NewError(400, "Permintaan tidak valid")
	}
	response, err := c.UseCase.Create(ctx.Context(), id, request)
	if err != nil {
		return err
	}
	return ctx.Status(fiber.StatusCreated).JSON(model.WebResponse[*model.JobResponse]{Success: true, Message: "Lowongan berhasil disimpan", Data: response})
}

func (c *JobController) List(ctx fiber.Ctx) error    { return c.list(ctx, false) }
func (c *JobController) ListOwn(ctx fiber.Ctx) error { return c.list(ctx, true) }

func (c *JobController) list(ctx fiber.Ctx, own bool) error {
	id, err := jobUserID(ctx)
	if err != nil {
		return err
	}
	request := &model.ListJobRequest{Page: 1, Size: 10, Search: ctx.Query("search"), Category: ctx.Query("category"), Status: ctx.Query("status")}
	for key, target := range map[string]*int{"page": &request.Page, "size": &request.Size} {
		if ctx.Request().URI().QueryArgs().Has(key) {
			value, err := strconv.Atoi(ctx.Query(key))
			if err != nil || value <= 0 || (key == "size" && value > 100) {
				return fiber.NewError(400, "page dan size harus bilangan positif; size maksimal 100")
			}
			*target = value
		}
	}
	var owner *string
	if own {
		owner = &id
	}
	response, paging, err := c.UseCase.List(ctx.Context(), request, owner)
	if err != nil {
		return err
	}
	return ctx.JSON(model.WebResponse[[]model.JobResponse]{Success: true, Message: "Lowongan berhasil diambil", Data: response, Paging: paging})
}

func (c *JobController) Get(ctx fiber.Ctx) error    { return c.get(ctx, false) }
func (c *JobController) GetOwn(ctx fiber.Ctx) error { return c.get(ctx, true) }

func (c *JobController) get(ctx fiber.Ctx, own bool) error {
	id, err := jobUserID(ctx)
	if err != nil {
		return err
	}
	var owner *string
	if own {
		owner = &id
	}
	response, err := c.UseCase.Get(ctx.Context(), ctx.Params("id"), owner)
	if err != nil {
		return err
	}
	return ctx.JSON(model.WebResponse[*model.JobResponse]{Success: true, Message: "Lowongan berhasil diambil", Data: response})
}

func (c *JobController) Update(ctx fiber.Ctx) error {
	id, err := jobUserID(ctx)
	if err != nil {
		return err
	}
	request := new(model.UpdateJobRequest)
	if err := ctx.Bind().Body(request); err != nil {
		return fiber.NewError(400, "Permintaan tidak valid")
	}
	response, err := c.UseCase.Update(ctx.Context(), ctx.Params("id"), id, request)
	if err != nil {
		return err
	}
	return ctx.JSON(model.WebResponse[*model.JobResponse]{Success: true, Message: "Lowongan berhasil diperbarui", Data: response})
}

func (c *JobController) Delete(ctx fiber.Ctx) error {
	id, err := jobUserID(ctx)
	if err != nil {
		return err
	}
	if err := c.UseCase.Delete(ctx.Context(), ctx.Params("id"), id); err != nil {
		return err
	}
	return ctx.JSON(model.WebResponse[any]{Success: true, Message: "Lowongan berhasil dihapus", Data: nil})
}
