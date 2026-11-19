package repository

import (
	"haji-koding/path-finder/internal/entity"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

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

func (r *ApplicationRepository) ListByMitraID(db *gorm.DB, apps *[]entity.Application, mitraID string, jobID string) error {
	query := db.Joins("JOIN jobs ON jobs.id = applications.job_id").
		Where("jobs.mitra_id = ?", mitraID).
		Preload("Job").
		Preload("Mahasiswa")

	if jobID != "" {
		query = query.Where("applications.job_id = ?", jobID)
	}

	return query.Order("applications.created_at DESC").Find(apps).Error
}

func (r *ApplicationRepository) UpdateStatus(db *gorm.DB, id string, status string) error {
	result := db.Model(&entity.Application{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
