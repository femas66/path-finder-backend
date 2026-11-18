package repository

import (
	"haji-koding/path-finder/internal/entity"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type JobRepository struct {
	Repository[entity.Job]
	Log *logrus.Logger
}

func NewJobRepository(log *logrus.Logger) *JobRepository {
	return &JobRepository{
		Log: log,
	}
}

// FindAll mengambil seluruh katalog lowongan dengan urutan ID yang stabil.
func (r *JobRepository) FindAll(db *gorm.DB, jobs *[]entity.Job) error {
	return db.Order("id ASC").Find(jobs).Error
}

// FindExistingIDs mengembalikan kumpulan ID lowongan yang sudah ada di database.
func (r *JobRepository) FindExistingIDs(db *gorm.DB, ids []string) (map[string]bool, error) {
	existing := make(map[string]bool, len(ids))
	if len(ids) == 0 {
		return existing, nil
	}

	var found []string
	if err := db.Model(&entity.Job{}).Where("id IN ?", ids).Pluck("id", &found).Error; err != nil {
		return nil, err
	}

	for _, id := range found {
		existing[id] = true
	}
	return existing, nil
}

// CreateBatch menyimpan banyak lowongan sekaligus.
func (r *JobRepository) CreateBatch(db *gorm.DB, jobs []entity.Job) error {
	if len(jobs) == 0 {
		return nil
	}
	return db.CreateInBatches(jobs, 50).Error
}
