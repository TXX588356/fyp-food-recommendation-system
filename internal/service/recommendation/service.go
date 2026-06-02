package recommendation

import (
	"context"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"strings"
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
func (s *service) GenerateCandidates(ctx context.Context, input interfaces.MealPromptInput) ([]interfaces.MatchedMealCandidate, error) {
	response, err := s.mealGenerator.GenerateMeals(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("generated meal candidates: %w", err)
	}

	candidates := make([]interfaces.MatchedMealCandidate, 0, len(response.Meals))

	for _, meal := range response.Meals {
		candidate, found, err := s.matchFood(ctx, meal)
		if err != nil {
			return nil, err
		}

		if !found {
			continue
		}

		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

// matchFood tries the normalized Gemini name first, followed by each fallback search term.
// A meal is discarded if every exact lookup returns no match.
func (s *service) matchFood(ctx context.Context, meal interfaces.GeneratedMeal) (interfaces.MatchedMealCandidate, bool, error) {
	searchTerms := buildSearchTerms(meal)

	for _, query := range searchTerms {
		food, found, err := s.foodSearcher.SearchFood(ctx, query)
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
