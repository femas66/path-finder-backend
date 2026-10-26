package test

import (
	"haji-koding/path-finder/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewViper(t *testing.T) {
	viperConfig := config.NewViper()

	assert.NotNil(t, viperConfig)
	assert.Equal(t, "path-finder-backend", viperConfig.GetString("app.name"))
	assert.Equal(t, 3000, viperConfig.GetInt("web.port"))
	assert.NotEmpty(t, viperConfig.GetString("database.host"))
	assert.Equal(t, 5432, viperConfig.GetInt("database.port"))
}

func TestNewLogger(t *testing.T) {
	log := config.NewLogger()

	assert.NotNil(t, log)
}

func TestNewValidator(t *testing.T) {
	validate := config.NewValidator()

	assert.NotNil(t, validate)

	type SampleUser struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	validUser := SampleUser{
		Name:  "Femas",
		Email: "femas@example.com",
	}
	err := validate.Struct(validUser)
	assert.NoError(t, err)

	invalidUser := SampleUser{
		Name:  "",
		Email: "not-an-email",
	}
	err = validate.Struct(invalidUser)
	assert.Error(t, err)
}

func TestNewFiber(t *testing.T) {
	viperConfig := config.NewViper()
	app := config.NewFiber(viperConfig)

	assert.NotNil(t, app)
}
