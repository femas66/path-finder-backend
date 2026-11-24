package model

import "time"

type CreateJobRequest struct {
	Status         string   `json:"status" validate:"omitempty,oneof=aktif ditutup"`
	Title          string   `json:"title" validate:"required,min=3,max=150"`
	Category       string   `json:"category" validate:"required"`
	Description    string   `json:"description" validate:"required,min=20,max=10000"`
	Location       string   `json:"location" validate:"required,max=255"`
	WorkType       string   `json:"work_type" validate:"required,max=50"`
	RequiredSkills []string `json:"required_skills" validate:"required,min=1,max=50,dive,required,max=100"`
	OptionalSkills []string `json:"optional_skills" validate:"max=50,dive,required,max=100"`
	SourceURL      string   `json:"source_url" validate:"max=1000"`
}

// PUT and PATCH both update only non-null fields. Omitted and null fields are unchanged.
// Empty optional_skills and source_url clear their values; required fields cannot be empty.
type UpdateJobRequest struct {
	Status         *string   `json:"status" validate:"omitempty,oneof=aktif ditutup"`
	Title          *string   `json:"title" validate:"omitempty,min=3,max=150"`
	Category       *string   `json:"category" validate:"omitempty,min=1"`
	Description    *string   `json:"description" validate:"omitempty,min=20,max=10000"`
	Location       *string   `json:"location" validate:"omitempty,min=1,max=255"`
	WorkType       *string   `json:"work_type" validate:"omitempty,min=1,max=50"`
	RequiredSkills *[]string `json:"required_skills" validate:"omitempty,min=1,max=50,dive,required,max=100"`
	OptionalSkills *[]string `json:"optional_skills" validate:"omitempty,max=50,dive,required,max=100"`
	SourceURL      *string   `json:"source_url" validate:"omitempty,max=1000"`
}

type ListJobRequest struct {
	Status   string `json:"status" validate:"omitempty,oneof=aktif ditutup"`
	Page     int    `json:"page" validate:"min=1"`
	Size     int    `json:"size" validate:"min=1,max=100"`
	Search   string `json:"search"`
	Category string `json:"category"`
}

type JobResponse struct {
	Status         string    `json:"status"`
	ID             string    `json:"id"`
	MitraID        *string   `json:"mitra_id"`
	Title          string    `json:"title"`
	Category       string    `json:"category"`
	Company        string    `json:"company"`
	Description    string    `json:"description"`
	Location       string    `json:"location"`
	WorkType       string    `json:"work_type"`
	Source         string    `json:"source"`
	SourceURL      string    `json:"source_url"`
	RequiredSkills []string  `json:"required_skills"`
	OptionalSkills []string  `json:"optional_skills"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
