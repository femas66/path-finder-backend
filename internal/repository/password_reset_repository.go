package repository

import (
	"haji-koding/path-finder/internal/entity"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type PasswordResetRepository struct {
	Repository[entity.PasswordReset]
	Log *logrus.Logger
}

func NewPasswordResetRepository(log *logrus.Logger) *PasswordResetRepository {
	return &PasswordResetRepository{
		Log: log,
	}
}

func (r *PasswordResetRepository) DeleteByEmail(db *gorm.DB, email string) error {
	return db.Where("email = ?", email).Delete(&entity.PasswordReset{}).Error
}

func (r *PasswordResetRepository) FindByEmailAndCode(db *gorm.DB, reset *entity.PasswordReset, email, code string) error {
	return db.Where("email = ? AND code = ?", email, code).Take(reset).Error
}
