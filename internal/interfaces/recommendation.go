package interfaces

import "context"

type MatchedMealCandidate struct {
	GeneratedMeal GeneratedMeal    `json:"generated_meal"`
	Food          FoodSearchResult `json:"food"`
	MatchedQuery  string           `json:"matched_query"`
}

type RecommendationService interface {
	GenerateCandidates(ctx context.Context, input MealPromptInput) ([]MatchedMealCandidate, error)
}
