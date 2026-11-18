package entity

import "time"

const (
	StoryAnalysisPending = "pending"
	// StoryAnalysisDone menandai cerita sudah dianalisis mesin rekomendasi.
	StoryAnalysisDone = "done"
)

type Story struct {
	ID             string    `gorm:"column:id;primaryKey"`
	UserID         string    `gorm:"column:user_id;index;not null"`
	Content        string    `gorm:"column:content;type:text;not null"`
	WordCount      int       `gorm:"column:word_count;not null;default:0"`
	AnalysisStatus string    `gorm:"column:analysis_status;not null;default:'pending'"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time `gorm:"column:updated_at;autoCreateTime;autoUpdateTime"`
}

func (s *Story) TableName() string {
	return "stories"
}
