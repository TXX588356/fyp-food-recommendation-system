package app

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"testing"

	"github.com/google/uuid"
)

type stubMealGenerator struct{}

func (stubMealGenerator) GenerateMeals(ctx context.Context, input interfaces.MealPromptInput) (interfaces.GeminiMealsResponse, error) {
	return interfaces.GeminiMealsResponse{}, nil
}

type stubFoodSearcher struct{}

func (stubFoodSearcher) SearchFood(ctx context.Context, userID uuid.UUID, query string) (interfaces.FoodSearchResult, bool, error) {
	return interfaces.FoodSearchResult{}, false, nil
}

func TestGetRecommendationServiceUsesPrebuiltMealDataset(t *testing.T) {
	originalMealGeneratorFactory := newMealGenerator
	originalFoodSearcherFactory := newFoodSearcher
	t.Cleanup(func() {
		newMealGenerator = originalMealGeneratorFactory
		newFoodSearcher = originalFoodSearcherFactory
	})

	var gotGeminiAPIKey string
	var gotDatasetPath string

	newMealGenerator = func(ctx context.Context, apiKey string) (interfaces.MealGenerator, error) {
		gotGeminiAPIKey = apiKey
		return stubMealGenerator{}, nil
	}
	newFoodSearcher = func(datasetPath string) (interfaces.FoodSearcher, error) {
		gotDatasetPath = datasetPath
		return stubFoodSearcher{}, nil
	}

	a := New(nil, "jwt-secret", "gemini-key")

	service, err := a.GetRecommendationService(context.Background())
	if err != nil {
		t.Fatalf("expected recommendation service, got error: %v", err)
	}
	if service == nil {
		t.Fatal("expected recommendation service")
	}
	if gotGeminiAPIKey != "gemini-key" {
		t.Fatalf("expected Gemini API key to be forwarded")
	}
	if gotDatasetPath != "internal/data/prebuilt_meals.json" {
		t.Fatalf("expected prebuilt meal dataset path to be forwarded")
	}
}
