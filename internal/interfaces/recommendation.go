package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type MatchedMealCandidate struct {
	GeneratedMeal GeneratedMeal    `json:"generated_meal"`
	Food          FoodSearchResult `json:"food"`
	MatchedQuery  string           `json:"matched_query"`
}

type RecommendationService interface {
	GenerateCandidates(ctx context.Context, userID uuid.UUID, input MealPromptInput) ([]MatchedMealCandidate, error)
}
