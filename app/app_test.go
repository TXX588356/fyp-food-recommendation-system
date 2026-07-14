package app

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type stubMealGenerator struct{}

func (stubMealGenerator) GenerateMeals(ctx context.Context, input interfaces.MealPromptInput) (interfaces.GeminiMealsResponse, error) {
	return interfaces.GeminiMealsResponse{}, nil
}

func (stubMealGenerator) AutocompleteCustomMeal(ctx context.Context, input interfaces.CustomMealAutocompleteInput) (interfaces.CustomMealAutocompleteResponse, error) {
	return interfaces.CustomMealAutocompleteResponse{}, nil
}

func (stubMealGenerator) ResolveMatches(ctx context.Context, tasks []interfaces.MealMatchTask) ([]interfaces.MealMatchDecision, error) {
	return nil, nil
}

func (stubMealGenerator) ExplainMealRecommendation(ctx context.Context, input interfaces.MealDetailExplanationInput) (string, error) {
	return "This meal fits the user's current preferences.", nil
}

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Suite")
}

var _ = Describe("App dependencies", func() {
	var originalMealGeneratorFactory func(context.Context, string) (AIClient, error)

	BeforeEach(func() {
		originalMealGeneratorFactory = newMealGenerator
	})

	AfterEach(func() {
		newMealGenerator = originalMealGeneratorFactory
	})

	It("should use a cached catalog searcher for recommendations", func() {
		var gotGeminiAPIKey string

		newMealGenerator = func(ctx context.Context, apiKey string) (AIClient, error) {
			gotGeminiAPIKey = apiKey
			return stubMealGenerator{}, nil
		}
		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images")

		service, err := a.GetRecommendationService(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(service).NotTo(BeNil())
		Expect(gotGeminiAPIKey).To(Equal("gemini-key"))

		first, err := a.GetCatalogFoodSearcher(context.Background())
		Expect(err).NotTo(HaveOccurred())
		second, err := a.GetCatalogFoodSearcher(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(first).To(BeIdenticalTo(second))
	})

	It("should use a cached AI client for custom meal autocomplete", func() {
		factoryCalls := 0

		newMealGenerator = func(ctx context.Context, apiKey string) (AIClient, error) {
			factoryCalls++
			return stubMealGenerator{}, nil
		}

		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images")

		first, err := a.GetCustomMealAutocompleter(context.Background())
		Expect(err).NotTo(HaveOccurred())

		second, err := a.GetCustomMealAutocompleter(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(first).To(BeIdenticalTo(second))
		Expect(factoryCalls).To(Equal(1))
	})
})
