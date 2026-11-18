package converter

import (
	"haji-koding/path-finder/internal/entity"
	"haji-koding/path-finder/internal/model"
)

func StoryToResponse(story *entity.Story) *model.StoryResponse {
	return &model.StoryResponse{
		ID:             story.ID,
		UserID:         story.UserID,
		Content:        story.Content,
		WordCount:      story.WordCount,
		AnalysisStatus: story.AnalysisStatus,
		CreatedAt:      story.CreatedAt,
		UpdatedAt:      story.UpdatedAt,
	}
}

func StoryToResponses(stories []entity.Story) []model.StoryResponse {
	responses := make([]model.StoryResponse, len(stories))
	for i := range stories {
		responses[i] = *StoryToResponse(&stories[i])
	}
	return responses
}
