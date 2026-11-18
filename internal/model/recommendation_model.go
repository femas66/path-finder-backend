package model

// CreateRecommendationRequest adalah body opsional saat meminta rekomendasi.
//
// StoryID kosong berarti server memakai cerita terakhir milik pengguna.
// TopK kosong berarti server memakai nilai default 3.
type CreateRecommendationRequest struct {
	StoryID string `json:"story_id"`
	TopK    *int   `json:"top_k"`
}

// RecommendationProfileResponse menampilkan keahlian yang berhasil dikenali
// dari cerita pengguna.
type RecommendationProfileResponse struct {
	HardSkills    []string `json:"hard_skills"`
	SoftSkills    []string `json:"soft_skills"`
	Interests     []string `json:"interests"`
	ProfileStatus string   `json:"profile_status"`
}

// RecommendationItemResponse mengikuti RecommendationItem kontrak ML,
// ditambah field presentasi lowongan.
type RecommendationItemResponse struct {
	JobID            string   `json:"job_id"`
	Title            string   `json:"title"`
	Category         string   `json:"category"`
	Company          string   `json:"company"`
	Location         string   `json:"location"`
	WorkType         string   `json:"work_type"`
	Source           string   `json:"source"`
	SourceURL        string   `json:"source_url"`
	MatchingScore    *int     `json:"matching_score"`
	CosineSimilarity *float64 `json:"cosine_similarity"`
	ReadinessScore   *int     `json:"readiness_score"`
	MatchedSkills    []string `json:"matched_skills"`
	MissingSkills    []string `json:"missing_skills"`
	ScoreStatus      string   `json:"score_status"`
	Rank             *int     `json:"rank"`
}

// RecommendationResponse mengikuti RecommendationResponse kontrak ML.
type RecommendationResponse struct {
	ID            string                        `json:"id"`
	StudentID     string                        `json:"student_id"`
	StoryID       string                        `json:"story_id"`
	GeneratedAt   string                        `json:"generated_at"`
	SchemaVersion string                        `json:"schema_version"`
	ModelVersion  string                        `json:"model_version"`
	TopK          int                           `json:"top_k"`
	Status        string                        `json:"status"`
	Profile       RecommendationProfileResponse `json:"profile"`
	Items         []RecommendationItemResponse  `json:"items"`
}
