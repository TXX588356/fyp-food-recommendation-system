package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type MatchedMealCandidate struct {
	GeneratedMeal GeneratedMeal    `json:"generated_meal"` // Meal produced by Gemini
	Food          FoodSearchResult `json:"food"`           // Matched food data from catalog / custom meal
	MatchedQuery  string           `json:"matched_query"`  // Records of successfull matched with generated meals
}

type RecommendationService interface {
	GenerateCandidates(ctx context.Context, userID uuid.UUID, input MealPromptInput) ([]MatchedMealCandidate, error)
}
