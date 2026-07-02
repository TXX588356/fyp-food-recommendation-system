package recommendation

import (
	"fyp/food-rs/types/model"
	"testing"
	"time"
)

func TestBuildMealHistoryContextDetectsRecentRepeatsAndCategoryCounts(t *testing.T) {
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	logs := []model.MealLog{
		{MealName: "Nasi Lemak", MealCategory: []string{"rice_dishes", "malaysian"}, EatenAt: time.Date(2026, time.July, 14, 8, 0, 0, 0, time.UTC)},
		{MealName: "Nasi Lemak", MealCategory: []string{"rice_dishes", "malaysian"}, EatenAt: time.Date(2026, time.July, 12, 8, 0, 0, 0, time.UTC)},
		{MealName: "Chicken Soup", MealCategory: []string{"soups"}, EatenAt: time.Date(2026, time.July, 5, 22, 0, 0, 0, time.UTC)},
	}

	got := buildMealHistoryContext(logs, now)

	if got.RecentMealNames[0] != "Nasi Lemak" {
		t.Fatalf("expected most recent meal first, got %#v", got.RecentMealNames)
	}

	if len(got.RepeatedMealNames) != 1 || got.RepeatedMealNames[0] != "Nasi Lemak" {
		t.Fatalf("expected repeated Nasi Lemak, got %#v", got.RepeatedMealNames)
	}

	if got.RecentCategoryCounts["rice_dishes"] != 2 {
		t.Fatalf("expected rice_dishes count 2, got %#v", got.RecentCategoryCounts)
	}
}
