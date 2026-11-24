package config

import (
	"errors"
	"fmt"
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
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" {
			return false
		}
		hostname := strings.ToLower(parsed.Hostname())
		// Blokir loopback dan metadata endpoint untuk mencegah SSRF
		if hostname == "localhost" || hostname == "127.0.0.1" || hostname == "0.0.0.0" || hostname == "169.254.169.254" {
			return false
		}
		return true
	})

	return validate
}

func FormatValidationError(err error) string {
	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		var msgs []string
		for _, fe := range valErrs {
			field := strings.ToLower(fe.Field())
			msgs = append(msgs, fmt.Sprintf("bidang '%s' tidak memenuhi aturan '%s'", field, fe.Tag()))
		}
		return strings.Join(msgs, "; ")
	}
	return err.Error()
}
