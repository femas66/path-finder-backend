package repository

import (
	"haji-koding/path-finder/internal/entity"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserRepository struct {
	Repository[entity.User]
	Log *logrus.Logger
}

func NewUserRepository(log *logrus.Logger) *UserRepository {
	return &UserRepository{
		Log: log,
	}
}

func (r *UserRepository) FindByEmail(db *gorm.DB, user *entity.User, email string) error {
	return db.Where("email = ?", email).Take(user).Error
}

func (r *UserRepository) CountByEmail(db *gorm.DB, email string) (int64, error) {
	var total int64
	err := db.Model(&entity.User{}).Where("email = ?", email).Count(&total).Error
	return total, err
}

func (r *UserRepository) CountByEmailAndNotID(db *gorm.DB, email string, excludeID string) (int64, error) {
	var total int64
	err := db.Model(&entity.User{}).Where("email = ? AND id != ?", email, excludeID).Count(&total).Error
	return total, err
}

func (r *UserRepository) CountByNIM(db *gorm.DB, nim string) (int64, error) {
	var total int64
	err := db.Model(&entity.User{}).Where("nim = ?", nim).Count(&total).Error
	return total, err
}

func (r *UserRepository) CountByNIMAndNotID(db *gorm.DB, nim string, excludeID string) (int64, error) {
	var total int64
	err := db.Model(&entity.User{}).Where("nim = ? AND id != ?", nim, excludeID).Count(&total).Error
	return total, err
}

func (r *UserRepository) FindMahasiswaByID(db *gorm.DB, user *entity.User, id string) error {
	return db.Where("id = ? AND role = ?", id, entity.RoleMahasiswa).Take(user).Error
}

func (r *UserRepository) FindAllMahasiswa(db *gorm.DB, users *[]entity.User, page, size int, search string) (int64, error) {
	query := db.Model(&entity.User{}).Where("role = ?", entity.RoleMahasiswa)

	if search != "" {
		s := "%" + search + "%"
		query = query.Where("name ILIKE ? OR nim ILIKE ? OR email ILIKE ?", s, s, s)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}

	offset := (page - 1) * size
	err := query.Order("created_at DESC").Offset(offset).Limit(size).Find(users).Error
	return total, err
}
