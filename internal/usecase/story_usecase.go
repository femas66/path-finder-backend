package usecase

import (
	"context"
	"strings"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/model/converter"
	"haji-koding/path-finder/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type StoryUseCase struct {
	DB              *gorm.DB
	Log             *logrus.Logger
	Validate        *validator.Validate
	StoryRepository *repository.StoryRepository
}

func NewStoryUseCase(
	db *gorm.DB,
	logger *logrus.Logger,
	validate *validator.Validate,
	storyRepository *repository.StoryRepository,
) *StoryUseCase {
	return &StoryUseCase{
		DB:              db,
		Log:             logger,
		Validate:        validate,
		StoryRepository: storyRepository,
	}
}

// Create menyimpan cerita diri milik pengguna yang sedang login.
//
// Konten dirapikan lebih dulu (trim) sebelum divalidasi agar spasi di awal/akhir
// tidak membuat panjang minimum 50 karakter lolos secara palsu.
func (u *StoryUseCase) Create(ctx context.Context, userID string, request *model.CreateStoryRequest) (*model.StoryResponse, error) {
	request.Content = strings.TrimSpace(request.Content)

	if err := u.Validate.Struct(request); err != nil {
		u.Log.Warnf("Invalid create story request: %v", err)
		return nil, fiber.NewError(fiber.StatusBadRequest, "cerita minimal 50 karakter dan maksimal 2000 karakter")
	}

	story := &entity.Story{
		ID:             uuid.New().String(),
		UserID:         userID,
		Content:        request.Content,
		WordCount:      len(strings.Fields(request.Content)),
		AnalysisStatus: entity.StoryAnalysisPending,
	}

	if err := u.StoryRepository.Create(u.DB.WithContext(ctx), story); err != nil {
		u.Log.Warnf("Failed to create story for user %s: %v", userID, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal menyimpan cerita")
	}

	return converter.StoryToResponse(story), nil
}

// Current mengembalikan cerita terakhir milik pengguna.
func (u *StoryUseCase) Current(ctx context.Context, userID string) (*model.StoryResponse, error) {
	story := new(entity.Story)
	if err := u.StoryRepository.FindLatestByUserID(u.DB.WithContext(ctx), story, userID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fiber.NewError(fiber.StatusNotFound, "cerita belum pernah dikirim")
		}
		u.Log.Warnf("Failed to find story for user %s: %v", userID, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengambil cerita")
	}

	return converter.StoryToResponse(story), nil
}
