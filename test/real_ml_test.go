package test

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	controller "haji-koding/path-finder/internal/delivery/http"
	"haji-koding/path-finder/internal/delivery/http/middleware"
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

// Opt-in real inference; all DB rows are synthetic fixtures in a disposable schema.

func TestRealMLRecommendation(t *testing.T) {
	upstream := os.Getenv("PATHFINDER_REAL_ML_URL")
	if upstream == "" {
		t.Skip("set PATHFINDER_REAL_ML_URL, PATHFINDER_REAL_ML_TOKEN, PATHFINDER_JOB_TEST_DSN")
	}
	require.NotEmpty(t, os.Getenv("PATHFINDER_REAL_ML_TOKEN"))
	require.NotEmpty(t, os.Getenv("PATHFINDER_JOB_TEST_DSN"))
	f := newJobTestFixture(t)
	require.NoError(t, f.db.Where("1 = 1").Delete(&entity.Job{}).Error)
	recommendationStory(t, f, f.student.ID, "Saya menguasai Python, SQL, dan Git serta mampu bekerja sama dalam tim untuk membangun aplikasi backend.", time.Now())
	recommendationStory(t, f, f.other.ID, "Unrelated user's newer story", time.Now().Add(time.Hour))
	ids := map[string]bool{}
	for _, skills := range [][]string{{"Python", "SQL"}, {"Java"}, {"Git"}, {"React"}} {
		payload := jobTestPayload()
		delete(payload, "source_url")
		payload["title"] = "Synthetic smoke Python Intern"
		payload["description"] = "Synthetic integration fixture, not a production vacancy"
		payload["required_skills"] = skills
		payload["optional_skills"] = []string{}
		created := f.create(t, f.ownerToken, payload)
		stored := f.stored(t, created.ID)
		require.Equal(t, "mitra", stored.Source)
		require.Empty(t, stored.SourceURL)
		require.Equal(t, &f.owner.ID, stored.MitraID)
		require.Equal(t, f.owner.Name, stored.Company)
		ids[created.ID] = true
	}
	closed := f.create(t, f.ownerToken, jobTestPayload())
	f.request(t, "PATCH", "/api/mitra/jobs/"+closed.ID, f.ownerToken, map[string]string{"status": entity.JobStatusDitutup}, 200, nil)
	settings := viper.New()
	secret := uuid.NewString()
	settings.Set("jwt.secret", secret)
	settings.Set("ml.service_url", upstream)
	settings.Set("ml.service_token", os.Getenv("PATHFINDER_REAL_ML_TOKEN"))
	settings.Set("ml.request_timeout_seconds", 60)
	app := fiber.New()
	uc := usecase.NewRecommendationUseCase(f.db, f.log, repository.NewStoryRepository(f.log), repository.NewJobRepository(f.log), service.NewMLService(settings, f.log))
	app.Post("/api/recommendations", middleware.NewAuth(settings), middleware.RequireRole(entity.RoleMahasiswa), controller.NewRecommendationController(uc, f.log).Recommend)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- app.Listener(listener) }()
	t.Cleanup(func() { require.NoError(t, app.Shutdown()); require.NoError(t, <-done) })
	client := &http.Client{Timeout: 90 * time.Second}
	request := func(token string, status int) map[string]json.RawMessage {
		req, err := http.NewRequest("POST", "http://"+listener.Addr().String()+"/api/recommendations", strings.NewReader(`{"consent_at":"`+recommendationConsent+`"}`))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, status, resp.StatusCode, "%s", raw)
		var envelope map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &envelope))
		return envelope
	}
	sign := func(role string) string {
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": f.student.ID, "role": role, "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
		require.NoError(t, err)
		return token
	}
	request("", 401)
	request(sign(entity.RoleMitra), 403)
	envelope := request(sign(entity.RoleMahasiswa), 200)
	var result model.RecommendationResponse
	require.NoError(t, json.Unmarshal(envelope["data"], &result))
	require.Equal(t, f.student.ID, result.StudentID)
	require.Equal(t, "1.0", result.SchemaVersion)
	require.NotEmpty(t, result.GeneratedAt)
	require.Equal(t, "ok", result.Status)
	require.Equal(t, 3, result.TopK)
	require.Contains(t, result.ModelVersion, "ranker:gb-e5-reviewed-dev-v1")
	require.Contains(t, result.ModelVersion, "intfloat/multilingual-e5-small")
	require.Len(t, result.Items, 3)
	seen := map[string]bool{}
	for i, item := range result.Items {
		require.True(t, ids[item.JobID])
		require.False(t, seen[item.JobID])
		seen[item.JobID] = true
		require.NotNil(t, item.Rank)
		require.Equal(t, i+1, *item.Rank)
		require.NotNil(t, item.MatchingScore)
		require.GreaterOrEqual(t, *item.MatchingScore, 0)
		require.LessOrEqual(t, *item.MatchingScore, 100)
		require.NotNil(t, item.ReadinessScore)
		require.GreaterOrEqual(t, *item.ReadinessScore, 0)
		require.LessOrEqual(t, *item.ReadinessScore, 100)
		require.NotNil(t, item.MatchedSkills)
		require.NotNil(t, item.MissingSkills)
	}
	// Same persisted catalog, no recognized skills: real pipeline must preserve null scores.
	require.NoError(t, f.db.Model(&entity.Story{}).Where("user_id = ?", f.student.ID).Update("content", "Saya ingin belajar dan mencari pengalaman baru dalam dunia kerja.").Error)
	partial := request(sign(entity.RoleMahasiswa), 200)
	var fields struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	require.NoError(t, json.Unmarshal(partial["data"], &fields))
	require.Len(t, fields.Items, 3)
	for _, item := range fields.Items {
		for _, key := range []string{"matching_score", "cosine_similarity", "readiness_score", "rank"} {
			require.Contains(t, item, key)
			require.JSONEq(t, "null", string(item[key]), key)
		}
	}
	t.Logf("real HTTP mitra-created catalog/Go/Python/NER/E5/GB/PostgreSQL passed; model=%s; top_k=%d; incomplete_items=%d", result.ModelVersion, result.TopK, len(fields.Items))
}
