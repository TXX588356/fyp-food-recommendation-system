package features

import (
	"testing"
)

func TestInitialFeatureSchemaOrder(t *testing.T) {
	got := FeatureNames()
	want := []string{
		"meal_price",
		"calories",
		"protein",
		"carbs",
		"fat",
		"calorie_difference",
		"price_difference_from_avg",
		"times_meal_eaten",
		"times_category_eaten",
		"days_since_last_eaten",
		"remaining_monthly_budget",
		"average_meal_spending",
		"category_match",
		"meal_type",
	}

	if len(got) != len(want) {
		t.Fatalf("feature count = %d, want %d: %v", len(got), len(want), got)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("feature[%d] = %q, want %q; full order: %v", index, got[index], want[index], got)
		}
	}
}

func TestFeatureSchemaPolicies(t *testing.T) {
	schema, err := LoadSchema()
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	if schema.Version == "" {
		t.Fatal("schema version is required")
	}
	if schema.ObservationWindowDays != 7 {
		t.Fatalf("observation window days = %d, want 7", schema.ObservationWindowDays)
	}
	if schema.HistoryCutoffPolicy == "" {
		t.Fatal("history cutoff policy is required")
	}

	featuresByName := map[string]Feature{}
	for _, feature := range schema.Features {
		if feature.Name == "" {
			t.Fatal("feature name is required")
		}
		if feature.Type == "" {
			t.Fatalf("feature %q type is required", feature.Name)
		}
		if feature.Source == "" {
			t.Fatalf("feature %q source is required", feature.Name)
		}
		if feature.MissingPolicy == "" {
			t.Fatalf("feature %q missing policy is required", feature.Name)
		}
		if !feature.Reliable {
			t.Fatalf("feature %q should not be active until reliable", feature.Name)
		}
		featuresByName[feature.Name] = feature
	}

	if _, found := featuresByName["cuisine_match"]; found {
		t.Fatal("cuisine_match should remain deferred until cuisine data is explicit")
	}
	if featuresByName["days_since_last_eaten"].MissingValue != float64(999) {
		t.Fatalf("days_since_last_eaten missing sentinel = %v, want 999", featuresByName["days_since_last_eaten"].MissingValue)
	}

	foundCuisineDeferred := false
	for _, feature := range schema.DeferredFeatures {
		if feature.Name == "cuisine_match" && feature.Reason != "" {
			foundCuisineDeferred = true
		}
	}
	if !foundCuisineDeferred {
		t.Fatal("cuisine_match deferred decision is required")
	}

	forbidden := map[string]bool{}
	for _, name := range schema.ForbiddenFeatures {
		forbidden[name] = true
	}
	for _, name := range []string{
		"allergy_conflict",
		"prohibited_ingredient",
		"hard_dietary_restriction",
		"medical_exclusion",
		"health_avoid_flag",
	} {
		if !forbidden[name] {
			t.Fatalf("forbidden feature %q is required", name)
		}
	}
}
