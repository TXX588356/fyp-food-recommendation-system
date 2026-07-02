package recommendation

import (
	"context"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

type service struct {
	mealGenerator           interfaces.MealGenerator
	foodSearcher            interfaces.FoodSearcher
	catalogService          interfaces.CatalogService
	customMealAutocompleter interfaces.CustomMealAutocompleter
	mealLogRepository       interfaces.MealLogRepository
	now                     func() time.Time
}

func NewService(mealGenerator interfaces.MealGenerator, foodSearcher interfaces.FoodSearcher, catalogService interfaces.CatalogService, customMealAutocompleter interfaces.CustomMealAutocompleter, mealLogRepository interfaces.MealLogRepository) interfaces.RecommendationService {
	return &service{
		mealGenerator:           mealGenerator,
		foodSearcher:            foodSearcher,
		catalogService:          catalogService,
		customMealAutocompleter: customMealAutocompleter,
		mealLogRepository:       mealLogRepository,
		now:                     time.Now,
	}
}

// GenerateCandidates requests meal suggestions from Gemini and enriches each
// candidate with factual macro nutrition data from the food searcher
func (s *service) GenerateCandidates(ctx context.Context, userID uuid.UUID, input interfaces.MealPromptInput) ([]interfaces.MatchedMealCandidate, error) {
	slog.Info("recommendation generation started",
		"user_id", userID,
		"meal_category", input.MealCategory,
		"goal", input.Goal,
	)

	slog.Info("calling Gemini meal generator",
		"user_id", userID,
		"meal_category", input.MealCategory,
	)

	// Load history
	historyStart := s.now().AddDate(0, 0, -30)
	historyLogs, err := s.mealLogRepository.ListByUserAndRange(ctx, userID, historyStart, s.now().AddDate(0, 0, 1))
	if err != nil {
		return nil, fmt.Errorf("load recommendation history: %w", err)
	}
	input.History = buildMealHistoryContext(historyLogs, s.now())

	response, err := s.mealGenerator.GenerateMeals(ctx, input)
	if err != nil {
		slog.Error("Gemini meal generation failed",
			"user_id", userID,
			"meal_category", input.MealCategory,
			"error", err,
		)
		return nil, fmt.Errorf("generated meal candidates: %w", err)
	}

	slog.Info("Gemini meal generation completed",
		"user_id", userID,
		"meal_category", input.MealCategory,
		"generated_count", len(response.Meals),
	)

	candidates := make([]interfaces.MatchedMealCandidate, 0, len(response.Meals))

	for index, meal := range response.Meals {
		slog.Info("matching generated meal",
			"user_id", userID,
			"index", index,
			"meal_name", meal.Name,
			"alternative_terms_count", len(meal.AlternativeSearchTerms),
		)

		candidate, found, err := s.matchFood(ctx, userID, meal)
		if err != nil {
			slog.Error("generated meal matching failed",
				"user_id", userID,
				"meal_name", meal.Name,
				"error", err,
			)
			return nil, err
		}

		if !found {
			slog.Info("generated meal unmatched; attempting prebuilt catalog auto-create",
				"user_id", userID,
				"meal_name", meal.Name,
			)

			candidate, err = s.createMissingGeneratedCatalogMeal(ctx, userID, meal)
			if err != nil {
				slog.Error("generated meal catalog auto-create failed",
					"user_id", userID,
					"meal_name", meal.Name,
					"error", err)
				continue
			}
		}

		slog.Info("generated meal matched",
			"user_id", userID,
			"meal_name", meal.Name,
			"matched_query", candidate.MatchedQuery,
			"matched_food", candidate.Food.Name,
		)
		candidates = append(candidates, candidate)
	}

	slog.Info("recommendation generation completed",
		"user_id", userID,
		"meal_category", input.MealCategory,
		"candidate_count", len(candidates),
	)

	return candidates, nil
}

func (s *service) createMissingGeneratedCatalogMeal(ctx context.Context, userID uuid.UUID, meal interfaces.GeneratedMeal) (interfaces.MatchedMealCandidate, error) {
	details, err := s.customMealAutocompleter.AutocompleteCustomMeal(ctx, interfaces.CustomMealAutocompleteInput{
		Name: meal.Name,
	})

	if err != nil {
		return interfaces.MatchedMealCandidate{}, fmt.Errorf("autocomplete generated catalog meal %q: %w", meal.Name, err)
	}

	catalogMeal, err := s.catalogService.CreateGeneratedMeal(ctx, interfaces.GeneratedCatalogMealInput{
		Name:               meal.Name,
		CategoryCodes:      details.MealCategoryTags,
		ServingDescription: "1 serving",
		Calories:           details.Calories,
		ProteinG:           details.ProteinG,
		CarbsG:             details.CarbsG,
		FatG:               details.FatG,
	})

	if err != nil {
		return interfaces.MatchedMealCandidate{}, fmt.Errorf("create generated catalog meal %q: %w", meal.Name, err)
	}

	return interfaces.MatchedMealCandidate{
		GeneratedMeal: meal,
		MatchedQuery:  meal.Name,
		Food:          catalogMealToFoodSearchResult(catalogMeal),
	}, nil
}

// matchFood tries the normalized Gemini name first, followed by each fallback search term.
// A meal is discarded if every exact lookup returns no match.
func (s *service) matchFood(ctx context.Context, userID uuid.UUID, meal interfaces.GeneratedMeal) (interfaces.MatchedMealCandidate, bool, error) {
	searchTerms := buildSearchTerms(meal)

	for _, query := range searchTerms {
		slog.Info("searching meal candidate",
			"user_id", userID,
			"generated_meal", meal.Name,
			"query", query,
		)

		food, found, err := s.foodSearcher.SearchFood(ctx, userID, query)
		if err != nil {
			return interfaces.MatchedMealCandidate{}, false, fmt.Errorf("search food %q: %w", query, err)
		}

		if found {
			return interfaces.MatchedMealCandidate{
				GeneratedMeal: meal,
				Food:          food,
				MatchedQuery:  query,
			}, true, nil
		}
	}

	return interfaces.MatchedMealCandidate{}, false, nil
}

func catalogMealToFoodSearchResult(meal interfaces.CatalogMeal) interfaces.FoodSearchResult {
	result := interfaces.FoodSearchResult{
		ID:   meal.ID.String(),
		Name: meal.Name,
		Tags: meal.Categories,
	}
	if meal.SelectedNutrition.Calories != nil {
		result.Calories = *meal.SelectedNutrition.Calories
	}
	if meal.SelectedNutrition.FatG != nil {
		result.FatG = *meal.SelectedNutrition.FatG
	}
	if meal.SelectedNutrition.ProteinG != nil {
		result.ProteinG = *meal.SelectedNutrition.ProteinG
	}
	if meal.SelectedNutrition.CarbsG != nil {
		result.CarbsG = *meal.SelectedNutrition.CarbsG
	}
	if meal.Image != nil {
		result.ImageURL = meal.Image.URL
	}

	return result
}

// buildSearchTerms returns unique non-empty lookup terms in priority order.
func buildSearchTerms(meal interfaces.GeneratedMeal) []string {
	searchTerms := make([]string, 0, 1+len(meal.AlternativeSearchTerms))

	seen := make(map[string]bool)

	addTerm := func(term string) {
		trimmed := strings.TrimSpace(term)
		normalized := strings.ToLower(trimmed)

		if trimmed == "" || seen[normalized] {
			return
		}

		seen[normalized] = true
		searchTerms = append(searchTerms, trimmed)
	}

	addTerm(meal.Name)

	for _, term := range meal.AlternativeSearchTerms {
		addTerm(term)
	}

	return searchTerms
}
