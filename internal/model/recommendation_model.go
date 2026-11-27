package model

// RecommendationRequest accepts consent only; identity and narrative come from trusted state.
type RecommendationRequest struct {
	ConsentAt string `json:"consent_at"`
}

type MLCandidate struct {
	StudentID     string               `json:"student_id"`
	Narrative     string               `json:"narrative"`
	CVText        *string              `json:"cv_text"`
	ManualProfile *map[string][]string `json:"manual_profile"`
	ConsentAt     string               `json:"consent_at"`
}

type MLJob struct {
	JobID          string   `json:"job_id"`
	Title          string   `json:"title"`
	Category       string   `json:"category"`
	Description    string   `json:"description"`
	RequiredSkills []string `json:"required_skills"`
	OptionalSkills []string `json:"optional_skills"`
	Source         string   `json:"source"`
	SourceURL      string   `json:"source_url"`
	CollectedAt    string   `json:"collected_at"`
	// requirements_text and is_synthetic are unavailable in DB; retain ML defaults.
}

type MLRecommendationRequest struct {
	Candidate MLCandidate `json:"candidate"`
	Jobs      []MLJob     `json:"jobs"`
}

type RecommendationItem struct {
	JobID            string   `json:"job_id"`
	Title            string   `json:"title"`
	MatchingScore    *int     `json:"matching_score"`
	CosineSimilarity *float64 `json:"cosine_similarity"`
	ReadinessScore   *int     `json:"readiness_score"`
	MatchedSkills    []string `json:"matched_skills"`
	MissingSkills    []string `json:"missing_skills"`
	ScoreStatus      string   `json:"score_status"`
	Rank             *int     `json:"rank"`
}

type RecommendationResponse struct {
	StudentID     string               `json:"student_id"`
	GeneratedAt   string               `json:"generated_at"`
	SchemaVersion string               `json:"schema_version"`
	ModelVersion  string               `json:"model_version"`
	TopK          int                  `json:"top_k"`
	Status        string               `json:"status"`
	Items         []RecommendationItem `json:"items"`
}
