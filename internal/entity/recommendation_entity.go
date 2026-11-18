package entity

import "time"

const (
	// RecommendationStatusOK berarti semua item punya skor lengkap.
	RecommendationStatusOK = "ok"
	// RecommendationStatusParsial berarti sebagian item belum bisa diberi skor.
	RecommendationStatusParsial = "parsial"
	// RecommendationStatusProfileIncomplete berarti tidak ada keahlian terdeteksi.
	RecommendationStatusProfileIncomplete = "profil_tidak_lengkap"

	// ProfileStatusLengkap berarti profil punya minimal satu hard/soft skill.
	ProfileStatusLengkap = "lengkap"
	// ProfileStatusTanpaSkill berarti tidak ada keahlian yang terdeteksi.
	ProfileStatusTanpaSkill = "tanpa_skill"

	// ScoreStatusOK berarti matching_score terisi.
	ScoreStatusOK = "ok"
	// ScoreStatusBelumTersedia berarti skor belum bisa dihitung.
	ScoreStatusBelumTersedia = "skor_belum_tersedia"
	// ScoreStatusPersyaratanBelumTersedia berarti lowongan tanpa skill wajib.
	ScoreStatusPersyaratanBelumTersedia = "persyaratan_belum_tersedia"
	// ScoreStatusProfilTanpaKeterampilan berarti profil tanpa keahlian.
	ScoreStatusProfilTanpaKeterampilan = "profil_tanpa_keterampilan"
)

// RecommendationItem adalah satu lowongan beserta skornya.
//
// Bentuk field mengikuti RecommendationItem pada kontrak ML
// (docs/architecture/data-contract.md) dan ditambah field presentasi lowongan
// yang dibutuhkan aplikasi (company, location, work_type, source).
type RecommendationItem struct {
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

// Recommendation menyimpan satu hasil analisis agar bisa dibuka kembali
// tanpa menghitung ulang.
type Recommendation struct {
	ID            string               `gorm:"column:id;primaryKey"`
	UserID        string               `gorm:"column:user_id;index;not null"`
	StoryID       string               `gorm:"column:story_id;index;not null"`
	Status        string               `gorm:"column:status;not null"`
	SchemaVersion string               `gorm:"column:schema_version;not null"`
	ModelVersion  string               `gorm:"column:model_version;not null"`
	TopK          int                  `gorm:"column:top_k;not null"`
	ProfileStatus string               `gorm:"column:profile_status;not null"`
	HardSkills    []string             `gorm:"column:hard_skills;serializer:json;type:text"`
	SoftSkills    []string             `gorm:"column:soft_skills;serializer:json;type:text"`
	Interests     []string             `gorm:"column:interests;serializer:json;type:text"`
	Items         []RecommendationItem `gorm:"column:items;serializer:json;type:text"`
	GeneratedAt   time.Time            `gorm:"column:generated_at;not null"`
	CreatedAt     time.Time            `gorm:"column:created_at;autoCreateTime"`
}

func (r *Recommendation) TableName() string {
	return "recommendations"
}
