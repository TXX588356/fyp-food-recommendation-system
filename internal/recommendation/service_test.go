package recommendation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"fyp/food-rs/internal/foodapi"
	"fyp/food-rs/internal/llm"
)

type stubGenerator struct {
	meals llm.GeminiMealsResponse
	err   error
}

func (s stubGenerator) GenerateMeals(ctx context.Context) (llm.GeminiMealsResponse, error) {
	return s.meals, s.err
}

type stubSearcher struct {
	results map[string]foodapi.KaloriSearchResponse
	errors  map[string]error
}

func (s stubSearcher) SearchFood(ctx context.Context, mealName string) (foodapi.KaloriSearchResponse, bool, error) {
	if err := s.errors[mealName]; err != nil {
		return foodapi.KaloriSearchResponse{}, false, err
	}

	result, found := s.results[mealName]
	return result, found, nil
}

func TestRecommendSkipsMealsWithoutKaloriMatch(t *testing.T) {
	service := NewService(
		stubGenerator{
			meals: llm.GeminiMealsResponse{
				Meals: []llm.Meal{
					{Name: "Nasi Lemak"},
					{Name: "Unknown Meal"},
				},
			},
		},
		stubSearcher{
			results: map[string]foodapi.KaloriSearchResponse{
				"Nasi Lemak": {
					Success: true,
					Data:    []json.RawMessage{json.RawMessage(`{"name":"Nasi Lemak"}`)},
					Count:   1,
				},
			},
			errors: map[string]error{},
		},
	)

	got, err := service.Recommend(context.Background())
	if err != nil {
		t.Fatalf("Recommend returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].Meal.Name != "Nasi Lemak" {
		t.Fatalf("expected Nasi Lemak, got %q", got[0].Meal.Name)
	}
}

func TestRecommendIncludesSearchErrors(t *testing.T) {
	service := NewService(
		stubGenerator{
			meals: llm.GeminiMealsResponse{
				Meals: []llm.Meal{{Name: "Nasi Lemak"}},
			},
		},
		stubSearcher{
			results: map[string]foodapi.KaloriSearchResponse{},
			errors: map[string]error{
				"Nasi Lemak": errors.New("api failed"),
			},
		},
	)

	got, err := service.Recommend(context.Background())
	if err != nil {
		t.Fatalf("Recommend returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].Error != "api failed" {
		t.Fatalf("expected api failed error, got %q", got[0].Error)
	}
}
