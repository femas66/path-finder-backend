package repository

import (
	"strings"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

// List applies the same tenant and catalog filters to count and page queries.
func (r *JobRepository) List(db *gorm.DB, jobs *[]entity.Job, request *model.ListJobRequest, mitraID *string) (int64, error) {
	query := db.Model(&entity.Job{})
	if mitraID != nil {
		query = query.Where("mitra_id = ?", *mitraID)
	}
	if request.Category != "" {
		query = query.Where("category = ?", request.Category)
	}
	if request.Search != "" {
		search := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(request.Search)
		query = query.Where("(title ILIKE ? OR company ILIKE ?)", "%"+search+"%", "%"+search+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	err := query.Order("created_at DESC").Order("id ASC").Limit(request.Size).Offset((request.Page - 1) * request.Size).Find(jobs).Error
	return total, err
}

func (r *JobRepository) FindOwned(db *gorm.DB, job *entity.Job, id, mitraID string) error {
	return db.Where("id = ? AND mitra_id = ?", id, mitraID).Take(job).Error
}

// Ownership remains part of the write predicate, not merely a preceding read.
func (r *JobRepository) UpdateOwned(db *gorm.DB, job *entity.Job, id, mitraID string, updates map[string]any) error {
	result := db.Model(job).Clauses(clause.Returning{}).Where("id = ? AND mitra_id = ?", id, mitraID).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *JobRepository) DeleteOwned(db *gorm.DB, id, mitraID string) error {
	result := db.Where("id = ? AND mitra_id = ?", id, mitraID).Delete(&entity.Job{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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
