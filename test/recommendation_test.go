package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"haji-koding/path-finder/internal/config"
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestRecommendationGenerate(t *testing.T) {
	viperConfig := config.NewViper()
	log := config.NewLogger()
	db := config.NewDatabase(viperConfig, log)
	validate := config.NewValidator()
	app := config.NewFiber(viperConfig)

	config.Bootstrap(&config.BootstrapConfig{
		DB:       db,
		App:      app,
		Log:      log,
		Validate: validate,
		Config:   viperConfig,
	})

	db.Exec("DELETE FROM recommendations")
	db.Exec("DELETE FROM stories")
	db.Exec("DELETE FROM users")

	createAndLogin := func(id, name, email, password, role string) string {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		user := &entity.User{
			ID:       id,
			Name:     name,
			Email:    email,
			Password: string(hashed),
			Role:     role,
		}
		require.NoError(t, db.Create(user).Error)

		loginBody := model.LoginUserRequest{Email: email, Password: password}
		loginBytes, _ := json.Marshal(loginBody)

		req := httptest.NewRequest(http.MethodPost, "/api/users/login", bytes.NewReader(loginBytes))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		require.NoError(t, err)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.LoginResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))

		return webResp.Data.Token
	}

	submitStory := func(token, content string) *model.StoryResponse {
		reqBody := model.CreateStoryRequest{Content: content}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/stories", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.StoryResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))

		return webResp.Data
	}

	postRecommendation := func(token string, body any) (*http.Response, model.WebResponse[*model.RecommendationResponse]) {
		var reader io.Reader
		if body != nil {
			bodyBytes, _ := json.Marshal(body)
			reader = bytes.NewReader(bodyBytes)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/recommendations", reader)
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		}

		resp, err := app.Test(req)
		require.NoError(t, err)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.RecommendationResponse]
		_ = json.Unmarshal(respBytes, &webResp)

		return resp, webResp
	}

	backendToken := createAndLogin("mhs-rec", "Rani Backend", "rani.rec@univ.ac.id", "pass123", entity.RoleMahasiswa)
	story := submitStory(backendToken, "Saya terbiasa membangun layanan backend dengan Go dan PostgreSQL. "+
		"Saya pernah membuat REST API memakai Fiber, mengelola versi kode dengan Git, dan menjalankan aplikasi lewat Docker. "+
		"Saya juga senang bekerja dalam tim dan mempelajari hal baru.")

	t.Run("Rekomendasi memakai cerita terakhir", func(t *testing.T) {
		resp, webResp := postRecommendation(backendToken, nil)

		require.Equal(t, http.StatusOK, resp.StatusCode)
		assert.True(t, webResp.Success)
		assert.Equal(t, "Rekomendasi berhasil dibuat", webResp.Message)
		require.NotNil(t, webResp.Data)

		data := webResp.Data
		assert.NotEmpty(t, data.ID)
		assert.Equal(t, "mhs-rec", data.StudentID)
		assert.Equal(t, story.ID, data.StoryID)
		assert.Equal(t, "1.0", data.SchemaVersion)
		assert.Contains(t, data.ModelVersion, "ner:")
		assert.Contains(t, data.ModelVersion, "encoder:")
		assert.Contains(t, data.ModelVersion, "ranker:")
		assert.Equal(t, 3, data.TopK)
		assert.Equal(t, entity.RecommendationStatusOK, data.Status)
		assert.Equal(t, entity.ProfileStatusLengkap, data.Profile.ProfileStatus)

		assert.Contains(t, data.Profile.HardSkills, "Go")
		assert.Contains(t, data.Profile.HardSkills, "PostgreSQL")

		require.NotEmpty(t, data.Items)
		assert.Len(t, data.Items, 3)

		top := data.Items[0]
		assert.Equal(t, "Backend Developer Intern", top.Category)
		assert.NotNil(t, top.MatchingScore)
		assert.NotNil(t, top.ReadinessScore)
		assert.NotNil(t, top.Rank)
		assert.Equal(t, 1, *top.Rank)
		assert.Equal(t, entity.ScoreStatusOK, top.ScoreStatus)
		assert.Contains(t, top.MatchedSkills, "Go")
		assert.NotEmpty(t, top.Company)
		assert.NotEmpty(t, top.Location)

		// Skor harus tidak naik setelah peringkat pertama.
		for index := 1; index < len(data.Items); index++ {
			previous := data.Items[index-1].MatchingScore
			current := data.Items[index].MatchingScore
			require.NotNil(t, previous)
			require.NotNil(t, current)
			assert.GreaterOrEqual(t, *previous, *current)
		}
	})

	t.Run("Cerita ditandai selesai dianalisis", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/stories/me", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", backendToken))

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.StoryResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))

		assert.Equal(t, entity.StoryAnalysisDone, webResp.Data.AnalysisStatus)
	})

	t.Run("top_k membatasi jumlah item", func(t *testing.T) {
		topK := 1
		resp, webResp := postRecommendation(backendToken, model.CreateRecommendationRequest{TopK: &topK})

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotNil(t, webResp.Data)
		assert.Len(t, webResp.Data.Items, 1)
	})

	t.Run("top_k di luar rentang ditolak", func(t *testing.T) {
		topK := 99
		resp, webResp := postRecommendation(backendToken, model.CreateRecommendationRequest{TopK: &topK})

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.False(t, webResp.Success)
	})

	t.Run("Rekomendasi terakhir bisa diambil", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/recommendations/me/latest", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", backendToken))

		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		respBytes, _ := io.ReadAll(resp.Body)
		var webResp model.WebResponse[*model.RecommendationResponse]
		require.NoError(t, json.Unmarshal(respBytes, &webResp))

		assert.True(t, webResp.Success)
		assert.Equal(t, "mhs-rec", webResp.Data.StudentID)
	})

	t.Run("Tanpa token ditolak", func(t *testing.T) {
		resp, _ := postRecommendation("", nil)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("Belum punya cerita", func(t *testing.T) {
		otherToken := createAndLogin("mhs-rec-empty", "Tanpa Cerita", "tanpa.cerita@univ.ac.id", "pass123", entity.RoleMahasiswa)

		resp, webResp := postRecommendation(otherToken, nil)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.False(t, webResp.Success)
	})

	t.Run("Cerita tanpa keahlian dikenali", func(t *testing.T) {
		noSkillToken := createAndLogin("mhs-rec-noskill", "Budi Berkebun", "budi.rec@univ.ac.id", "pass123", entity.RoleMahasiswa)
		submitStory(noSkillToken, "Setiap pagi saya berjalan kaki menyusuri sawah sambil menikmati udara dingin "+
			"dan kabut tipis yang turun dari lereng gunung di dekat rumah.")

		resp, webResp := postRecommendation(noSkillToken, nil)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NotNil(t, webResp.Data)

		assert.Equal(t, entity.RecommendationStatusProfileIncomplete, webResp.Data.Status)
		assert.Equal(t, entity.ProfileStatusTanpaSkill, webResp.Data.Profile.ProfileStatus)
		assert.Empty(t, webResp.Data.Items)
	})

	t.Run("Cerita milik orang lain tidak bisa dipakai", func(t *testing.T) {
		otherToken := createAndLogin("mhs-rec-other", "Orang Lain", "orang.lain@univ.ac.id", "pass123", entity.RoleMahasiswa)

		resp, _ := postRecommendation(otherToken, model.CreateRecommendationRequest{StoryID: story.ID})
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
