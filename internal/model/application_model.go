package model

import "time"

type ApplyJobRequest struct {
	JobID string `json:"job_id" validate:"required"`
}

type ListMitraApplicationRequest struct {
	JobID string `query:"job_id"`
	Page  int    `query:"page" validate:"min=1"`
	Size  int    `query:"size" validate:"min=1,max=100"`
}

type UpdateApplicationStatusRequest struct {
	Status string `json:"status" validate:"required,oneof=accepted rejected"`
}

type MahasiswaApplicant struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	NIM            string `json:"nim,omitempty"`
	Email          string `json:"email"`
	ProfilePicture string `json:"profile_picture"`
	CV             string `json:"cv"`
	PortfolioURL   string `json:"portfolio_url"`
	Location       string `json:"location,omitempty"`
}

type ApplicationResponse struct {
	ID          string              `json:"id"`
	JobID       string              `json:"job_id"`
	JobTitle    string              `json:"job_title,omitempty"`
	Company     string              `json:"company,omitempty"`
	MahasiswaID string              `json:"mahasiswa_id"`
	Mahasiswa   *MahasiswaApplicant `json:"mahasiswa,omitempty"`
	Status      string              `json:"status"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}
