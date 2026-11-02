package converter

import (
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
)

func UserToMahasiswaResponse(user *entity.User) *model.MahasiswaResponse {
	return &model.MahasiswaResponse{
		ID:             user.ID,
		NIM:            user.NIM,
		Name:           user.Name,
		Email:          user.Email,
		Role:           user.Role,
		ProfilePicture: user.ProfilePicture,
		CV:             user.CV,
		PortfolioURL:   user.PortfolioURL,
		CreatedAt:      user.CreatedAt,
		UpdatedAt:      user.UpdatedAt,
	}
}

func UsersToMahasiswaResponses(users []entity.User) []*model.MahasiswaResponse {
	responses := make([]*model.MahasiswaResponse, len(users))
	for i, u := range users {
		responses[i] = UserToMahasiswaResponse(&u)
	}
	return responses
}
