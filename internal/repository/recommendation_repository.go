package repository

import (
	"haji-koding/path-finder/internal/entity"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type RecommendationRepository struct {
	Repository[entity.Recommendation]
	Log *logrus.Logger
}

func NewRecommendationRepository(log *logrus.Logger) *RecommendationRepository {
	return &RecommendationRepository{
		Log: log,
	}
}

// FindLatestByUserID mengambil rekomendasi terbaru milik satu pengguna.
func (r *RecommendationRepository) FindLatestByUserID(db *gorm.DB, recommendation *entity.Recommendation, userID string) error {
	return db.Where("user_id = ?", userID).Order("created_at DESC").Take(recommendation).Error
}
