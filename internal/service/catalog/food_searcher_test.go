package catalog

import (
	"context"
	"testing"

	"fyp/food-rs/types/model"
	"github.com/google/uuid"
)

func TestFoodSearcherUsesDefaultPortionAndCompleteMacros(t *testing.T) {
	id := uuid.New()
	repo := &testRepository{meals: []model.PrebuiltMeal{{
		ID: id, Name: "Soup", CategoryCodes: []string{"soups"}, ServingDescription: "bowl", ServingGramWeight: 300,
		Calories: float(240), ProteinG: float(12), CarbsG: float(30), FatG: float(6),
	}}}
	searcher := NewFoodSearcher(NewService(repo, testResolver{}))

	got, found, err := searcher.SearchFood(context.Background(), uuid.New(), "soup")
	if err != nil || !found {
		t.Fatalf("expected match, found=%v err=%v", found, err)
	}
	if got.Calories != 240 || got.ProteinG != 12 {
		t.Fatalf("unexpected nutrition: %#v", got)
	}
}
