package usecase

import (
	"context"
	"math"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/model/converter"
	"haji-koding/path-finder/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type MahasiswaUseCase struct {
	DB             *gorm.DB
	Log            *logrus.Logger
	Validate       *validator.Validate
	UserRepository *repository.UserRepository
}

func NewMahasiswaUseCase(
	db *gorm.DB,
	logger *logrus.Logger,
	validate *validator.Validate,
	userRepo *repository.UserRepository,
) *MahasiswaUseCase {
	return &MahasiswaUseCase{
		DB:             db,
		Log:            logger,
		Validate:       validate,
		UserRepository: userRepo,
	}
}

func (u *MahasiswaUseCase) Create(ctx context.Context, request *model.CreateMahasiswaRequest) (*model.MahasiswaResponse, error) {
	if err := u.Validate.Struct(request); err != nil {
		u.Log.Warnf("Invalid create mahasiswa request: %v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// 1. Cek email unik
	totalEmail, err := u.UserRepository.CountByEmail(u.DB.WithContext(ctx), request.Email)
	if err != nil {
		u.Log.Errorf("Failed to check email: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "terjadi kesalahan pada server")
	}
	if totalEmail > 0 {
		return nil, fiber.NewError(fiber.StatusConflict, "email sudah terdaftar")
	}

	// 2. Cek NIM unik
	totalNIM, err := u.UserRepository.CountByNIM(u.DB.WithContext(ctx), request.NIM)
	if err != nil {
		u.Log.Errorf("Failed to check NIM: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "terjadi kesalahan pada server")
	}
	if totalNIM > 0 {
		return nil, fiber.NewError(fiber.StatusConflict, "NIM sudah terdaftar")
	}

	// 3. Default password adalah NIM dari mahasiswa itu sendiri
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.NIM), bcrypt.DefaultCost)
	if err != nil {
		u.Log.Errorf("Failed to hash default password (NIM): %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengenkripsi password")
	}

	user := &entity.User{
		ID:             uuid.New().String(),
		NIM:            request.NIM,
		Name:           request.Name,
		Email:          request.Email,
		Password:       string(hashedPassword),
		Role:           entity.RoleMahasiswa,
		ProfilePicture: request.ProfilePicture,
		CV:             request.CV,
		PortfolioURL:   request.PortfolioURL,
	}

	if err := u.UserRepository.Create(u.DB.WithContext(ctx), user); err != nil {
		u.Log.Errorf("Failed to save mahasiswa: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal menyimpan data mahasiswa")
	}

	return converter.UserToMahasiswaResponse(user), nil
}

func (u *MahasiswaUseCase) Get(ctx context.Context, id string) (*model.MahasiswaResponse, error) {
	user := new(entity.User)
	if err := u.UserRepository.FindMahasiswaByID(u.DB.WithContext(ctx), user, id); err != nil {
		u.Log.Warnf("Mahasiswa not found with id %s: %v", id, err)
		return nil, fiber.NewError(fiber.StatusNotFound, "data mahasiswa tidak ditemukan")
	}

	return converter.UserToMahasiswaResponse(user), nil
}

func (u *MahasiswaUseCase) Search(ctx context.Context, request *model.SearchMahasiswaRequest) ([]*model.MahasiswaResponse, *model.PageMetadata, error) {
	if request.Page <= 0 {
		request.Page = 1
	}
	if request.Size <= 0 {
		request.Size = 10
	}
	if request.Size > 100 {
		request.Size = 100
	}

	var users []entity.User
	total, err := u.UserRepository.FindAllMahasiswa(u.DB.WithContext(ctx), &users, request.Page, request.Size, request.Search)
	if err != nil {
		u.Log.Errorf("Failed to find all mahasiswa: %v", err)
		return nil, nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengambil data mahasiswa")
	}

	totalPage := int64(math.Ceil(float64(total) / float64(request.Size)))
	paging := &model.PageMetadata{
		Page:      request.Page,
		Size:      request.Size,
		TotalItem: total,
		TotalPage: totalPage,
	}

	return converter.UsersToMahasiswaResponses(users), paging, nil
}

func (u *MahasiswaUseCase) Update(ctx context.Context, id string, request *model.UpdateMahasiswaByAdminRequest) (*model.MahasiswaResponse, error) {
	if err := u.Validate.Struct(request); err != nil {
		u.Log.Warnf("Invalid update mahasiswa request: %v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user := new(entity.User)
	if err := u.UserRepository.FindMahasiswaByID(u.DB.WithContext(ctx), user, id); err != nil {
		u.Log.Warnf("Mahasiswa not found with id %s: %v", id, err)
		return nil, fiber.NewError(fiber.StatusNotFound, "data mahasiswa tidak ditemukan")
	}

	// Cek keunikan email baru jika diubah
	if request.Email != "" && request.Email != user.Email {
		total, err := u.UserRepository.CountByEmailAndNotID(u.DB.WithContext(ctx), request.Email, id)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal memeriksa email")
		}
		if total > 0 {
			return nil, fiber.NewError(fiber.StatusConflict, "email sudah digunakan oleh pengguna lain")
		}
		user.Email = request.Email
	}

	// Cek keunikan NIM baru jika diubah
	if request.NIM != "" && request.NIM != user.NIM {
		total, err := u.UserRepository.CountByNIMAndNotID(u.DB.WithContext(ctx), request.NIM, id)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal memeriksa NIM")
		}
		if total > 0 {
			return nil, fiber.NewError(fiber.StatusConflict, "NIM sudah digunakan oleh mahasiswa lain")
		}
		user.NIM = request.NIM
	}

	if request.Name != "" {
		user.Name = request.Name
	}
	if request.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengenkripsi password")
		}
		user.Password = string(hashed)
	}
	if request.ProfilePicture != "" {
		user.ProfilePicture = request.ProfilePicture
	}
	if request.CV != "" {
		user.CV = request.CV
	}
	if request.PortfolioURL != "" {
		user.PortfolioURL = request.PortfolioURL
	}

	if err := u.UserRepository.Update(u.DB.WithContext(ctx), user); err != nil {
		u.Log.Errorf("Failed to update mahasiswa %s: %v", id, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal memperbarui data mahasiswa")
	}

	return converter.UserToMahasiswaResponse(user), nil
}

func (u *MahasiswaUseCase) Delete(ctx context.Context, id string) error {
	user := new(entity.User)
	if err := u.UserRepository.FindMahasiswaByID(u.DB.WithContext(ctx), user, id); err != nil {
		u.Log.Warnf("Mahasiswa not found with id %s: %v", id, err)
		return fiber.NewError(fiber.StatusNotFound, "data mahasiswa tidak ditemukan")
	}

	if err := u.UserRepository.Delete(u.DB.WithContext(ctx), user); err != nil {
		u.Log.Errorf("Failed to delete mahasiswa %s: %v", id, err)
		return fiber.NewError(fiber.StatusInternalServerError, "gagal menghapus data mahasiswa")
	}

	return nil
}
