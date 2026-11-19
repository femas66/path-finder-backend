package repository

import (
	"haji-koding/path-finder/internal/entity"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type StoryRepository struct {
	Repository[entity.Story]
	Log *logrus.Logger
}

func NewStoryRepository(log *logrus.Logger) *StoryRepository {
	return &StoryRepository{
		Log: log,
	}
}

func (r *StoryRepository) FindAllByUserID(db *gorm.DB, stories *[]entity.Story, userID string) error {
	return db.Where("user_id = ?", userID).Order("created_at DESC").Find(stories).Error
}

func (r *StoryRepository) FindLatestByUserID(db *gorm.DB, story *entity.Story, userID string) error {
	return db.Where("user_id = ?", userID).Order("created_at DESC, id DESC").Take(story).Error
}

func (r *StoryRepository) Update(db *gorm.DB, story *entity.Story) error {
	return db.Model(story).
		Select("content", "word_count", "analysis_status", "updated_at").
		Updates(story).Error
}

// FindByIDAndUserID mengambil satu cerita tertentu milik pengguna.
// Dipakai agar pengguna tidak bisa menganalisis cerita milik orang lain.
func (r *StoryRepository) FindByIDAndUserID(db *gorm.DB, story *entity.Story, id string, userID string) error {
	return db.Where("id = ? AND user_id = ?", id, userID).Take(story).Error
}

// UpdateAnalysisStatus mengubah status analisis sebuah cerita.
func (r *StoryRepository) UpdateAnalysisStatus(db *gorm.DB, storyID string, status string) error {
	return db.Model(&entity.Story{}).Where("id = ?", storyID).Update("analysis_status", status).Error
}

func (r *StoryRepository) CountByUserID(db *gorm.DB, userID string) (int64, error) {
	var total int64
	err := db.Model(&entity.Story{}).Where("user_id = ?", userID).Count(&total).Error
	return total, err
}
