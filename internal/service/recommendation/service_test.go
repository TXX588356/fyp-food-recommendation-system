package recommendation

import (
	"context"
	"errors"
	"fyp/food-rs/internal/interfaces"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRecommendation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Recommendation Suite")
}

type stubMealGenerator struct {
	response interfaces.GeminiMealsResponse
	err      error
}

func (s stubMealGenerator) GenerateMeals(ctx context.Context, input interfaces.MealPromptInput) (interfaces.GeminiMealsResponse, error) {
	return s.response, s.err
}

type stubFoodSearcher struct {
	results map[string]interfaces.FoodSearchResult
	errors  map[string]error
	queries []string // slice that records every query setn to the stub
}

func (s *stubFoodSearcher) SearchFood(ctx context.Context, query string) (interfaces.FoodSearchResult, bool, error) {
	s.queries = append(s.queries, query) // append queries each time SearchFood is called

	if err := s.errors[query]; err != nil {
		return interfaces.FoodSearchResult{}, false, err
	}

	food, found := s.results[query]
	return food, found, nil
}

var _ = Describe("Recommendation candidate generation", func() {
	var (
		ctx          context.Context
		foodSearcher *stubFoodSearcher
	)

	BeforeEach(func() {
		ctx = context.Background()
		foodSearcher = &stubFoodSearcher{
			results: map[string]interfaces.FoodSearchResult{},
			errors:  map[string]error{},
		}
	})

	It("should match a meal using its normalized name first", func() {
		foodSearcher.results["Nasi Lemak"] = interfaces.FoodSearchResult{
			ID:   "food-1",
			Name: "Nasi Lemak",
		}

		service := NewService(stubMealGenerator{
			response: interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Nasi Lemak"},
				},
			},
		}, foodSearcher)

		candidates, err := service.GenerateCandidates(ctx, interfaces.MealPromptInput{})

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].MatchedQuery).To(Equal("Nasi Lemak"))
		Expect(foodSearcher.queries).To(Equal([]string{"Nasi Lemak"}))
	})

	It("should try alternative search terms when the normalized name is not found", func() {
		foodSearcher.results["Wan Tan Mee"] = interfaces.FoodSearchResult{
			ID:   "food-2",
			Name: "Wantan Mee",
		}

		service := NewService(stubMealGenerator{
			response: interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{
						Name: "Wantan Mee",
						AlternativeSearchTerms: []string{
							"Wonton Mee",
							"Wan Tan Mee",
						},
					},
				},
			},
		}, foodSearcher)

		candidates, err := service.GenerateCandidates(ctx, interfaces.MealPromptInput{})

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].MatchedQuery).To(Equal("Wan Tan Mee"))

		// verifies the order of API search attempts
		Expect(foodSearcher.queries).To(Equal([]string{
			"Wantan Mee",
			"Wonton Mee",
			"Wan Tan Mee",
		}))
	})

	It("should discard a meal when every search term returns no result", func() {
		service := NewService(stubMealGenerator{
			response: interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{
						Name: "Unknown Meal",
						AlternativeSearchTerms: []string{
							"Unknown Food",
						},
					},
				},
			},
		}, foodSearcher,
		)

		candidates, err := service.GenerateCandidates(ctx, interfaces.MealPromptInput{})

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(BeEmpty())
	})

	It("should return a controlled error when Gemini fails", func() {
		service := NewService(stubMealGenerator{
			err: errors.New("Gemini unavailable"),
		}, foodSearcher)

		_, err := service.GenerateCandidates(ctx, interfaces.MealPromptInput{})

		Expect(err).To(MatchError("generated meal candidates: Gemini unavailable"))
	})

	It("should return a controlled error when food search fails", func() {
		foodSearcher.errors["Nasi Lemak"] = errors.New("food API unavailable")

		service := NewService(stubMealGenerator{
			response: interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Nasi Lemak"},
				},
			},
		}, foodSearcher)

		_, err := service.GenerateCandidates(ctx, interfaces.MealPromptInput{})

		Expect(err).To(MatchError(`search food "Nasi Lemak": food API unavailable`))
	})

	It("should remove duplicate and blank fallback search terms", func() {
		terms := buildSearchTerms(interfaces.GeneratedMeal{
			Name: "Wantan Mee",
			AlternativeSearchTerms: []string{
				" ",
				"wantan mee",
				"Wan Tan Mee",
			},
		})

		Expect(terms).To(Equal([]string{
			"Wantan Mee",
			"Wan Tan Mee",
		}))
	})
})
