package test

import (
	"testing"
	"time"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model/converter"

	"github.com/stretchr/testify/assert"
)

func TestUserConverter(t *testing.T) {
	now := time.Now()
	user := &entity.User{
		ID:        "user-uuid-123",
		Name:      "Femas Akbar",
		Email:     "femas@example.com",
		Password:  "hashed_secret",
		Role:      entity.RoleMahasiswa,
		CreatedAt: now,
		UpdatedAt: now,
	}

	t.Run("UserToResponse", func(t *testing.T) {
		res := converter.UserToResponse(user)
		assert.Equal(t, "user-uuid-123", res.ID)
		assert.Equal(t, "Femas Akbar", res.Name)
		assert.Equal(t, "femas@example.com", res.Email)
		assert.Equal(t, entity.RoleMahasiswa, res.Role)
		assert.Equal(t, now, res.CreatedAt)
	})

	t.Run("UserToLoginResponse", func(t *testing.T) {
		token := "mock.jwt.token"
		res := converter.UserToLoginResponse(user, token)
		assert.Equal(t, token, res.Token)
		assert.Equal(t, "user-uuid-123", res.User.ID)
		assert.Equal(t, "Femas Akbar", res.User.Name)
		assert.Equal(t, "femas@example.com", res.User.Email)
		assert.Equal(t, entity.RoleMahasiswa, res.User.Role)
	})
}
