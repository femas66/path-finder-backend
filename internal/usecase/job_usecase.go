package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/repository"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type JobUseCase struct {
	DB            *gorm.DB
	Log           *logrus.Logger
	Validate      *validator.Validate
	JobRepository *repository.JobRepository
}

func NewJobUseCase(db *gorm.DB, log *logrus.Logger, validate *validator.Validate, jobs *repository.JobRepository) *JobUseCase {
	return &JobUseCase{DB: db, Log: log, Validate: validate, JobRepository: jobs}
}

func normalizeJobSkills(skills []string) []string {
	result := make([]string, 0, len(skills))
	seen := make(map[string]bool, len(skills))
	for _, skill := range skills {
		skill = strings.TrimSpace(skill)
		key := strings.ToLower(skill)
		if !seen[key] {
			result = append(result, skill)
			seen[key] = true
		}
	}
	return result
}

func validJobURL(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != ""
}

func jobResponse(job *entity.Job) *model.JobResponse {
	return &model.JobResponse{
		ID: job.ID, MitraID: job.MitraID, Title: job.Title, Category: job.Category,
		Company: job.Company, Description: job.Description, Location: job.Location,
		WorkType: job.WorkType, Source: job.Source, SourceURL: job.SourceURL,
		RequiredSkills: append([]string{}, job.RequiredSkills...), OptionalSkills: append([]string{}, job.OptionalSkills...),
		CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt,
	}
}

func (u *JobUseCase) jobError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fiber.NewError(fiber.StatusNotFound, "lowongan tidak ditemukan")
	}
	u.Log.WithError(err).Warn("Job database operation failed")
	return fiber.NewError(fiber.StatusInternalServerError, "gagal memproses lowongan")
}

func (u *JobUseCase) mitra(ctx context.Context, id string) (*entity.User, error) {
	user := new(entity.User)
	if err := u.DB.WithContext(ctx).Where("id = ?", id).Take(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fiber.ErrUnauthorized
		}
		return nil, u.jobError(err)
	}
	if user.Role != entity.RoleMitra {
		return nil, fiber.ErrForbidden
	}
	return user, nil
}

func (u *JobUseCase) Create(ctx context.Context, mitraID string, request *model.CreateJobRequest) (*model.JobResponse, error) {
	request.Title = strings.TrimSpace(request.Title)
	request.Category = strings.TrimSpace(request.Category)
	request.Description = strings.TrimSpace(request.Description)
	request.Location = strings.TrimSpace(request.Location)
	request.WorkType = strings.TrimSpace(request.WorkType)
	request.SourceURL = strings.TrimSpace(request.SourceURL)
	// Validate trimmed entries before deduplication so oversized input cannot bypass limits.
	for i := range request.RequiredSkills {
		request.RequiredSkills[i] = strings.TrimSpace(request.RequiredSkills[i])
	}
	for i := range request.OptionalSkills {
		request.OptionalSkills[i] = strings.TrimSpace(request.OptionalSkills[i])
	}
	if err := u.Validate.Struct(request); err != nil {
		return nil, fiber.NewError(400, err.Error())
	}
	if !entity.IsValidPosition(request.Category) {
		return nil, fiber.NewError(400, "kategori lowongan tidak valid")
	}
	if !validJobURL(request.SourceURL) {
		return nil, fiber.NewError(400, "source_url harus URL http/https atau kosong")
	}
	user, err := u.mitra(ctx, mitraID)
	if err != nil {
		return nil, err
	}
	job := &entity.Job{
		ID: uuid.NewString(), MitraID: &mitraID, Title: request.Title, Category: request.Category,
		Company: user.Name, Description: request.Description, Location: request.Location,
		WorkType: request.WorkType, Source: "mitra", SourceURL: request.SourceURL,
		RequiredSkills: normalizeJobSkills(request.RequiredSkills), OptionalSkills: normalizeJobSkills(request.OptionalSkills),
	}
	if err := u.JobRepository.Create(u.DB.WithContext(ctx), job); err != nil {
		return nil, u.jobError(err)
	}
	return jobResponse(job), nil
}

func (u *JobUseCase) List(ctx context.Context, request *model.ListJobRequest, mitraID *string) ([]model.JobResponse, *model.PageMetadata, error) {
	request.Search = strings.TrimSpace(request.Search)
	request.Category = strings.TrimSpace(request.Category)
	if err := u.Validate.Struct(request); err != nil {
		return nil, nil, fiber.NewError(400, err.Error())
	}
	if request.Page-1 > int(^uint(0)>>1)/request.Size {
		return nil, nil, fiber.NewError(400, "page terlalu besar")
	}
	if request.Category != "" && !entity.IsValidPosition(request.Category) {
		return nil, nil, fiber.NewError(400, "kategori lowongan tidak valid")
	}
	jobs := make([]entity.Job, 0)
	total, err := u.JobRepository.List(u.DB.WithContext(ctx), &jobs, request, mitraID)
	if err != nil {
		return nil, nil, u.jobError(err)
	}
	responses := make([]model.JobResponse, 0, len(jobs))
	for i := range jobs {
		responses = append(responses, *jobResponse(&jobs[i]))
	}
	paging := &model.PageMetadata{Page: request.Page, Size: request.Size, TotalItem: total, TotalPage: total / int64(request.Size)}
	if total%int64(request.Size) != 0 {
		paging.TotalPage++
	}
	return responses, paging, nil
}

func (u *JobUseCase) Get(ctx context.Context, id string, mitraID *string) (*model.JobResponse, error) {
	job := new(entity.Job)
	db := u.DB.WithContext(ctx)
	var err error
	if mitraID == nil {
		err = u.JobRepository.FindById(db, job, id)
	} else {
		err = u.JobRepository.FindOwned(db, job, id, *mitraID)
	}
	if err != nil {
		return nil, u.jobError(err)
	}
	return jobResponse(job), nil
}

func (u *JobUseCase) Update(ctx context.Context, id, mitraID string, request *model.UpdateJobRequest) (*model.JobResponse, error) {
	for _, field := range []*string{request.Title, request.Category, request.Description, request.Location, request.WorkType, request.SourceURL} {
		if field != nil {
			*field = strings.TrimSpace(*field)
		}
	}
	for _, skills := range []*[]string{request.RequiredSkills, request.OptionalSkills} {
		if skills != nil {
			for i := range *skills {
				(*skills)[i] = strings.TrimSpace((*skills)[i])
			}
		}
	}
	if err := u.Validate.Struct(request); err != nil {
		return nil, fiber.NewError(400, err.Error())
	}
	if request.Category != nil && !entity.IsValidPosition(*request.Category) {
		return nil, fiber.NewError(400, "kategori lowongan tidak valid")
	}
	if request.SourceURL != nil && !validJobURL(*request.SourceURL) {
		return nil, fiber.NewError(400, "source_url harus URL http/https atau kosong")
	}
	job := new(entity.Job)
	if err := u.JobRepository.FindOwned(u.DB.WithContext(ctx), job, id, mitraID); err != nil {
		return nil, u.jobError(err)
	}
	user, err := u.mitra(ctx, mitraID)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{"company": user.Name, "source": "mitra"}
	for key, field := range map[string]*string{
		"title": request.Title, "category": request.Category, "description": request.Description,
		"location": request.Location, "work_type": request.WorkType, "source_url": request.SourceURL,
	} {
		if field != nil {
			updates[key] = *field
		}
	}
	for key, skills := range map[string]*[]string{"required_skills": request.RequiredSkills, "optional_skills": request.OptionalSkills} {
		if skills != nil {
			// Map updates bypass GORM's field serializer; encode JSON explicitly for text columns.
			encoded, err := json.Marshal(normalizeJobSkills(*skills))
			if err != nil {
				return nil, u.jobError(err)
			}
			updates[key] = string(encoded)
		}
	}
	if err := u.JobRepository.UpdateOwned(u.DB.WithContext(ctx), job, id, mitraID, updates); err != nil {
		return nil, u.jobError(err)
	}
	return jobResponse(job), nil
}

func (u *JobUseCase) Delete(ctx context.Context, id, mitraID string) error {
	if err := u.JobRepository.DeleteOwned(u.DB.WithContext(ctx), id, mitraID); err != nil {
		return u.jobError(err)
	}
	return nil
}
