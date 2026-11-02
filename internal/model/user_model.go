package model

import "time"

type RegisterUserRequest struct {
	Name            string `json:"name" validate:"required,min=3,max=100"`
	Email           string `json:"email" validate:"required,email,max=100"`
	Password        string `json:"password" validate:"required,min=6,max=100"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
	Role            string `json:"role" validate:"omitempty,oneof=admin mitra perusahaan mahasiswa"`
}

type LoginUserRequest struct {
	Email    string `json:"email" validate:"required,email,max=100"`
	Password string `json:"password" validate:"required,min=6,max=100"`
}

type UserResponse struct {
	ID             string    `json:"id"`
	NIM            string    `json:"nim,omitempty"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Role           string    `json:"role"`
	ProfilePicture string    `json:"profile_picture,omitempty"`
	Location       string    `json:"location,omitempty"`
	CV             string    `json:"cv,omitempty"`
	PortfolioURL   string    `json:"portfolio_url,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UpdateCompanyProfileRequest struct {
	Name           string `json:"name" validate:"omitempty,min=2,max=150"`
	NamaPerusahaan string `json:"nama_perusahaan" validate:"omitempty,min=2,max=150"`
	Location       string `json:"location" validate:"omitempty,max=255"`
	Lokasi         string `json:"lokasi" validate:"omitempty,max=255"`
	ProfilePicture string `json:"profile_picture" validate:"omitempty,max=500"`
	FotoProfile    string `json:"foto_profile" validate:"omitempty,max=500"`
}

func (r *UpdateCompanyProfileRequest) GetName() string {
	if r.NamaPerusahaan != "" {
		return r.NamaPerusahaan
	}
	return r.Name
}

func (r *UpdateCompanyProfileRequest) GetLocation() string {
	if r.Lokasi != "" {
		return r.Lokasi
	}
	return r.Location
}

func (r *UpdateCompanyProfileRequest) GetProfilePicture() string {
	if r.FotoProfile != "" {
		return r.FotoProfile
	}
	return r.ProfilePicture
}

type UpdateMahasiswaProfileRequest struct {
	ProfilePicture string `json:"profile_picture" validate:"omitempty,max=500"`
	FotoProfile    string `json:"foto_profile" validate:"omitempty,max=500"`
	CV             string `json:"cv" validate:"omitempty,max=500"`
	PortfolioURL   string `json:"portfolio_url" validate:"omitempty,max=500"`
	LinkPortofolio string `json:"link_portofolio" validate:"omitempty,max=500"`
}

func (r *UpdateMahasiswaProfileRequest) GetProfilePicture() string {
	if r.FotoProfile != "" {
		return r.FotoProfile
	}
	return r.ProfilePicture
}

func (r *UpdateMahasiswaProfileRequest) GetCV() string {
	return r.CV
}

func (r *UpdateMahasiswaProfileRequest) GetPortfolioURL() string {
	if r.LinkPortofolio != "" {
		return r.LinkPortofolio
	}
	return r.PortfolioURL
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email,max=100"`
}

type ForgotPasswordResponse struct {
	Email string `json:"email"`
}

type VerifyUserRequest struct {
	Token string `validate:"required,max=100"`
}
