package model

import "time"

type CreateStoryRequest struct {
	Content string `json:"content" validate:"required,min=50,max=2000"`
}

type StoryResponse struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	Content        string    `json:"content"`
	WordCount      int       `json:"word_count"`
	AnalysisStatus string    `json:"analysis_status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
