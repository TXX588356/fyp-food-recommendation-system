package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type MatchedMealCandidate struct {
	GeneratedMeal  GeneratedMeal           `json:"generated_meal"` // Meal produced by Gemini
	Food           FoodSearchResult        `json:"food"`           // Matched food data from catalog / custom meal
	MatchedQuery   string                  `json:"matched_query"`  // Records of successful matched with generated meals
	Score          float64                 `json:"score"`
	ScoreBreakdown CandidateScoreBreakdown `json:"score_breakdown"`
}

type CandidateScoreBreakdown struct {
	GoalAlignment  float64 `json:"goal_alignment"`
	BudgetFit      float64 `json:"budget_fit"`
	RecencyPenalty float64 `json:"recency_penalty"`
	Preference     float64 `json:"preference"`
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

// RecommendationRequestInput contains only request-level values supplied by the
// client. The recommendation service enriches it with saved preferences and
// meal history before calling the LLM.
type RecommendationRequestInput struct {
	MealCategory      string
	CurrentMonthSpent float64
	PerMealBudget     float64
	Location          string
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

// MealDetailInput is the service input for building a selected recommendation's detail.
type MealDetailInput struct {
	MealCategory string
	Candidate    MatchedMealCandidate
	Location     string
}

// MealDetailResult is the service result returned to the meal detail endpoint
type MealDetailResult struct {
	Meal                      MealDetailMeal
	RecommendationExplanation string
	Location                  MealDetailLocation
	Restaurants               []RestaurantResult
	RestaurantLookupStatus    string
}

// MealDetailMeal is the display-ready selected meal detail.
type MealDetailMeal struct {
	ID                  string
	Name                string
	MealCategory        string
	ImageURL            string
	ServingDescription  string
	EstimatedPriceRange PriceRange
	Nutrition           MealDetailNutrition
	Signals             MealDetailSignals
}

// MealDetailNutrition contains serving-based macro nutrition for the selected meal.
type MealDetailNutrition struct {
	Calories float64
	FatG     float64
	ProteinG float64
	CarbsG   float64
}

// MealDetailSignals contains generated health/risk signals for the selected meal.
type MealDetailSignals struct {
	SodiumLevel string
	SugarLevel  string
	PurineRisk  string
	HealthFlags map[string]string
}

// MealDetailLocation describes the preference-based location used for restaurant lookup.
type MealDetailLocation struct {
	Query string
	Basis string
}

type RecommendationService interface {
	GenerateRecommendationResult(ctx context.Context, userID uuid.UUID, input RecommendationRequestInput) (RecommendationResult, error)
	BuildMealDetail(ctx context.Context, userID uuid.UUID, input MealDetailInput) (MealDetailResult, error)
}

// MealMatchAdjudicator resolves ambiguous recommendation matches in one batch.
// It is used after backend candidate retrieval when more than one plausible match exists
// or when only fuzzy / partial matches are available.
type MealMatchAdjudicator interface {
	ResolveMatches(ctx context.Context, tasks []MealMatchTask) ([]MealMatchDecision, error)
}
