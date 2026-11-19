package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const jobTestSecret = "pathfinder-job-integration-only-secret-" // Never uses deployment credentials for JWTs.

type jobTestFixture struct {
	app                      *fiber.App
	db                       *gorm.DB
	log                      *logrus.Logger
	owner, other, student    entity.User
	ownerToken, otherToken   string
	studentToken, adminToken string
}

// Every test gets a private schema. search_path is a connection startup parameter,
// not a SET on one pooled connection; public is deliberately excluded.
func newJobTestFixture(t *testing.T) *jobTestFixture {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("PATHFINDER_JOB_TEST_DSN"))
	ci := strings.EqualFold(os.Getenv("CI"), "true")
	if dsn == "" && !ci {
		t.Skip("set PATHFINDER_JOB_TEST_DSN to run isolated PostgreSQL jobs integration tests (CI=true uses config.NewViper)")
	}
	settings := viper.New()
	secret := jobTestSecret + uuid.NewString()
	if dsn == "" {
		// NewViper validates JWT_SECRET before Bootstrap; do not depend on CI secrets.
		t.Setenv("JWT_SECRET", secret)
		settings = config.NewViper()
		base := &url.URL{Scheme: "postgres", Host: fmt.Sprintf("%s:%d", settings.GetString("database.host"), settings.GetInt("database.port")), Path: "/" + settings.GetString("database.name")}
		base.User = url.UserPassword(settings.GetString("database.username"), settings.GetString("database.password"))
		query := base.Query()
		query.Set("sslmode", "disable")
		base.RawQuery = query.Encode()
		dsn = base.String()
	}
	settings.Set("jwt.secret", secret)
	settings.Set("app.name", "pathfinder-jobs-test")
	pgConfig, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	base := stdlib.OpenDB(*pgConfig)
	t.Cleanup(func() { require.NoError(t, base.Close()) })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.NoError(t, base.PingContext(ctx), "jobs test PostgreSQL must be reachable")
	schema := "job_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	_, err = base.ExecContext(ctx, "CREATE SCHEMA "+quotedSchema)
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		_, err := base.ExecContext(cleanupCtx, "DROP SCHEMA "+quotedSchema+" CASCADE")
		require.NoError(t, err)
	})
	isolated := pgConfig.Copy()
	isolated.RuntimeParams["search_path"] = schema
	connection := stdlib.OpenDB(*isolated)
	connection.SetMaxOpenConns(1)
	connection.SetMaxIdleConns(1)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: connection}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	var currentSchema string
	require.NoError(t, db.Raw("SELECT current_schema()").Scan(&currentSchema).Error)
	require.Equal(t, schema, currentSchema)
	log := logrus.New()
	log.SetOutput(io.Discard)
	// Bootstrap uses Fatal on migration errors. Fail this test rather than exit the suite.
	log.ExitFunc = func(int) { t.Fatal("jobs test Bootstrap failed") }
	app := config.NewFiber(settings)
	config.Bootstrap(&config.BootstrapConfig{DB: db, App: app, Log: log, Validate: config.NewValidator(), Config: settings})
	t.Cleanup(func() { require.NoError(t, app.Shutdown()) })
	f := &jobTestFixture{app: app, db: db, log: log}
	createUser := func(name, role string) (entity.User, string) {
		user := entity.User{ID: uuid.NewString(), Name: name, Email: uuid.NewString() + "@jobs.test", Password: "unused-direct-jwt", Role: role}
		require.NoError(t, db.Create(&user).Error)
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": user.ID, "role": role, "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
		require.NoError(t, err)
		return user, token
	}
	f.owner, f.ownerToken = createUser("Jobs Test Owner", entity.RoleMitra)
	f.other, f.otherToken = createUser("Other Test Company", entity.RoleMitra)
	f.student, f.studentToken = createUser("Jobs Test Student", entity.RoleMahasiswa)
	_, f.adminToken = createUser("Jobs Test Admin", entity.RoleAdmin)
	return f
}

func (f *jobTestFixture) request(t *testing.T, method, path, token string, body any, status int, output any) []byte {
	t.Helper()
	var reader io.Reader
	if raw, ok := body.(string); ok {
		reader = strings.NewReader(raw)
	} else if body != nil {
		encoded, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(encoded)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := f.app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, status, resp.StatusCode, "%s %s: %s", method, path, data)
	var envelope model.WebResponse[json.RawMessage]
	require.NoError(t, json.Unmarshal(data, &envelope), "%s", data)
	require.Equal(t, status < 400, envelope.Success, "%s", data)
	require.NotEmpty(t, envelope.Message)
	if status >= 400 {
		require.NotEmpty(t, envelope.Errors)
	}
	if output != nil {
		require.NoError(t, json.Unmarshal(data, output), "%s", data)
	}
	return data
}

func jobTestPayload() map[string]any {
	return map[string]any{
		"title": "Backend Integration Engineer", "category": "Backend Developer Intern",
		"description": "Build reliable backend services with Go and PostgreSQL.",
		"location":    "Jakarta", "work_type": "Hybrid", "source_url": "https://example.org/jobs/backend",
		"required_skills": []string{"Go", "PostgreSQL"}, "optional_skills": []string{"Docker"},
	}
}

func (f *jobTestFixture) create(t *testing.T, token string, payload map[string]any) model.JobResponse {
	t.Helper()
	var response model.WebResponse[model.JobResponse]
	f.request(t, "POST", "/api/mitra/jobs", token, payload, 201, &response)
	require.NotEmpty(t, response.Data.ID)
	_, err := uuid.Parse(response.Data.ID)
	require.NoError(t, err)
	return response.Data
}

func (f *jobTestFixture) list(t *testing.T, path, token string) model.WebResponse[[]model.JobResponse] {
	t.Helper()
	var response model.WebResponse[[]model.JobResponse]
	f.request(t, "GET", path, token, nil, 200, &response)
	require.NotNil(t, response.Data, "empty lists must be JSON arrays")
	require.NotNil(t, response.Paging)
	return response
}

func (f *jobTestFixture) stored(t *testing.T, id string) entity.Job {
	t.Helper()
	var job entity.Job
	require.NoError(t, f.db.First(&job, "id = ?", id).Error)
	return job
}

func requirePersistedJobEqual(t *testing.T, expected, actual model.JobResponse) {
	t.Helper()
	// PostgreSQL timestamps retain microseconds, while create responses can contain nanoseconds.
	require.WithinDuration(t, expected.CreatedAt, actual.CreatedAt, time.Microsecond)
	require.WithinDuration(t, expected.UpdatedAt, actual.UpdatedAt, time.Microsecond)
	expected.CreatedAt, expected.UpdatedAt = actual.CreatedAt, actual.UpdatedAt
	require.Equal(t, expected, actual)
}

func TestJobsCRUDPersistenceAndSpoofing(t *testing.T) {
	f := newJobTestFixture(t)
	payload := jobTestPayload()
	payload["title"] = "  Backend Integration Engineer  "
	payload["mitra_id"], payload["owner_id"], payload["company"], payload["source"] = f.other.ID, f.other.ID, "Spoofed Company", "seed"
	payload["id"] = uuid.NewString()
	payload["required_skills"] = []string{" Go ", "go", " PostgreSQL "}
	payload["optional_skills"] = []string{" Docker ", "docker"}
	created := f.create(t, f.ownerToken, payload)
	require.NotEqual(t, payload["id"], created.ID)
	require.Equal(t, &f.owner.ID, created.MitraID)
	require.Equal(t, f.owner.Name, created.Company)
	require.Equal(t, "mitra", created.Source)
	require.Equal(t, "Backend Integration Engineer", created.Title)
	require.Equal(t, []string{"Go", "PostgreSQL"}, created.RequiredSkills)
	require.Equal(t, []string{"Docker"}, created.OptionalSkills)
	require.False(t, created.CreatedAt.IsZero())
	require.False(t, created.UpdatedAt.IsZero())
	stored := f.stored(t, created.ID)
	require.Equal(t, created.RequiredSkills, stored.RequiredSkills)
	require.Equal(t, created.OptionalSkills, stored.OptionalSkills)
	for _, path := range []string{"/api/jobs/", "/api/mitra/jobs/"} {
		var response model.WebResponse[model.JobResponse]
		f.request(t, "GET", path+created.ID, f.ownerToken, nil, 200, &response)
		requirePersistedJobEqual(t, created, response.Data)
	}
	own := f.list(t, "/api/mitra/jobs", f.ownerToken)
	require.EqualValues(t, 1, own.Paging.TotalItem)
	requirePersistedJobEqual(t, created, own.Data[0])
	var updated model.WebResponse[model.JobResponse]
	f.request(t, "PATCH", "/api/mitra/jobs/"+created.ID, f.ownerToken, map[string]any{
		"title": "  Updated Backend Engineer  ", "required_skills": []string{" SQL ", "sql", "Go"},
		"optional_skills": []string{" Kubernetes ", "kubernetes"},
		"mitra_id":        f.other.ID, "owner_id": f.other.ID, "company": "Spoofed", "source": "seed", "id": uuid.NewString(),
	}, 200, &updated)
	require.Equal(t, "Updated Backend Engineer", updated.Data.Title)
	require.Equal(t, created.ID, updated.Data.ID)
	require.Equal(t, &f.owner.ID, updated.Data.MitraID)
	require.Equal(t, f.owner.Name, updated.Data.Company)
	require.Equal(t, "mitra", updated.Data.Source)
	require.Equal(t, created.Description, updated.Data.Description)
	require.WithinDuration(t, created.CreatedAt, updated.Data.CreatedAt, time.Microsecond)
	require.False(t, updated.Data.UpdatedAt.Before(created.UpdatedAt))
	stored = f.stored(t, created.ID)
	require.Equal(t, []string{"SQL", "Go"}, stored.RequiredSkills)
	require.Equal(t, []string{"Kubernetes"}, stored.OptionalSkills)
	var raw struct{ RequiredSkills, OptionalSkills string }
	require.NoError(t, f.db.Raw("SELECT required_skills, optional_skills FROM jobs WHERE id = ?", created.ID).Scan(&raw).Error)
	var skills []string
	require.NoError(t, json.Unmarshal([]byte(raw.RequiredSkills), &skills))
	require.Equal(t, stored.RequiredSkills, skills)
	require.NoError(t, json.Unmarshal([]byte(raw.OptionalSkills), &skills))
	require.Equal(t, stored.OptionalSkills, skills)
	var readback model.WebResponse[model.JobResponse]
	f.request(t, "GET", "/api/jobs/"+created.ID, f.studentToken, nil, 200, &readback)
	requirePersistedJobEqual(t, updated.Data, readback.Data)
	f.request(t, "PUT", "/api/mitra/jobs/"+created.ID, f.ownerToken, map[string]any{
		"location": " Bandung ", "optional_skills": []string{}, "source_url": "", "title": nil, "required_skills": nil,
		"mitra_id": f.other.ID, "company": "Spoofed", "source": "csv",
	}, 200, &updated)
	require.Equal(t, "Bandung", updated.Data.Location)
	require.Equal(t, "Updated Backend Engineer", updated.Data.Title)
	require.Equal(t, []string{"SQL", "Go"}, updated.Data.RequiredSkills)
	require.NotNil(t, updated.Data.OptionalSkills)
	require.Empty(t, updated.Data.OptionalSkills)
	require.Empty(t, updated.Data.SourceURL)
	stored = f.stored(t, created.ID)
	require.Equal(t, &f.owner.ID, stored.MitraID)
	require.Equal(t, f.owner.Name, stored.Company)
	require.Equal(t, "mitra", stored.Source)
	require.Equal(t, "Bandung", stored.Location)
	require.Empty(t, stored.OptionalSkills)
	require.Empty(t, stored.SourceURL)
	for _, method := range []string{"PATCH", "PUT"} {
		f.request(t, method, "/api/mitra/jobs/"+created.ID, f.ownerToken, map[string]any{}, 200, &readback)
		require.Equal(t, updated.Data.Title, readback.Data.Title)
		require.Equal(t, updated.Data.RequiredSkills, readback.Data.RequiredSkills)
	}
	f.request(t, "DELETE", "/api/mitra/jobs/"+created.ID, f.ownerToken, nil, 200, nil)
	var count int64
	require.NoError(t, f.db.Model(&entity.Job{}).Where("id = ?", created.ID).Count(&count).Error)
	require.Zero(t, count)
	for _, path := range []string{"/api/jobs/", "/api/mitra/jobs/"} {
		f.request(t, "GET", path+created.ID, f.ownerToken, nil, 404, nil)
	}
	f.request(t, "DELETE", "/api/mitra/jobs/"+created.ID, f.ownerToken, nil, 404, nil)
	require.Empty(t, f.list(t, "/api/mitra/jobs", f.ownerToken).Data)
}

func TestJobsAuthorizationOwnershipAndSeedReadOnly(t *testing.T) {
	f := newJobTestFixture(t)
	created := f.create(t, f.ownerToken, jobTestPayload())
	routes := []struct{ method, path string }{
		{"GET", "/api/jobs"}, {"GET", "/api/jobs/" + created.ID},
		{"POST", "/api/mitra/jobs"}, {"GET", "/api/mitra/jobs"},
		{"GET", "/api/mitra/jobs/" + created.ID}, {"PATCH", "/api/mitra/jobs/" + created.ID},
		{"PUT", "/api/mitra/jobs/" + created.ID}, {"DELETE", "/api/mitra/jobs/" + created.ID},
	}
	for _, route := range routes {
		t.Run("missing_auth/"+route.method+route.path, func(t *testing.T) {
			f.request(t, route.method, route.path, "", jobTestPayload(), 401, nil)
		})
		if strings.HasPrefix(route.path, "/api/mitra/") {
			for role, token := range map[string]string{"student": f.studentToken, "admin": f.adminToken} {
				t.Run(role+"/"+route.method+route.path, func(t *testing.T) {
					f.request(t, route.method, route.path, token, jobTestPayload(), 403, nil)
				})
			}
		} else {
			for _, token := range []string{f.studentToken, f.adminToken, f.otherToken} {
				f.request(t, route.method, route.path, token, nil, 200, nil)
			}
		}
	}
	f.request(t, "GET", "/api/jobs", "invalid-token", nil, 401, nil)
	for _, method := range []string{"GET", "PATCH", "PUT", "DELETE"} {
		f.request(t, method, "/api/mitra/jobs/"+created.ID, f.otherToken, jobTestPayload(), 404, nil)
	}
	require.Equal(t, created.Title, f.stored(t, created.ID).Title)
	require.Empty(t, f.list(t, "/api/mitra/jobs", f.otherToken).Data)
	for _, id := range []string{uuid.NewString(), "not-a-uuid"} {
		f.request(t, "GET", "/api/jobs/"+id, f.ownerToken, nil, 404, nil)
		for _, method := range []string{"GET", "PATCH", "PUT", "DELETE"} {
			f.request(t, method, "/api/mitra/jobs/"+id, f.ownerToken, jobTestPayload(), 404, nil)
		}
	}
	seed := entity.Job{
		ID:             uuid.NewString(),
		MitraID:        nil,
		Title:          "Seed Catalog Job",
		Category:       "Backend Developer Intern",
		Company:        "Catalog Company",
		Description:    "Seed job without mitra owner.",
		Location:       "Jakarta",
		WorkType:       "Hybrid",
		Source:         "seed",
		RequiredSkills: []string{"Go"},
		OptionalSkills: []string{},
	}
	require.NoError(t, f.db.Create(&seed).Error)
	seed = f.stored(t, seed.ID)

	var seeds []entity.Job
	require.NoError(t, f.db.Where("mitra_id IS NULL").Find(&seeds).Error)
	require.NotEmpty(t, seeds, "Catalog should contain seed jobs")
	f.request(t, "GET", "/api/jobs/"+seed.ID, f.studentToken, nil, 200, nil)
	for _, method := range []string{"GET", "PATCH", "PUT", "DELETE"} {
		f.request(t, method, "/api/mitra/jobs/"+seed.ID, f.ownerToken, jobTestPayload(), 404, nil)
	}
	require.Equal(t, seed, f.stored(t, seed.ID))
}

func TestJobsValidation(t *testing.T) {
	f := newJobTestFixture(t)
	created := f.create(t, f.ownerToken, jobTestPayload())
	beforeInvalidRequests := f.stored(t, created.ID)
	invalid := []struct {
		name, field string
		value       any
	}{
		{"title_empty", "title", "   "}, {"title_short", "title", "ab"}, {"title_long", "title", strings.Repeat("x", 151)},
		{"category_empty", "category", ""}, {"category_unknown", "category", "Backend"},
		{"description_short", "description", "Too short"}, {"description_long", "description", strings.Repeat("x", 10001)},
		{"location_empty", "location", "  "}, {"location_long", "location", strings.Repeat("x", 256)},
		{"work_type_empty", "work_type", " "}, {"work_type_long", "work_type", strings.Repeat("x", 51)},
		{"required_empty", "required_skills", []string{}}, {"required_blank", "required_skills", []string{" "}},
		{"required_long_skill", "required_skills", []string{strings.Repeat("x", 101)}},
		{"required_too_many_duplicates", "required_skills", strings.Fields(strings.Repeat("Go ", 51))},
		{"optional_blank", "optional_skills", []string{" "}}, {"optional_long_skill", "optional_skills", []string{strings.Repeat("x", 101)}},
		{"optional_too_many_duplicates", "optional_skills", strings.Fields(strings.Repeat("Go ", 51))},
		{"url_relative", "source_url", "/jobs/1"}, {"url_scheme", "source_url", "javascript:alert(1)"},
		{"url_ftp", "source_url", "ftp://example.org/job"}, {"url_host_missing", "source_url", "https:///jobs"},
		{"url_long", "source_url", "https://example.org/" + strings.Repeat("x", 1001)},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			payload := jobTestPayload()
			payload[tc.field] = tc.value
			f.request(t, "POST", "/api/mitra/jobs", f.ownerToken, payload, 400, nil)
			for _, method := range []string{"PATCH", "PUT"} {
				f.request(t, method, "/api/mitra/jobs/"+created.ID, f.ownerToken, map[string]any{tc.field: tc.value}, 400, nil)
			}
		})
	}
	for _, field := range []string{"title", "category", "description", "location", "work_type", "required_skills"} {
		payload := jobTestPayload()
		delete(payload, field)
		f.request(t, "POST", "/api/mitra/jobs", f.ownerToken, payload, 400, nil)
		payload[field] = nil
		f.request(t, "POST", "/api/mitra/jobs", f.ownerToken, payload, 400, nil)
	}
	for _, body := range []string{`{`, `[]`, `{"title":42}`, `{"required_skills":"Go"}`, `{"required_skills":[42]}`} {
		f.request(t, "POST", "/api/mitra/jobs", f.ownerToken, body, 400, nil)
		for _, method := range []string{"PATCH", "PUT"} {
			f.request(t, method, "/api/mitra/jobs/"+created.ID, f.ownerToken, body, 400, nil)
		}
	}
	stored := f.stored(t, created.ID)
	require.Equal(t, created.Title, stored.Title)
	require.Equal(t, created.RequiredSkills, stored.RequiredSkills)
	require.Equal(t, created.OptionalSkills, stored.OptionalSkills)
	require.Equal(t, beforeInvalidRequests, stored, "invalid requests must not mutate persisted job")
	require.EqualValues(t, 1, f.list(t, "/api/mitra/jobs", f.ownerToken).Paging.TotalItem)
	// Optional fields may be absent, and both HTTP URL schemes are valid.
	for _, sourceURL := range []string{"", "http://example.org/job", "https://example.org/job"} {
		payload := jobTestPayload()
		delete(payload, "optional_skills")
		payload["source_url"] = sourceURL
		job := f.create(t, f.ownerToken, payload)
		require.Empty(t, job.OptionalSkills)
		require.NotNil(t, job.OptionalSkills)
	}
}

func TestJobsPaginationAndLiteralFilters(t *testing.T) {
	f := newJobTestFixture(t)
	var baseline int64
	require.NoError(t, f.db.Model(&entity.Job{}).Count(&baseline).Error)
	var ids []string
	stamp := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 12; i++ {
		payload := jobTestPayload()
		payload["title"] = fmt.Sprintf("CatalogTest Backend %02d", i)
		if i%2 == 1 {
			payload["category"] = "Frontend Developer Intern"
		}
		job := f.create(t, f.ownerToken, payload)
		ids = append(ids, job.ID)
		require.NoError(t, f.db.Model(&entity.Job{}).Where("id = ?", job.ID).Update("created_at", stamp.Add(time.Duration(i)*time.Second)).Error)
	}
	otherPayload := jobTestPayload()
	otherPayload["title"] = "CatalogTest Other Owner"
	f.create(t, f.otherToken, otherPayload)
	defaults := f.list(t, "/api/mitra/jobs", f.ownerToken)
	require.Equal(t, &model.PageMetadata{Page: 1, Size: 10, TotalItem: 12, TotalPage: 2}, defaults.Paging)
	require.Len(t, defaults.Data, 10)
	for i, job := range defaults.Data {
		require.Equal(t, ids[11-i], job.ID)
	}
	second := f.list(t, "/api/mitra/jobs?page=2&size=10", f.ownerToken)
	require.Len(t, second.Data, 2)
	require.Equal(t, ids[1], second.Data[0].ID)
	require.Equal(t, ids[0], second.Data[1].ID)
	catalog := f.list(t, "/api/jobs", f.studentToken)
	require.EqualValues(t, baseline+13, catalog.Paging.TotalItem)
	require.Equal(t, 1, catalog.Paging.Page)
	require.Equal(t, 10, catalog.Paging.Size)
	filtered := f.list(t, "/api/jobs?search=cAtAlOgTeSt&category="+url.QueryEscape(" Backend Developer Intern ")+"&size=100", f.studentToken)
	require.Len(t, filtered.Data, 7)
	require.EqualValues(t, 7, filtered.Paging.TotalItem)
	require.EqualValues(t, 1, filtered.Paging.TotalPage)
	ownFiltered := f.list(t, "/api/mitra/jobs?search=CatalogTest&category="+url.QueryEscape("Backend Developer Intern")+"&size=2&page=2", f.ownerToken)
	require.Len(t, ownFiltered.Data, 2)
	require.EqualValues(t, 6, ownFiltered.Paging.TotalItem)
	require.EqualValues(t, 3, ownFiltered.Paging.TotalPage)
	byCompany := f.list(t, "/api/jobs?search="+url.QueryEscape("  other TEST company  "), f.studentToken)
	require.EqualValues(t, 1, byCompany.Paging.TotalItem)
	require.Equal(t, f.other.Name, byCompany.Data[0].Company)
	for _, path := range []string{"/api/jobs", "/api/mitra/jobs"} {
		for _, query := range []string{"page=0", "page=-1", "page=abc", "page=1.5", "page=", "page=9223372036854775808", "page=9223372036854775807&size=100", "size=0", "size=-1", "size=abc", "size=1.5", "size=", "size=101", "size=999999999999999999999999", "category=unknown"} {
			t.Run(path+"?"+query, func(t *testing.T) {
				f.request(t, "GET", path+"?"+query, f.ownerToken, nil, 400, nil)
			})
		}
		empty := f.list(t, path+"?search=NoSuchJobUniqueToken", f.ownerToken)
		require.Empty(t, empty.Data)
		require.Zero(t, empty.Paging.TotalItem)
		require.Zero(t, empty.Paging.TotalPage)
		beyond := f.list(t, path+"?search=CatalogTest&page=100&size=100", f.ownerToken)
		require.Empty(t, beyond.Data)
		require.Positive(t, beyond.Paging.TotalItem)
	}
	for _, literal := range []string{"%", "_", `\`, `%' OR 1=1 --`} {
		payload := jobTestPayload()
		payload["title"] = "LiteralSearch " + literal + " marker"
		job := f.create(t, f.ownerToken, payload)
		for _, path := range []string{"/api/jobs", "/api/mitra/jobs"} {
			result := f.list(t, path+"?search="+url.QueryEscape(literal), f.ownerToken)
			require.EqualValues(t, 1, result.Paging.TotalItem, "literal %q", literal)
			require.Len(t, result.Data, 1)
			require.Equal(t, job.ID, result.Data[0].ID)
		}
	}
	// Equal timestamps must use ID as deterministic tie-breaker.
	tieA := f.create(t, f.ownerToken, jobTestPayload())
	tieB := f.create(t, f.ownerToken, jobTestPayload())
	require.NoError(t, f.db.Model(&entity.Job{}).Where("id IN ?", []string{tieA.ID, tieB.ID}).Update("created_at", stamp.Add(time.Hour)).Error)
	first := f.list(t, "/api/mitra/jobs?size=1", f.ownerToken)
	next := f.list(t, "/api/mitra/jobs?size=1&page=2", f.ownerToken)
	if tieA.ID > tieB.ID {
		tieA, tieB = tieB, tieA
	}
	require.Equal(t, tieA.ID, first.Data[0].ID)
	require.Equal(t, tieB.ID, next.Data[0].ID)
}
