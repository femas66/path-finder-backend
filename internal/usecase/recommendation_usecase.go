package usecase

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/repository"
	"haji-koding/path-finder/internal/service"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type RecommendationUseCase struct {
	DB              *gorm.DB
	Log             *logrus.Logger
	StoryRepository *repository.StoryRepository
	JobRepository   *repository.JobRepository
	MLService       *service.MLService
}

func NewRecommendationUseCase(db *gorm.DB, logger *logrus.Logger, stories *repository.StoryRepository, jobs *repository.JobRepository, ml *service.MLService) *RecommendationUseCase {
	return &RecommendationUseCase{DB: db, Log: logger, StoryRepository: stories, JobRepository: jobs, MLService: ml}
}

func (u *RecommendationUseCase) Recommend(ctx context.Context, studentID string, request *model.RecommendationRequest) (*model.RecommendationResponse, error) {
	if strings.TrimSpace(studentID) == "" {
		return nil, fiber.NewError(401, "Unauthorized")
	}
	if request == nil {
		return nil, fiber.NewError(400, "consent_at must be RFC3339")
	}
	if _, err := time.Parse(time.RFC3339, request.ConsentAt); err != nil {
		return nil, fiber.NewError(400, "consent_at must be RFC3339")
	}
	story := new(entity.Story)
	if err := u.StoryRepository.FindLatestByUserID(u.DB.WithContext(ctx), story, studentID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.NewError(404, "Story not found")
		}
		return nil, fiber.NewError(500, "Unable to retrieve story")
	}
	jobs := []entity.Job{}
	if err := u.JobRepository.FindActive(u.DB.WithContext(ctx), &jobs); err != nil {
		return nil, fiber.NewError(500, "Unable to retrieve catalog")
	}
	catalog := make([]model.MLJob, 0, len(jobs))
	seen := make(map[string]bool)
	for _, job := range jobs {
		valid := !seen[job.ID] && strings.TrimSpace(job.ID) != "" && strings.TrimSpace(job.Title) != "" && entity.IsValidPosition(job.Category) && !job.CreatedAt.IsZero() && ((job.Source == "mitra" && job.SourceURL == "") || strings.TrimSpace(job.SourceURL) != "")
		switch job.Source {
		case "mitra", "Glints", "Dealls", "Jobstreet", "LinkedIn":
		default:
			valid = false
		}
		for _, text := range []string{job.ID, job.Title, job.Category, job.Source, job.SourceURL} {
			if utf8.RuneCountInString(text) > 2000 {
				valid = false
			}
		}
		if utf8.RuneCountInString(job.Description) > 20000 {
			valid = false
		}
		for _, list := range [][]string{job.RequiredSkills, job.OptionalSkills} {
			unique := make(map[string]bool)
			if len(list) > 200 {
				valid = false
			}
			for _, skill := range list {
				key := strings.ToLower(skill)
				if strings.TrimSpace(skill) == "" || utf8.RuneCountInString(skill) > 2000 || unique[key] {
					valid = false
				}
				unique[key] = true
			}
		}
		if !valid || len(jobs) > 200 {
			return nil, fiber.NewError(422, "Catalog incompatible with ML contract")
		}
		seen[job.ID] = true
		required, optional := job.RequiredSkills, job.OptionalSkills
		if required == nil {
			required = []string{}
		}
		if optional == nil {
			optional = []string{}
		}
		catalog = append(catalog, model.MLJob{JobID: job.ID, Title: job.Title, Category: job.Category, Description: job.Description, RequiredSkills: required, OptionalSkills: optional, Source: job.Source, SourceURL: job.SourceURL, CollectedAt: job.CreatedAt.UTC().Format(time.RFC3339Nano)})
	}
	return u.MLService.Recommend(ctx, &model.MLRecommendationRequest{Candidate: model.MLCandidate{StudentID: studentID, Narrative: story.Content, ConsentAt: request.ConsentAt}, Jobs: catalog})
}
