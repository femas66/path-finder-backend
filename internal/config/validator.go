package config

import (
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
)

func NewValidator() *validator.Validate {
	validate := validator.New()

	_ = validate.RegisterValidation("http_url", func(fl validator.FieldLevel) bool {
		val := strings.TrimSpace(fl.Field().String())
		if val == "" {
			return true
		}
		parsed, err := url.ParseRequestURI(val)
		return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != ""
	})

	return validate
}
