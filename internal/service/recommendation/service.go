package recommendation

import (
	"context"

	"fyp/food-rs/internal/service/foodapi"
	"fyp/food-rs/internal/service/llm"
)

type MealGenerator interface {
	GenerateMeals(ctx context.Context) (llm.GeminiMealsResponse, error)
}

type FoodSearcher interface {
	SearchFood(ctx context.Context, mealName string) (foodapi.KaloriSearchResponse, bool, error)
}

type Service struct {
	generator MealGenerator
	searcher  FoodSearcher
}

type MealWithKaloriResult struct {
	Meal         llm.Meal                     `json:"meal"`
	KaloriResult foodapi.KaloriSearchResponse `json:"kalori_result,omitempty"`
	Error        string                       `json:"error,omitempty"`
}

func NewService(generator MealGenerator, searcher FoodSearcher) Service {
	return Service{
		generator: generator,
		searcher:  searcher,
	}
}

func (s Service) Recommend(ctx context.Context) ([]MealWithKaloriResult, error) {
	meals, err := s.generator.GenerateMeals(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]MealWithKaloriResult, 0, len(meals.Meals))
	for _, meal := range meals.Meals {
		kaloriResult, found, err := s.searcher.SearchFood(ctx, meal.Name)
		item := MealWithKaloriResult{Meal: meal}
		if err != nil {
			item.Error = err.Error()
		} else if !found {
			continue
		} else {
			item.KaloriResult = kaloriResult
		}

		results = append(results, item)
	}

	return results, nil
}
