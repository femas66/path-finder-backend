package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"haji-koding/path-finder/internal/model"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const MLPayloadLimit = 1 << 20

type MLService struct {
	client   *http.Client
	endpoint string
	token    string
	log      *logrus.Logger
}

// Invalid or absent ML configuration disables inference, not the rest of the API.
func NewMLService(config *viper.Viper, logger *logrus.Logger) *MLService {
	s := &MLService{log: logger}
	raw := config.GetString("ml.service_url")
	token := config.GetString("ml.service_token")
	seconds := 60
	if config.IsSet("ml.request_timeout_seconds") {
		value, err := strconv.Atoi(config.GetString("ml.request_timeout_seconds"))
		if err != nil || value < 1 || value > 120 {
			return s
		}
		seconds = value
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return s
	}
	if token == "" || strings.TrimSpace(token) != token {
		return s
	}
	for _, c := range token {
		if c < 33 || c > 126 {
			return s
		}
	}
	s.endpoint = strings.TrimRight(raw, "/") + "/recommend"
	s.token = token
	s.client = &http.Client{Timeout: time.Duration(seconds) * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	return s
}

func (s *MLService) Recommend(ctx context.Context, input *model.MLRecommendationRequest) (*model.RecommendationResponse, error) {
	if s == nil || s.client == nil {
		return nil, fiber.NewError(503, "ML service unavailable")
	}
	body, err := json.Marshal(input)
	if err != nil {
		return nil, fiber.NewError(422, "Invalid recommendation input")
	}
	if len(body) > MLPayloadLimit {
		return nil, fiber.NewError(413, "Recommendation payload exceeds 1 MiB")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fiber.NewError(503, "ML service unavailable")
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	res, err := s.client.Do(req)
	if err != nil {
		var timeout interface{ Timeout() bool }
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &timeout) && timeout.Timeout()) {
			return nil, fiber.NewError(504, "ML service timed out")
		}
		return nil, fiber.NewError(503, "ML service unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		if s.log != nil {
			s.log.WithField("status", res.StatusCode).Warn("ML request failed")
		}
		switch res.StatusCode {
		case 400:
			return nil, fiber.NewError(422, "Invalid recommendation input")
		case 413:
			return nil, fiber.NewError(413, "Recommendation payload exceeds 1 MiB")
		// Busy and internal authentication failures both expose only service unavailability.
		default:
			return nil, fiber.NewError(503, "ML service unavailable")
		}
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, MLPayloadLimit+1))
	if err != nil || len(raw) > MLPayloadLimit {
		return nil, fiber.NewError(502, "Invalid ML response")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return nil, fiber.NewError(502, "Invalid ML response")
	}
	for _, key := range []string{"student_id", "generated_at", "schema_version", "model_version", "top_k", "status", "items"} {
		if value, ok := fields[key]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fiber.NewError(502, "Invalid ML response")
		}
	}
	var items []map[string]json.RawMessage
	if json.Unmarshal(fields["items"], &items) != nil {
		return nil, fiber.NewError(502, "Invalid ML response")
	}
	for _, item := range items {
		for _, key := range []string{"job_id", "title", "matching_score", "cosine_similarity", "readiness_score", "matched_skills", "missing_skills", "score_status", "rank"} {
			if _, ok := item[key]; !ok {
				return nil, fiber.NewError(502, "Invalid ML response")
			}
		}
	}
	var result model.RecommendationResponse
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&result) != nil || decoder.Decode(new(any)) != io.EOF || !validRecommendation(&result, input) {
		return nil, fiber.NewError(502, "Invalid ML response")
	}
	return &result, nil
}

func validRecommendation(r *model.RecommendationResponse, input *model.MLRecommendationRequest) bool {
	if r.StudentID != input.Candidate.StudentID || r.TopK != 3 || r.Items == nil || len(r.Items) > 3 || r.SchemaVersion != "1.0" {
		return false
	}
	if _, err := time.Parse(time.RFC3339, r.GeneratedAt); err != nil {
		return false
	}
	for _, segment := range []string{"ner:", "encoder:", "ranker:"} {
		found := false
		for _, part := range strings.Split(r.ModelVersion, ";") {
			if strings.HasPrefix(part, segment) && strings.TrimSpace(strings.TrimPrefix(part, segment)) != "" {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	if r.Status != "ok" && r.Status != "parsial" && r.Status != "profil_tidak_lengkap" {
		return false
	}
	if r.Status == "profil_tidak_lengkap" && len(r.Items) != 0 {
		return false
	}
	jobs := make(map[string]bool, len(input.Jobs))
	for _, job := range input.Jobs {
		jobs[job.JobID] = true
	}
	seen := make(map[string]bool)
	for _, item := range r.Items {
		if !jobs[item.JobID] || seen[item.JobID] || strings.TrimSpace(item.Title) == "" || item.MatchedSkills == nil || item.MissingSkills == nil {
			return false
		}
		seen[item.JobID] = true
		for _, list := range [][]string{item.MatchedSkills, item.MissingSkills} {
			for _, text := range list {
				if strings.TrimSpace(text) == "" {
					return false
				}
			}
		}
		for _, score := range []*int{item.MatchingScore, item.ReadinessScore} {
			if score != nil && (*score < 0 || *score > 100) {
				return false
			}
		}
		if item.CosineSimilarity != nil && (*item.CosineSimilarity < 0 || *item.CosineSimilarity > 1) {
			return false
		}
		if item.Rank != nil && *item.Rank < 1 {
			return false
		}
		switch item.ScoreStatus {
		case "ok":
			if item.MatchingScore == nil {
				return false
			}
		case "skor_belum_tersedia", "persyaratan_belum_tersedia", "profil_tanpa_keterampilan":
			if item.MatchingScore != nil {
				return false
			}
		default:
			return false
		}
		if item.ScoreStatus == "persyaratan_belum_tersedia" && len(item.MissingSkills) != 0 {
			return false
		}
		if r.Status == "ok" && item.ScoreStatus != "ok" {
			return false
		}
	}
	return true
}
