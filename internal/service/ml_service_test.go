package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"haji-koding/path-finder/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func mlTestInput() *model.MLRecommendationRequest {
	return &model.MLRecommendationRequest{Candidate: model.MLCandidate{StudentID: "student-1", Narrative: "I build Go services", ConsentAt: "2026-11-26T00:00:00Z"}, Jobs: []model.MLJob{{JobID: "job-1"}}}
}
func mlTestResponse() map[string]any {
	return map[string]any{"student_id": "student-1", "generated_at": "2026-11-26T00:00:00Z", "schema_version": "1.0", "model_version": "ner:test;encoder:test;ranker:test", "top_k": 3, "status": "parsial", "items": []any{map[string]any{"job_id": "job-1", "title": "Backend Intern", "matching_score": nil, "cosine_similarity": nil, "readiness_score": nil, "matched_skills": []string{}, "missing_skills": []string{}, "score_status": "skor_belum_tersedia", "rank": nil}}}
}
func mlTestService(url string) *MLService {
	config := viper.New()
	config.Set("ml.service_url", url)
	config.Set("ml.service_token", "internal-service-secret")
	return NewMLService(config, nil)
}
func requireMLError(t *testing.T, err error, code int, message string) {
	t.Helper()
	var public *fiber.Error
	require.ErrorAs(t, err, &public)
	require.Equal(t, code, public.Code)
	require.Equal(t, message, public.Message)
}
func TestMLServiceRequestAndNullableResponse(t *testing.T) {
	received := make(chan *http.Request, 1)
	payload := make(chan map[string]json.RawMessage, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received <- r.Clone(context.Background())
		raw, _ := io.ReadAll(r.Body)
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(raw, &fields)
		payload <- fields
		_ = json.NewEncoder(w).Encode(mlTestResponse())
	}))
	defer server.Close()
	result, err := mlTestService(server.URL).Recommend(context.Background(), mlTestInput())
	require.NoError(t, err)
	request := <-received
	require.Equal(t, "POST", request.Method)
	require.Equal(t, "/recommend", request.URL.Path)
	require.Equal(t, "Bearer internal-service-secret", request.Header.Get("Authorization"))
	require.Equal(t, "application/json", request.Header.Get("Content-Type"))
	var candidate map[string]json.RawMessage
	require.NoError(t, json.Unmarshal((<-payload)["candidate"], &candidate))
	require.JSONEq(t, `null`, string(candidate["cv_text"]))
	require.JSONEq(t, `null`, string(candidate["manual_profile"]))
	require.Len(t, result.Items, 1)
	item := result.Items[0]
	require.Nil(t, item.MatchingScore)
	require.Nil(t, item.CosineSimilarity)
	require.Nil(t, item.ReadinessScore)
	require.Nil(t, item.Rank)
	require.NotNil(t, item.MatchedSkills)
	require.NotNil(t, item.MissingSkills)
	encoded, err := json.Marshal(item)
	require.NoError(t, err)
	require.JSONEq(t, `{"job_id":"job-1","title":"Backend Intern","matching_score":null,"cosine_similarity":null,"readiness_score":null,"matched_skills":[],"missing_skills":[],"score_status":"skor_belum_tersedia","rank":null}`, string(encoded))
}
func TestMLServiceConfiguration(t *testing.T) {
	for _, tc := range []struct {
		name, key string
		value     any
	}{
		{"missing_url", "ml.service_url", ""}, {"missing_token", "ml.service_token", ""}, {"credentials", "ml.service_url", "http://user:pass@example.org"}, {"path", "ml.service_url", "http://example.org/recommend"}, {"query", "ml.service_url", "http://example.org?q=1"}, {"fragment", "ml.service_url", "http://example.org/#x"}, {"scheme", "ml.service_url", "ftp://example.org"}, {"whitespace_token", "ml.service_token", " secret"}, {"newline_token", "ml.service_token", "secret\n"}, {"timeout_zero", "ml.request_timeout_seconds", 0}, {"timeout_large", "ml.request_timeout_seconds", 121}, {"timeout_text", "ml.request_timeout_seconds", "bad"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := viper.New()
			config.Set("ml.service_url", "http://example.org")
			config.Set("ml.service_token", "secret")
			config.Set(tc.key, tc.value)
			result, err := NewMLService(config, nil).Recommend(context.Background(), mlTestInput())
			require.Nil(t, result)
			requireMLError(t, err, 503, "ML service unavailable")
		})
	}
	var disabled *MLService
	_, err := disabled.Recommend(context.Background(), mlTestInput())
	requireMLError(t, err, 503, "ML service unavailable")
}
func TestMLServiceUpstreamErrors(t *testing.T) {
	for _, tc := range []struct {
		status, code int
		message      string
	}{{400, 422, "Invalid recommendation input"}, {413, 413, "Recommendation payload exceeds 1 MiB"}, {401, 503, "ML service unavailable"}, {403, 503, "ML service unavailable"}, {429, 503, "ML service unavailable"}, {500, 503, "ML service unavailable"}, {503, 503, "ML service unavailable"}} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, "private upstream details")
			}))
			defer server.Close()
			result, err := mlTestService(server.URL).Recommend(context.Background(), mlTestInput())
			require.Nil(t, result)
			requireMLError(t, err, tc.code, tc.message)
		})
	}
	t.Run("unreachable", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		server.Close()
		_, err := mlTestService(server.URL).Recommend(context.Background(), mlTestInput())
		requireMLError(t, err, 503, "ML service unavailable")
	})
	t.Run("timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
			case <-time.After(1500 * time.Millisecond):
			}
		}))
		defer server.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		_, err := mlTestService(server.URL).Recommend(ctx, mlTestInput())
		requireMLError(t, err, 504, "ML service timed out")
	})
	t.Run("redirect_not_followed", func(t *testing.T) {
		hits := 0
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
		defer target.Close()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
		defer server.Close()
		_, err := mlTestService(server.URL).Recommend(context.Background(), mlTestInput())
		requireMLError(t, err, 503, "ML service unavailable")
		require.Zero(t, hits)
	})
	t.Run("oversized_request", func(t *testing.T) {
		input := mlTestInput()
		input.Candidate.Narrative = strings.Repeat("x", MLPayloadLimit)
		_, err := mlTestService("http://127.0.0.1:1").Recommend(context.Background(), input)
		requireMLError(t, err, 413, "Recommendation payload exceeds 1 MiB")
	})
}
func TestMLServiceResponseContract(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"wrong_student", func(r map[string]any) { r["student_id"] = "other" }}, {"wrong_top_k", func(r map[string]any) { r["top_k"] = 4 }}, {"wrong_schema", func(r map[string]any) { r["schema_version"] = "2.0" }}, {"invalid_timestamp", func(r map[string]any) { r["generated_at"] = "today" }}, {"missing_model_component", func(r map[string]any) { r["model_version"] = "ner:test;encoder:test" }}, {"unknown_status", func(r map[string]any) { r["status"] = "success" }}, {"null_items", func(r map[string]any) { r["items"] = nil }}, {"missing_field", func(r map[string]any) { delete(r, "top_k") }}, {"unknown_field", func(r map[string]any) { r["private"] = true }}, {"incomplete_with_items", func(r map[string]any) { r["status"] = "profil_tidak_lengkap" }},
	}
	for _, tc := range []struct {
		name, key string
		value     any
	}{{"unknown_job", "job_id", "other"}, {"blank_title", "title", " "}, {"negative_score", "matching_score", -1}, {"score_above_100", "matching_score", 101}, {"cosine_above_one", "cosine_similarity", 1.1}, {"negative_readiness", "readiness_score", -1}, {"invalid_rank", "rank", 0}, {"null_skills", "matched_skills", nil}, {"blank_skill", "missing_skills", []string{" "}}, {"unknown_score_status", "score_status", "pending"}, {"ok_without_score", "score_status", "ok"}, {"unknown_item_field", "private", true}} {
		key, value := tc.key, tc.value
		cases = append(cases, struct {
			name   string
			mutate func(map[string]any)
		}{tc.name, func(r map[string]any) { r["items"].([]any)[0].(map[string]any)[key] = value }})
	}
	cases = append(cases, struct {
		name   string
		mutate func(map[string]any)
	}{"missing_nullable_key", func(r map[string]any) { delete(r["items"].([]any)[0].(map[string]any), "rank") }})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := mlTestResponse()
			tc.mutate(response)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(response) }))
			defer server.Close()
			result, err := mlTestService(server.URL).Recommend(context.Background(), mlTestInput())
			require.Nil(t, result)
			requireMLError(t, err, 502, "Invalid ML response")
		})
	}
	for _, raw := range []string{"{", `null`, `{} {}`, strings.Repeat("x", MLPayloadLimit+1)} {
		t.Run("invalid_body", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, raw) }))
			defer server.Close()
			_, err := mlTestService(server.URL).Recommend(context.Background(), mlTestInput())
			requireMLError(t, err, 502, "Invalid ML response")
		})
	}
	for _, status := range []string{"ok", "profil_tidak_lengkap"} {
		t.Run(status+"_empty_success", func(t *testing.T) {
			response := mlTestResponse()
			response["status"] = status
			response["items"] = []any{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _ = json.NewEncoder(w).Encode(response) }))
			defer server.Close()
			result, err := mlTestService(server.URL).Recommend(context.Background(), mlTestInput())
			require.NoError(t, err)
			require.NotNil(t, result.Items)
			require.Empty(t, result.Items)
		})
	}
}
