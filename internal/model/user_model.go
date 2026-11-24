package model

import "time"

type RegisterUserRequest struct {
	Name            string `json:"name" validate:"required,min=3,max=100"`
	Email           string `json:"email" validate:"required,email,max=100"`
	Password        string `json:"password" validate:"required,min=6,max=100"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
	Role            string `json:"role" validate:"omitempty,oneof=mitra mahasiswa"`
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
	FCMToken       string    `json:"fcm_token,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UpdateMitraProfileRequest struct {
	Name           string `json:"name" validate:"omitempty,min=2,max=150"`
	NamaMitra      string `json:"nama_mitra" validate:"omitempty,min=2,max=150"`
	NamaPerusahaan string `json:"nama_perusahaan" validate:"omitempty,min=2,max=150"`
	Location       string `json:"location" validate:"omitempty,max=255"`
	Lokasi         string `json:"lokasi" validate:"omitempty,max=255"`
	ProfilePicture string `json:"profile_picture" validate:"omitempty,http_url,max=500"`
	FotoProfile    string `json:"foto_profile" validate:"omitempty,http_url,max=500"`
}

func (r *UpdateMitraProfileRequest) GetName() string {
	if r.NamaMitra != "" {
		return r.NamaMitra
	}
	if r.NamaPerusahaan != "" {
		return r.NamaPerusahaan
	}
	return r.Name
}

func (r *UpdateMitraProfileRequest) GetLocation() string {
	if r.Lokasi != "" {
		return r.Lokasi
	}
	return r.Location
}

func (r *UpdateMitraProfileRequest) GetProfilePicture() string {
	if r.FotoProfile != "" {
		return r.FotoProfile
	}
	return r.ProfilePicture
}

type UpdateCompanyProfileRequest = UpdateMitraProfileRequest

type UpdateMahasiswaProfileRequest struct {
	Location       string `json:"location" validate:"omitempty,max=255"`
	Lokasi         string `json:"lokasi" validate:"omitempty,max=255"`
	ProfilePicture string `json:"profile_picture" validate:"omitempty,http_url,max=500"`
	FotoProfile    string `json:"foto_profile" validate:"omitempty,http_url,max=500"`
	CV             string `json:"cv" validate:"omitempty,http_url,max=500"`
	PortfolioURL   string `json:"portfolio_url" validate:"omitempty,http_url,max=500"`
	LinkPortofolio string `json:"link_portofolio" validate:"omitempty,http_url,max=500"`
}

func (r *UpdateMahasiswaProfileRequest) GetLocation() string {
	if r.Lokasi != "" {
		return r.Lokasi
	}
	return r.Location
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

type VerifyResetCodeRequest struct {
	Email string `json:"email" validate:"required,email,max=100"`
	Code  string `json:"code" validate:"required,len=6"`
}

type VerifyResetCodeResponse struct {
	Email string `json:"email"`
	Valid bool   `json:"valid"`
}

type ResetPasswordRequest struct {
	Email           string `json:"email" validate:"required,email,max=100"`
	Code            string `json:"code" validate:"required,len=6"`
	Password        string `json:"password" validate:"required,min=6,max=100"`
	ConfirmPassword string `json:"confirm_password" validate:"required,eqfield=Password"`
}

type VerifyUserRequest struct {
	Token string `validate:"required,max=100"`
}

type UpdateFCMTokenRequest struct {
	FCMToken string `json:"fcm_token" validate:"required,min=10,max=500"`
}
