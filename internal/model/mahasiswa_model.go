package model

import "time"

type CreateMahasiswaRequest struct {
	Name           string `json:"name" validate:"required,min=3,max=100"`
	Email          string `json:"email" validate:"required,email,max=100"`
	NIM            string `json:"nim" validate:"required,min=5,max=50"`
	ProfilePicture string `json:"profile_picture" validate:"omitempty,max=500"`
	CV             string `json:"cv" validate:"omitempty,max=500"`
	PortfolioURL   string `json:"portfolio_url" validate:"omitempty,max=500"`
}

type UpdateMahasiswaByAdminRequest struct {
	Name           string `json:"name" validate:"omitempty,min=3,max=100"`
	Email          string `json:"email" validate:"omitempty,email,max=100"`
	NIM            string `json:"nim" validate:"omitempty,min=5,max=50"`
	Password       string `json:"password" validate:"omitempty,min=6,max=100"`
	ProfilePicture string `json:"profile_picture" validate:"omitempty,max=500"`
	CV             string `json:"cv" validate:"omitempty,max=500"`
	PortfolioURL   string `json:"portfolio_url" validate:"omitempty,max=500"`
}

type MahasiswaResponse struct {
	ID             string    `json:"id"`
	NIM            string    `json:"nim"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	ProfilePicture string    `json:"profile_picture,omitempty"`
	CV             string    `json:"cv,omitempty"`
	PortfolioURL   string    `json:"portfolio_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type SearchMahasiswaRequest struct {
	Page   int    `query:"page"`
	Size   int    `query:"size"`
	Search string `query:"search"`
}
