package mealsearch

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"log/slog"

	"github.com/google/uuid"
)

type CombinedSearcher struct {
	customMealService interfaces.CustomMealService
	prebuiltSearcher  interfaces.FoodSearcher
}

var _ interfaces.FoodSearcher = (*CombinedSearcher)(nil)

func NewCombinedSearcher(customMealService interfaces.CustomMealService, prebuiltSearcher interfaces.FoodSearcher) interfaces.FoodSearcher {
	return &CombinedSearcher{
		customMealService: customMealService,
		prebuiltSearcher:  prebuiltSearcher,
	}
}

func (s *CombinedSearcher) SearchFood(ctx context.Context, userID uuid.UUID, query string) (interfaces.FoodSearchResult, bool, error) {
	slog.Info("combined meal search started",
		"user_id", userID,
		"query", query,
	)

	customMeal, found, err := s.searchCustomMeals(ctx, userID, query)
	if err != nil {
		slog.Error("custom meal search failed",
			"user_id", userID,
			"query", query,
			"error", err,
		)
		return interfaces.FoodSearchResult{}, false, err
	}
	if found {
		slog.Info("combined meal search matched custom meal",
			"user_id", userID,
			"query", query,
			"matched_food", customMeal.Name,
		)
		return customMeal, true, nil
	}

	slog.Info("custom meal search missed; trying prebuilt dataset",
		"user_id", userID,
		"query", query,
	)

	prebuiltMeal, found, err := s.prebuiltSearcher.SearchFood(ctx, userID, query)
	if err != nil {
		slog.Error("prebuilt meal search failed",
			"user_id", userID,
			"query", query,
			"error", err,
		)
		return interfaces.FoodSearchResult{}, false, err
	}
	if found {
		slog.Info("combined meal search matched prebuilt meal",
			"user_id", userID,
			"query", query,
			"matched_food", prebuiltMeal.Name,
		)
		return prebuiltMeal, true, nil
	}

	slog.Info("combined meal search found no match",
		"user_id", userID,
		"query", query,
	)
	return interfaces.FoodSearchResult{}, false, nil
}

func (s *CombinedSearcher) searchCustomMeals(ctx context.Context, userID uuid.UUID, query string) (interfaces.FoodSearchResult, bool, error) {
	meals, err := s.customMealService.ListVisible(ctx, userID, query)
	if err != nil {
		return interfaces.FoodSearchResult{}, false, err
	}

	slog.Info("custom meal search completed",
		"user_id", userID,
		"query", query,
		"match_count", len(meals),
	)

	if len(meals) == 0 {
		return interfaces.FoodSearchResult{}, false, nil
	}

	meal := meals[0]

	return interfaces.FoodSearchResult{
		ID:       meal.ID,
		Name:     meal.Name,
		Source:   "custom",
		Tags:     meal.MealCategoryTags,
		Calories: meal.Calories,
		FatG:     meal.FatG,
		ProteinG: meal.ProteinG,
		CarbsG:   meal.CarbsG,
	}, true, nil
}
