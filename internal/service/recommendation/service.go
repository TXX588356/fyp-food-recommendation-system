package recommendation

import (
	"context"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"log/slog"
	"strings"

	"github.com/google/uuid"
)

type service struct {
	mealGenerator interfaces.MealGenerator
	foodSearcher  interfaces.FoodSearcher
}

func NewService(mealGenerator interfaces.MealGenerator, foodSearcher interfaces.FoodSearcher) interfaces.RecommendationService {
	return &service{
		mealGenerator: mealGenerator,
		foodSearcher:  foodSearcher,
	}
}

// GenerateCandidates requests meal suggestions from Gemini and enriches each
// cancidate with factual macro nutrition data from the food searcher
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
			slog.Info("generated meal skipped: no dataset/custom match",
				"user_id", userID,
				"meal_name", meal.Name,
			)
			continue
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
