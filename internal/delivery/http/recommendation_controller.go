package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/service"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

type RecommendationController struct {
	UseCase *usecase.RecommendationUseCase
	Log     *logrus.Logger
}

func NewRecommendationController(useCase *usecase.RecommendationUseCase, logger *logrus.Logger) *RecommendationController {
	return &RecommendationController{UseCase: useCase, Log: logger}
}

func (c *RecommendationController) Recommend(ctx fiber.Ctx) error {
	fail := func(code int, message string) error {
		return ctx.Status(code).JSON(model.WebResponse[model.RecommendationResponse]{Success: false, Message: message, Errors: message})
	}
	id, _ := ctx.Locals("user_id").(string)
	if strings.TrimSpace(id) == "" {
		return fail(401, "Unauthorized")
	}
	role, _ := ctx.Locals("role").(string)
	if !strings.EqualFold(role, entity.RoleMahasiswa) {
		return fail(403, "Forbidden")
	}
	body := ctx.Body()
	if len(body) > service.MLPayloadLimit {
		return fail(413, "Request exceeds 1 MiB")
	}
	// Token-level parsing rejects duplicates, unknown fields, null, and trailing JSON.
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return fail(400, "Invalid recommendation request")
	}
	request := new(model.RecommendationRequest)
	found := false
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil || key != "consent_at" || found {
			return fail(400, "Invalid recommendation request")
		}
		value, err := decoder.Token()
		text, ok := value.(string)
		if err != nil || !ok {
			return fail(400, "Invalid recommendation request")
		}
		request.ConsentAt, found = text, true
	}
	if _, err := decoder.Token(); err != nil || !found || decoder.Decode(new(any)) != io.EOF {
		return fail(400, "Invalid recommendation request")
	}
	response, err := c.UseCase.Recommend(ctx.Context(), id, request)
	if err != nil {
		var public *fiber.Error
		if errors.As(err, &public) {
			return fail(public.Code, public.Message)
		}
		return fail(500, "Recommendation failed")
	}
	return ctx.Status(200).JSON(model.WebResponse[model.RecommendationResponse]{Success: true, Message: "Recommendation generated", Data: *response})
}
