package preference

import "testing"

func TestValidateMealPreferencesAcceptsCurrentFrontendCategoryCodes(t *testing.T) {
	tags := []string{
		"singaporean",
		"rice_dishes",
		"noodle_dishes",
		"condiments_sauces",
		"poultry",
		"tofu_soy",
		"nuts_seeds",
	}

	if err := validateMealPreferences(tags); err != nil {
		t.Fatalf("expected current frontend meal category codes to be accepted, got %v", err)
	}
}

func TestValidateMealPreferencesAgainstDietaryRestrictionsUsesCurrentCategoryCodes(t *testing.T) {
	err := validateMealPreferencesAgainstDietaryRestrictions([]string{"nuts_seeds"}, []string{"nut_free"})
	if err == nil {
		t.Fatal("expected nuts_seeds to conflict with nut_free")
	}
}
