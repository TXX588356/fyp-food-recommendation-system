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

type RecommendationResult struct {
	Candidates       []MatchedMealCandidate  `json:"candidates"`
	FilteredOut      []FilteredMealCandidate `json:"filtered_out"`
	FilteringApplied bool                    `json:"filtering_applied"`
}

type FilteredMealCandidate struct {
	Candidate MatchedMealCandidate `json:"candidate"`
	Reason    string               `json:"reason"`
}

type RecommendationService interface {
	GenerateRecommendationResult(ctx context.Context, userID uuid.UUID, input MealPromptInput) (RecommendationResult, error)
}
