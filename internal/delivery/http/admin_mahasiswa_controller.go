package http

import (
	"strconv"

	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

type AdminMahasiswaController struct {
	UseCase *usecase.MahasiswaUseCase
	Log     *logrus.Logger
}

func NewAdminMahasiswaController(useCase *usecase.MahasiswaUseCase, logger *logrus.Logger) *AdminMahasiswaController {
	return &AdminMahasiswaController{
		UseCase: useCase,
		Log:     logger,
	}
}

func (c *AdminMahasiswaController) Create(ctx fiber.Ctx) error {
	request := new(model.CreateMahasiswaRequest)
	if err := ctx.Bind().Body(request); err != nil {
		c.Log.Warnf("Failed to parse create mahasiswa request: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Permintaan tidak valid",
			Errors:  err.Error(),
		})
	}

	response, err := c.UseCase.Create(ctx.Context(), request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusCreated).JSON(model.WebResponse[*model.MahasiswaResponse]{
		Success: true,
		Message: "Mahasiswa berhasil ditambahkan",
		Data:    response,
	})
}

func (c *AdminMahasiswaController) Get(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	response, err := c.UseCase.Get(ctx.Context(), id)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.MahasiswaResponse]{
		Success: true,
		Message: "Berhasil mengambil data mahasiswa",
		Data:    response,
	})
}

func (c *AdminMahasiswaController) List(ctx fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	size, _ := strconv.Atoi(ctx.Query("size", "10"))
	search := ctx.Query("search", "")

	request := &model.SearchMahasiswaRequest{
		Page:   page,
		Size:   size,
		Search: search,
	}

	responses, paging, err := c.UseCase.Search(ctx.Context(), request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[[]*model.MahasiswaResponse]{
		Success: true,
		Message: "Berhasil mengambil daftar mahasiswa",
		Data:    responses,
		Paging:  paging,
	})
}

func (c *AdminMahasiswaController) Update(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	request := new(model.UpdateMahasiswaByAdminRequest)
	if err := ctx.Bind().Body(request); err != nil {
		c.Log.Warnf("Failed to parse update mahasiswa request: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(model.WebResponse[any]{
			Success: false,
			Message: "Permintaan tidak valid",
			Errors:  err.Error(),
		})
	}

	response, err := c.UseCase.Update(ctx.Context(), id, request)
	if err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[*model.MahasiswaResponse]{
		Success: true,
		Message: "Data mahasiswa berhasil diperbarui",
		Data:    response,
	})
}

func (c *AdminMahasiswaController) Delete(ctx fiber.Ctx) error {
	id := ctx.Params("id")
	if err := c.UseCase.Delete(ctx.Context(), id); err != nil {
		return err
	}

	return ctx.Status(fiber.StatusOK).JSON(model.WebResponse[any]{
		Success: true,
		Message: "Data mahasiswa berhasil dihapus",
		Data:    nil,
	})
}
