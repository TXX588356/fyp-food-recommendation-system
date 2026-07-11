package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type MatchedMealCandidate struct {
	GeneratedMeal GeneratedMeal    `json:"generated_meal"` // Meal produced by Gemini
	Food          FoodSearchResult `json:"food"`           // Matched food data from catalog / custom meal
	MatchedQuery  string           `json:"matched_query"`  // Records of successful matched with generated meals
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

// MealMatchTask is one ambiguous generated meal plus the candidates that the backend allows Gemini to choose from.
// MealIndex is the original generated meal index
// and must be preserved through the whole matching flow.
type MealMatchTask struct {
	MealIndex     int                  `json:"meal_index"`
	GeneratedMeal GeneratedMeal        `json:"generated_meal"`
	Candidates    []FoodMatchCandidate `json:"candidates"`
}

// MealMatchDecision is Gemini's answer for one MealMatchTask
// Candidate ID is required only when Decision is MATCH; NO_MATCH must leave it empty.
type MealMatchDecision struct {
	MealIndex   int    `json:"meal_index"`
	Decision    string `json:"decision"`
	CandidateID string `json:"candidate_id,omitempty"`
}

type RecommendationService interface {
	GenerateRecommendationResult(ctx context.Context, userID uuid.UUID, input MealPromptInput) (RecommendationResult, error)
}

// MealMatchAdjudicator resolves ambiguous recommendation matches in one batch.
// It is used after backend candidate retrieval when more than one plausible match exists
// or when only fuzzy / partial matches are available.
type MealMatchAdjudicator interface {
	ResolveMatches(ctx context.Context, tasks []MealMatchTask) ([]MealMatchDecision, error)
}
