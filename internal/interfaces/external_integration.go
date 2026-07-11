package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type MealPromptInput struct {
	Goal                string
	DietaryRestrictions []string
	HealthConcerns      []string
	PreferredMealTags   []string
	MealCategory        string

	MonthlyMealBudget float64
	CurrentMonthSpent float64
	RemainingBudget   float64
	PerMealBudget     float64

	PriceMarketLocation     string
	PriceMarketLocationType string // "weekday" or "weekend"
	History                 MealHistoryContext
}

type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type GeneratedMeal struct {
	// Food API search
	Name string `json:"name"`

	// Fallback exact-search terms if Name does not match the food API
	AlternativeSearchTerms []string   `json:"alternative_search_terms"`
	EstimatedPriceRange    PriceRange `json:"estimated_price_range"`

	// Gemini estimation
	SodiumLevel string `json:"sodium_level"`
	SugarLevel  string `json:"sugar_level"`
	PurineRisk  string `json:"purine_risk"`

	// Include only conditions supplied in MealPromptInput.HealthConcerns
	// Values: SAFE, CAUTION, AVOID
	HealthFlags map[string]string `json:"health_flags"`
}

type GeminiMealsResponse struct {
	Meals []GeneratedMeal `json:"meals"`
}

type FoodSearchResult struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Tags     []string `json:"tags"`
	Calories float64  `json:"calories"`
	FatG     float64  `json:"fat_g"`
	ProteinG float64  `json:"protein_g"`
	CarbsG   float64  `json:"carbs_g"`
	ImageURL string   `json:"image_url,omitempty"`
}

// FoodMatchKind explains why a food candidate matched generated meal terms.
// Recommendation matching uses this to accept one exact match locally and send
// fuzzy or ambiguous matches to adjudication.
type FoodMatchKind string

const (
	FoodMatchExactName  FoodMatchKind = "exact_name"
	FoodMatchExactAlias FoodMatchKind = "exact_alias"
	FoodMatchFuzzy      FoodMatchKind = "fuzzy"
	FoodMatchPartial    FoodMatchKind = "partial"
)

type FoodMatchCandidate struct {
	Food        FoodSearchResult `json:"food"`
	MatchKind   FoodMatchKind    `json:"matched_kind"`
	MatchedTerm string           `json:"matched_term"`
	Score       float64          `json:"score"`
}

// MealHistoryContext shows the summary of user's meal log history
type MealHistoryContext struct {
	RecentMealNames      []string       // List of recently eaten meal names
	RecentCategoryCounts map[string]int // Appearance count for each meal category
	RepeatedMealNames    []string       // Names that appears more than once in recent history
	RecentlyEatenByName  map[string]int // Maps meal name to how many days ago it was last eaten (for scoring, ranking)
}

type MealGenerator interface {
	GenerateMeals(ctx context.Context, input MealPromptInput) (GeminiMealsResponse, error)
}

// FoodSearcher returns the single best food match for a query.
// Use this for manual search and existing simple lookup flows where callers
// need one accepted result or a not-found response.
type FoodSearcher interface {
	SearchFood(ctx context.Context, userID uuid.UUID, query string) (FoodSearchResult, bool, error)
}

// FoodCandidateSearcher returns ranked match candidates for generated meal terms.
// Use this only for recommendation matching, where ambiguous results must be
// compared, adjudicated, and validated instead of accepting the first match.
type FoodCandidateSearcher interface {
	SearchFoodCandidates(ctx context.Context, userID uuid.UUID, queries []string, limit int) ([]FoodMatchCandidate, error)
}
