package converter

import (
	"time"

	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
)

// RecommendationToResponse mengubah entitas tersimpan menjadi bentuk response API.
func RecommendationToResponse(recommendation *entity.Recommendation) *model.RecommendationResponse {
	items := make([]model.RecommendationItemResponse, len(recommendation.Items))
	for index := range recommendation.Items {
		items[index] = recommendationItemToResponse(recommendation.Items[index])
	}

	return &model.RecommendationResponse{
		ID:            recommendation.ID,
		StudentID:     recommendation.UserID,
		StoryID:       recommendation.StoryID,
		GeneratedAt:   recommendation.GeneratedAt.UTC().Format(time.RFC3339),
		SchemaVersion: recommendation.SchemaVersion,
		ModelVersion:  recommendation.ModelVersion,
		TopK:          recommendation.TopK,
		Status:        recommendation.Status,
		Profile: model.RecommendationProfileResponse{
			HardSkills:    nonNilStrings(recommendation.HardSkills),
			SoftSkills:    nonNilStrings(recommendation.SoftSkills),
			Interests:     nonNilStrings(recommendation.Interests),
			ProfileStatus: recommendation.ProfileStatus,
		},
		Items: items,
	}
}

func recommendationItemToResponse(item entity.RecommendationItem) model.RecommendationItemResponse {
	return model.RecommendationItemResponse{
		JobID:            item.JobID,
		Title:            item.Title,
		Category:         item.Category,
		Company:          item.Company,
		Location:         item.Location,
		WorkType:         item.WorkType,
		Source:           item.Source,
		SourceURL:        item.SourceURL,
		MatchingScore:    item.MatchingScore,
		CosineSimilarity: item.CosineSimilarity,
		ReadinessScore:   item.ReadinessScore,
		MatchedSkills:    nonNilStrings(item.MatchedSkills),
		MissingSkills:    nonNilStrings(item.MissingSkills),
		ScoreStatus:      item.ScoreStatus,
		Rank:             item.Rank,
	}
}

// nonNilStrings memastikan JSON berisi [] dan bukan null.
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
