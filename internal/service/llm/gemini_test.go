package llm

import "testing"

func TestParseMealsAcceptsFencedJSON(t *testing.T) {
	input := "```json\n{\"meals\":[{\"name\":\"Nasi Lemak\",\"format\":\"rice\"}]}\n```"

	got, err := ParseMeals(input)
	if err != nil {
		t.Fatalf("ParseMeals returned error: %v", err)
	}

	if len(got.Meals) != 1 {
		t.Fatalf("expected 1 meal, got %d", len(got.Meals))
	}
	if got.Meals[0].Name != "Nasi Lemak" {
		t.Fatalf("expected Nasi Lemak, got %q", got.Meals[0].Name)
	}
}

func TestParseMealsRejectsInvalidJSON(t *testing.T) {
	_, err := ParseMeals("not-json")
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}
