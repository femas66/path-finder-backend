package test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	controller "haji-koding/path-finder/internal/delivery/http"
	"haji-koding/path-finder/internal/delivery/http/middleware"
	"haji-koding/path-finder/internal/delivery/http/route"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
	"haji-koding/path-finder/internal/repository"
	"haji-koding/path-finder/internal/service"
	"haji-koding/path-finder/internal/usecase"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

const recommendationConsent = "2026-11-26T00:00:00Z"

// Reuse schema isolation only; Bootstrap's disabled ML client and random JWT secret
// are deliberately not reused by the recommendation route.
func recommendationTestApp(t *testing.T, f *jobTestFixture, url string) (*jobTestFixture, func(string, string) string) {
	t.Helper()
	settings := viper.New()
	secret := "recommendation-integration-only-secret-" + uuid.NewString()
	settings.Set("jwt.secret", secret)
	settings.Set("ml.service_url", url)
	settings.Set("ml.service_token", "python-internal-secret")
	settings.Set("ml.request_timeout_seconds", 1)
	app := fiber.New()
	uc := usecase.NewRecommendationUseCase(f.db, f.log, repository.NewStoryRepository(f.log), repository.NewJobRepository(f.log), service.NewMLService(settings, f.log))
	routes := route.RouteConfig{App: app, RecommendationController: controller.NewRecommendationController(uc, f.log), AuthMiddleware: middleware.NewAuth(settings), MahasiswaMiddleware: middleware.RequireRole(entity.RoleMahasiswa)}
	// SetupAuthRoute also registers unrelated handlers; register the production
	// recommendation chain directly without unrelated controllers.
	app.Post("/api/recommendations", routes.AuthMiddleware, routes.MahasiswaMiddleware, routes.RecommendationController.Recommend)
	t.Cleanup(func() { require.NoError(t, app.Shutdown()) })
	sign := func(id, role string) string {
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": id, "role": role, "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
		require.NoError(t, err)
		return token
	}
	return &jobTestFixture{app: app, db: f.db, log: f.log}, sign
}
func recommendationResponse(id, status string, items []any) map[string]any {
	return map[string]any{"student_id": id, "generated_at": recommendationConsent, "schema_version": "1.0", "model_version": "ner:test;encoder:test;ranker:test", "top_k": 3, "status": status, "items": items}
}
func recommendationStory(t *testing.T, f *jobTestFixture, id, content string, stamp time.Time) {
	t.Helper()
	require.NoError(t, f.db.Create(&entity.Story{ID: uuid.NewString(), UserID: id, Content: content, CreatedAt: stamp}).Error)
}
func TestRecommendationIntegrationMappingAndIdentity(t *testing.T) {
	f := newJobTestFixture(t)
	// Bootstrap may seed the catalog; remove it only inside this private schema.
	require.NoError(t, f.db.Where("1 = 1").Delete(&entity.Job{}).Error)
	stamp := time.Date(2026, 11, 25, 12, 0, 0, 123456000, time.FixedZone("test", 7*3600))
	recommendationStory(t, f, f.student.ID, "old student narrative", stamp.Add(-time.Hour))
	recommendationStory(t, f, f.student.ID, "latest student narrative", stamp)
	recommendationStory(t, f, f.other.ID, "another user's newer narrative", stamp.Add(time.Hour))
	job := entity.Job{ID: uuid.NewString(), Title: "Backend Intern", Category: "Backend Developer Intern", Description: "Build Go services", Source: "Glints", SourceURL: "https://example.org/jobs/1", RequiredSkills: []string{"Go"}, CreatedAt: stamp}
	require.NoError(t, f.db.Create(&job).Error)
	closed := job
	closed.ID = uuid.NewString()
	closed.Status = entity.JobStatusDitutup
	closed.Source = "mitra"
	require.NoError(t, f.db.Create(&closed).Error)
	type observed struct {
		method, path, auth string
		input              model.MLRecommendationRequest
		raw                map[string]json.RawMessage
		err                error
	}
	calls := make(chan observed, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		var input model.MLRecommendationRequest
		if err == nil {
			err = json.Unmarshal(raw, &input)
		}
		var fields map[string]json.RawMessage
		if err == nil {
			err = json.Unmarshal(raw, &fields)
		}
		calls <- observed{r.Method, r.URL.Path, r.Header.Get("Authorization"), input, fields, err}
		_ = json.NewEncoder(w).Encode(recommendationResponse(input.Candidate.StudentID, "parsial", []any{map[string]any{"job_id": job.ID, "title": job.Title, "matching_score": nil, "cosine_similarity": nil, "readiness_score": nil, "matched_skills": []string{}, "missing_skills": []string{}, "score_status": "skor_belum_tersedia", "rank": nil}}))
	}))
	defer server.Close()
	app, sign := recommendationTestApp(t, f, server.URL)
	token := sign(f.student.ID, entity.RoleMahasiswa)
	var result model.WebResponse[model.RecommendationResponse]
	raw := app.request(t, "POST", "/api/recommendations", token, map[string]string{"consent_at": recommendationConsent}, 200, &result)
	got := <-calls
	require.NoError(t, got.err)
	require.Equal(t, "POST", got.method)
	require.Equal(t, "/recommend", got.path)
	require.Equal(t, "Bearer python-internal-secret", got.auth)
	require.NotContains(t, got.auth, token)
	require.Equal(t, model.MLCandidate{StudentID: f.student.ID, Narrative: "latest student narrative", ConsentAt: recommendationConsent}, got.input.Candidate)
	require.Equal(t, []model.MLJob{{JobID: job.ID, Title: job.Title, Category: job.Category, Description: job.Description, RequiredSkills: []string{"Go"}, OptionalSkills: []string{}, Source: job.Source, SourceURL: job.SourceURL, CollectedAt: stamp.UTC().Format(time.RFC3339Nano)}}, got.input.Jobs)
	var candidate map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(got.raw["candidate"], &candidate))
	require.JSONEq(t, `null`, string(candidate["cv_text"]))
	require.JSONEq(t, `null`, string(candidate["manual_profile"]))
	require.Len(t, candidate, 5)
	require.Equal(t, f.student.ID, result.Data.StudentID)
	require.Nil(t, result.Data.Items[0].MatchingScore)
	require.Contains(t, string(raw), `"matching_score":null`)
	require.Contains(t, string(raw), `"rank":null`)
	for _, tc := range []struct {
		name, token, body string
		status            int
	}{
		{"missing_auth", "", `{"consent_at":"` + recommendationConsent + `"}`, 401}, {"invalid_auth", "invalid", `{}`, 401}, {"mitra", sign(f.owner.ID, entity.RoleMitra), `{}`, 403}, {"admin", sign(f.student.ID, entity.RoleAdmin), `{}`, 403}, {"missing_id", sign("", entity.RoleMahasiswa), `{}`, 401},
		{"spoof_student", token, `{"consent_at":"` + recommendationConsent + `","student_id":"` + f.other.ID + `"}`, 400}, {"spoof_narrative", token, `{"consent_at":"` + recommendationConsent + `","narrative":"fake"}`, 400}, {"fake_cv", token, `{"consent_at":"` + recommendationConsent + `","cv_text":"fake"}`, 400}, {"duplicate", token, `{"consent_at":"` + recommendationConsent + `","consent_at":"` + recommendationConsent + `"}`, 400}, {"null", token, `{"consent_at":null}`, 400}, {"trailing", token, `{"consent_at":"` + recommendationConsent + `"} {}`, 400}, {"invalid_date", token, `{"consent_at":"today"}`, 400}, {"missing_consent", token, `{}`, 400},
	} {
		t.Run(tc.name, func(t *testing.T) { app.request(t, "POST", "/api/recommendations", tc.token, tc.body, tc.status, nil) })
	}
	require.Empty(t, calls, "rejected requests must not reach Python")
	require.NoError(t, f.db.Model(&job).Update("source_url", "").Error)
	app.request(t, "POST", "/api/recommendations", token, map[string]string{"consent_at": recommendationConsent}, 422, nil)
	require.Empty(t, calls, "external blank URL must not be fabricated or silently omitted")
}

// Exercises real authenticated job creation and recommendation handlers; only ML inference is mocked.
func TestRecommendationIntegrationCreatedMitraJob(t *testing.T) {
	f := newJobTestFixture(t)
	require.NoError(t, f.db.Where("1 = 1").Delete(&entity.Job{}).Error)
	payload := jobTestPayload()
	delete(payload, "source_url")
	f.request(t, "POST", "/api/mitra/jobs", "", payload, 401, nil)
	f.request(t, "POST", "/api/mitra/jobs", f.studentToken, payload, 403, nil)
	created := f.create(t, f.ownerToken, payload)
	stored := f.stored(t, created.ID)
	require.Equal(t, "mitra", stored.Source)
	require.Empty(t, stored.SourceURL)
	recommendationStory(t, f, f.student.ID, "Student backend narrative", time.Now())
	inputs := make(chan model.MLRecommendationRequest, 1)
	auth := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input model.MLRecommendationRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		inputs <- input
		auth <- r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(recommendationResponse(input.Candidate.StudentID, "ok", []any{}))
	}))
	defer server.Close()
	app, sign := recommendationTestApp(t, f, server.URL)
	body := map[string]string{"consent_at": recommendationConsent}
	app.request(t, "POST", "/api/recommendations", "", body, 401, nil)
	app.request(t, "POST", "/api/recommendations", sign(f.owner.ID, entity.RoleMitra), body, 403, nil)
	require.Empty(t, inputs)
	token := sign(f.student.ID, entity.RoleMahasiswa)
	app.request(t, "POST", "/api/recommendations", token, body, 200, nil)
	input := <-inputs
	require.Equal(t, f.student.ID, input.Candidate.StudentID)
	require.Equal(t, []model.MLJob{{JobID: stored.ID, Title: stored.Title, Category: stored.Category,
		Description: stored.Description, RequiredSkills: stored.RequiredSkills, OptionalSkills: stored.OptionalSkills,
		Source: "mitra", SourceURL: "", CollectedAt: stored.CreatedAt.UTC().Format(time.RFC3339Nano)}}, input.Jobs)
	require.Equal(t, created.ID, input.Jobs[0].JobID)
	require.Equal(t, "Bearer python-internal-secret", <-auth)
	for _, source := range []string{"Glints", "Dealls", "Jobstreet", "LinkedIn"} {
		require.NoError(t, f.db.Model(&stored).Update("source", source).Error)
		app.request(t, "POST", "/api/recommendations", token, body, 422, nil)
		require.Empty(t, inputs, "external blank URL must be rejected before ML")
	}
}

func TestRecommendationIntegrationEmptyCatalogAndIncompleteProfile(t *testing.T) {
	for _, status := range []string{"ok", "profil_tidak_lengkap"} {
		t.Run(status, func(t *testing.T) {
			f := newJobTestFixture(t)
			require.NoError(t, f.db.Where("1 = 1").Delete(&entity.Job{}).Error)
			recommendationStory(t, f, f.student.ID, "Short narrative", time.Now())
			inputs := make(chan model.MLRecommendationRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var input model.MLRecommendationRequest
				_ = json.NewDecoder(r.Body).Decode(&input)
				inputs <- input
				_ = json.NewEncoder(w).Encode(recommendationResponse(input.Candidate.StudentID, status, []any{}))
			}))
			defer server.Close()
			app, sign := recommendationTestApp(t, f, server.URL)
			var response model.WebResponse[model.RecommendationResponse]
			app.request(t, "POST", "/api/recommendations", sign(f.student.ID, entity.RoleMahasiswa), map[string]string{"consent_at": recommendationConsent}, 200, &response)
			input := <-inputs
			require.NotNil(t, input.Jobs)
			require.Empty(t, input.Jobs)
			require.Equal(t, status, response.Data.Status)
			require.NotNil(t, response.Data.Items)
			require.Empty(t, response.Data.Items)
		})
	}
}
func TestRecommendationIntegrationServiceFailures(t *testing.T) {
	f := newJobTestFixture(t)
	require.NoError(t, f.db.Where("1 = 1").Delete(&entity.Job{}).Error)
	recommendationStory(t, f, f.student.ID, "Student narrative", time.Now())
	var calls atomic.Int32
	for _, tc := range []struct {
		name           string
		upstream, code int
		message        string
	}{{"missing_config", 0, 503, "ML service unavailable"}, {"bad_input", 400, 422, "Invalid recommendation input"}, {"busy", 429, 503, "ML service unavailable"}, {"internal_auth", 401, 503, "ML service unavailable"}, {"unavailable", 503, 503, "ML service unavailable"}, {"timeout", -1, 504, "ML service timed out"}, {"malformed", 200, 502, "Invalid ML response"}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if tc.upstream == -1 {
					select {
					case <-r.Context().Done():
					case <-time.After(1500 * time.Millisecond):
					}
					return
				}
				w.WriteHeader(tc.upstream)
				_, _ = io.WriteString(w, "private Python details")
			}))
			defer server.Close()
			url := server.URL
			if tc.upstream == 0 {
				url = ""
			}
			app, sign := recommendationTestApp(t, f, url)
			var response model.WebResponse[json.RawMessage]
			request := httptest.NewRequest(http.MethodPost, "/api/recommendations", strings.NewReader(`{"consent_at":"`+recommendationConsent+`"}`))
			request.Header.Set("Authorization", "Bearer "+sign(f.student.ID, entity.RoleMahasiswa))
			request.Header.Set("Content-Type", "application/json")
			// Fiber defaults to one second, equal to the upstream timeout under test.
			result, err := app.app.Test(request, fiber.TestConfig{Timeout: 3 * time.Second})
			require.NoError(t, err)
			defer result.Body.Close()
			require.Equal(t, tc.code, result.StatusCode)
			require.NoError(t, json.NewDecoder(result.Body).Decode(&response))
			require.False(t, response.Success)
			require.NotEmpty(t, response.Errors)
			require.Equal(t, tc.message, response.Message)
			require.NotContains(t, response.Message, "Python")
		})
	}
	require.EqualValues(t, 6, calls.Load())
	app, sign := recommendationTestApp(t, f, "")
	app.request(t, "POST", "/api/recommendations", sign(f.other.ID, entity.RoleMahasiswa), map[string]string{"consent_at": recommendationConsent}, 404, nil)
	app.request(t, "POST", "/api/recommendations", sign(f.student.ID, entity.RoleMahasiswa), strings.Repeat("x", service.MLPayloadLimit+1), 413, nil)
}
