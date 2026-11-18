package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/model/converter"
	"haji-koding/path-finder/internal/repository"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// minimumStoryLength sama dengan aturan validasi saat cerita dikirim.
const minimumStoryLength = 50

type RecommendationUseCase struct {
	DB                       *gorm.DB
	Log                      *logrus.Logger
	RecommendationRepository *repository.RecommendationRepository
	JobRepository            *repository.JobRepository
	StoryRepository          *repository.StoryRepository
	Lexicon                  *SkillLexicon
}

func NewRecommendationUseCase(
	db *gorm.DB,
	logger *logrus.Logger,
	recommendationRepository *repository.RecommendationRepository,
	jobRepository *repository.JobRepository,
	storyRepository *repository.StoryRepository,
	lexicon *SkillLexicon,
) *RecommendationUseCase {
	return &RecommendationUseCase{
		DB:                       db,
		Log:                      logger,
		RecommendationRepository: recommendationRepository,
		JobRepository:            jobRepository,
		StoryRepository:          storyRepository,
		Lexicon:                  lexicon,
	}
}

// Generate menganalisis cerita pengguna lalu menyimpan hasilnya.
func (u *RecommendationUseCase) Generate(
	ctx context.Context,
	userID string,
	request *model.CreateRecommendationRequest,
) (*model.RecommendationResponse, error) {
	if request == nil {
		request = new(model.CreateRecommendationRequest)
	}

	topK := defaultTopK
	if request.TopK != nil {
		topK = *request.TopK
	}
	if topK < 1 || topK > maxTopK {
		return nil, fiber.NewError(fiber.StatusBadRequest, "top_k harus bernilai 1 sampai 8")
	}

	story, err := u.loadStory(ctx, userID, strings.TrimSpace(request.StoryID))
	if err != nil {
		return nil, err
	}

	if len([]rune(strings.TrimSpace(story.Content))) < minimumStoryLength {
		return nil, fiber.NewError(fiber.StatusBadRequest, "cerita terlalu pendek untuk dianalisis")
	}

	db := u.DB.WithContext(ctx)

	profile := u.Lexicon.Extract(story.Content)

	jobs := new([]entity.Job)
	if err := u.JobRepository.FindAll(db, jobs); err != nil {
		u.Log.Warnf("Gagal mengambil katalog lowongan: %v", err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengambil katalog lowongan")
	}
	if len(*jobs) == 0 {
		u.Log.Warn("Katalog lowongan kosong saat rekomendasi diminta")
		return nil, fiber.NewError(fiber.StatusInternalServerError, "katalog lowongan belum tersedia")
	}

	recommendation := &entity.Recommendation{
		ID:            uuid.New().String(),
		UserID:        userID,
		StoryID:       story.ID,
		SchemaVersion: schemaVersion,
		ModelVersion:  modelVersion,
		TopK:          topK,
		ProfileStatus: entity.ProfileStatusLengkap,
		HardSkills:    profile.HardSkills,
		SoftSkills:    profile.SoftSkills,
		Interests:     profile.Interests,
		Items:         []entity.RecommendationItem{},
		GeneratedAt:   time.Now().UTC(),
	}

	if !profile.HasSkills() {
		// Kontrak ML: profil tanpa keahlian tidak menghasilkan item.
		recommendation.Status = entity.RecommendationStatusProfileIncomplete
		recommendation.ProfileStatus = entity.ProfileStatusTanpaSkill
	} else {
		completeness := profileCompleteness(profile)

		items := make([]entity.RecommendationItem, 0, len(*jobs))
		for _, job := range *jobs {
			items = append(items, buildItem(profile, job, completeness))
		}
		items = rankItems(items)

		if len(items) > topK {
			items = items[:topK]
		}

		recommendation.Items = items
		recommendation.Status = entity.RecommendationStatusOK
		for _, item := range items {
			if item.ScoreStatus != entity.ScoreStatusOK {
				recommendation.Status = entity.RecommendationStatusParsial
				break
			}
		}
	}

	if err := u.RecommendationRepository.Create(db, recommendation); err != nil {
		u.Log.Warnf("Gagal menyimpan rekomendasi untuk user %s: %v", userID, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal menyimpan rekomendasi")
	}

	// Kegagalan menandai cerita tidak boleh membatalkan hasil yang sudah jadi.
	if err := u.StoryRepository.UpdateAnalysisStatus(db, story.ID, entity.StoryAnalysisDone); err != nil {
		u.Log.Warnf("Gagal memperbarui status analisis cerita %s: %v", story.ID, err)
	}

	return converter.RecommendationToResponse(recommendation), nil
}

// Latest mengembalikan rekomendasi terakhir milik pengguna.
func (u *RecommendationUseCase) Latest(ctx context.Context, userID string) (*model.RecommendationResponse, error) {
	recommendation := new(entity.Recommendation)

	err := u.RecommendationRepository.FindLatestByUserID(u.DB.WithContext(ctx), recommendation, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "rekomendasi belum pernah dibuat")
		}
		u.Log.Warnf("Gagal mengambil rekomendasi user %s: %v", userID, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengambil rekomendasi")
	}

	return converter.RecommendationToResponse(recommendation), nil
}

// loadStory mengambil cerita yang akan dianalisis.
func (u *RecommendationUseCase) loadStory(ctx context.Context, userID string, storyID string) (*entity.Story, error) {
	db := u.DB.WithContext(ctx)
	story := new(entity.Story)

	if storyID != "" {
		if err := u.StoryRepository.FindByIDAndUserID(db, story, storyID, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fiber.NewError(fiber.StatusNotFound, "cerita tidak ditemukan")
			}
			u.Log.Warnf("Gagal mengambil cerita %s milik user %s: %v", storyID, userID, err)
			return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengambil cerita")
		}
		return story, nil
	}

	if err := u.StoryRepository.FindLatestByUserID(db, story, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(fiber.StatusNotFound, "cerita belum pernah dikirim, kirim cerita dulu")
		}
		u.Log.Warnf("Gagal mengambil cerita terakhir user %s: %v", userID, err)
		return nil, fiber.NewError(fiber.StatusInternalServerError, "gagal mengambil cerita")
	}

	return story, nil
}
