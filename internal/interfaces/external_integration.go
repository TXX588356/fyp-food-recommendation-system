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

type MealHistoryContext struct {
	RecentMealNames      []string
	RecentCategoryCounts map[string]int
	RepeatedMealNames    []string
	RecentlyEatenByName  map[string]int
}

type MealGenerator interface {
	GenerateMeals(ctx context.Context, input MealPromptInput) (GeminiMealsResponse, error)
}

type FoodSearcher interface {
	SearchFood(ctx context.Context, userID uuid.UUID, query string) (FoodSearchResult, bool, error)
}
