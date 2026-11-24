package repository

import (
	"errors"
	"math"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var ErrApplicationStatusNotPending = errors.New("application status is not pending")

type ApplicationRepository struct {
	Repository[entity.Application]
	Log *logrus.Logger
}

func NewApplicationRepository(log *logrus.Logger) *ApplicationRepository {
	return &ApplicationRepository{
		Log: log,
	}
}

func (r *ApplicationRepository) FindByIDWithRelations(db *gorm.DB, app *entity.Application, id string) error {
	return db.Preload("Job").Preload("Mahasiswa").Where("id = ?", id).Take(app).Error
}

func (r *ApplicationRepository) FindActiveByMahasiswaID(db *gorm.DB, app *entity.Application, mahasiswaID string) error {
	return db.Where("mahasiswa_id = ? AND status IN (?, ?)", mahasiswaID, entity.ApplicationStatusPending, entity.ApplicationStatusAccepted).Order("created_at DESC").Take(app).Error
}

func (r *ApplicationRepository) ListByMahasiswaID(db *gorm.DB, apps *[]entity.Application, mahasiswaID string) error {
	return db.Preload("Job").Where("mahasiswa_id = ?", mahasiswaID).Order("created_at DESC").Find(apps).Error
}

func (r *ApplicationRepository) ListByMitraID(db *gorm.DB, apps *[]entity.Application, mitraID string, jobID string, page, size int) (*model.PageMetadata, error) {
	query := db.Model(&entity.Application{}).
		Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.mitra_id = ?", mitraID)

	if jobID != "" {
		query = query.Where("applications.job_id = ?", jobID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	if page > 100000 || page-1 > int(^uint(0)>>1)/size {
		return nil, errors.New("page terlalu besar")
	}
	offset := (page - 1) * size

	err := query.Select("applications.*").
		Preload("Job").
		Preload("Mahasiswa").
		Order("applications.created_at DESC, applications.id ASC").
		Limit(size).
		Offset(offset).
		Find(apps).Error
	if err != nil {
		return nil, err
	}

	totalPage := int64(0)
	if size > 0 && total > 0 {
		totalPage = int64(math.Ceil(float64(total) / float64(size)))
	}

	paging := &model.PageMetadata{
		Page:      page,
		Size:      size,
		TotalItem: total,
		TotalPage: totalPage,
	}

	return paging, nil
}

func (r *ApplicationRepository) UpdateStatus(db *gorm.DB, id string, status string) error {
	result := db.Model(&entity.Application{}).
		Where("id = ? AND status = ?", id, entity.ApplicationStatusPending).
		Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrApplicationStatusNotPending
	}
	return nil
}
