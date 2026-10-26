package converter

import (
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
)

func UserToResponse(user *entity.User) *model.UserResponse {
	return &model.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func UserToLoginResponse(user *entity.User, token string) *model.LoginResponse {
	return &model.LoginResponse{
		Token: token,
		User:  *UserToResponse(user),
	}
}
