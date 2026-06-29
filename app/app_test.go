package app

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"testing"
)

type stubMealGenerator struct{}

func (stubMealGenerator) GenerateMeals(ctx context.Context, input interfaces.MealPromptInput) (interfaces.GeminiMealsResponse, error) {
	return interfaces.GeminiMealsResponse{}, nil
}

func (stubMealGenerator) AutocompleteCustomMeal(ctx context.Context, input interfaces.CustomMealAutocompleteInput) (interfaces.CustomMealAutocompleteResponse, error) {
	return interfaces.CustomMealAutocompleteResponse{}, nil
}

func TestGetRecommendationServiceUsesCachedCatalogSearcher(t *testing.T) {
	originalMealGeneratorFactory := newMealGenerator
	t.Cleanup(func() {
		newMealGenerator = originalMealGeneratorFactory
	})

	var gotGeminiAPIKey string

	newMealGenerator = func(ctx context.Context, apiKey string) (AIClient, error) {
		gotGeminiAPIKey = apiKey
		return stubMealGenerator{}, nil
	}
	a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images")

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

	first, err := a.GetCatalogFoodSearcher(context.Background())
	if err != nil {
		t.Fatalf("expected catalogue searcher, got error: %v", err)
	}
	second, err := a.GetCatalogFoodSearcher(context.Background())
	if err != nil {
		t.Fatalf("expected cached catalogue searcher, got error: %v", err)
	}
	if first != second {
		t.Fatal("expected catalogue searcher to be cached")
	}
}

func TestGetCustomMealAutocompleterUsesCachedAIClient(t *testing.T) {
	originalMealGeneratorFactory := newMealGenerator
	t.Cleanup(func() {
		newMealGenerator = originalMealGeneratorFactory
	})

	factoryCalls := 0

	newMealGenerator = func(ctx context.Context, apiKey string) (AIClient, error) {
		factoryCalls++
		return stubMealGenerator{}, nil
	}

	a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images")

	first, err := a.GetCustomMealAutocompleter(context.Background())
	if err != nil {
		t.Fatalf("expected custom meal autocompleter, got error: %v", err)
	}

	second, err := a.GetCustomMealAutocompleter(context.Background())
	if err != nil {
		t.Fatalf("expected cached custom meal autocompleter, got error: %v", err)
	}

	if first != second {
		t.Fatal("expected custom meal autocompleter to be cached")
	}

	if factoryCalls != 1 {
		t.Fatalf("expected AI client factory to be called once, got %d", factoryCalls)
	}
}
