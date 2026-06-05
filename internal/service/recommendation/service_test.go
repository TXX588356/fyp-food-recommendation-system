package recommendation

import (
	"context"
	"errors"
	"testing"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

// TestRecommendation runs the recommendation service test suite.
func TestRecommendation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Recommendation Suite")
}

var _ = Describe("Recommendation candidate generation", func() {
	var (
		ctx           context.Context
		mealGenerator *mocks.MealGenerator
		foodSearcher  *mocks.FoodSearcher
		service       interfaces.RecommendationService
		input         interfaces.MealPromptInput
	)

	BeforeEach(func() {
		ctx = context.Background()
		input = interfaces.MealPromptInput{}
		mealGenerator = mocks.NewMealGenerator(GinkgoT())
		foodSearcher = mocks.NewFoodSearcher(GinkgoT())
		service = NewService(mealGenerator, foodSearcher)
	})

	It("should match a meal using its normalized name first", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Nasi Lemak"},
				},
			}, nil).
			Once()

		foodSearcher.EXPECT().SearchFood(mock.Anything, "Nasi Lemak").
			Return(interfaces.FoodSearchResult{
				ID:   "food-1",
				Name: "Nasi Lemak",
			}, true, nil).
			Once()

		candidates, err := service.GenerateCandidates(ctx, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].MatchedQuery).To(Equal("Nasi Lemak"))
	})

	It("should try alternative search terms when the normalized name is not found", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{
						Name: "Wantan Mee",
						AlternativeSearchTerms: []string{
							"Wonton Mee",
							"Wan Tan Mee",
						},
					},
				},
			}, nil).
			Once()

		foodSearcher.EXPECT().SearchFood(mock.Anything, "Wantan Mee").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()
		foodSearcher.EXPECT().SearchFood(mock.Anything, "Wonton Mee").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()
		foodSearcher.EXPECT().SearchFood(mock.Anything, "Wan Tan Mee").
			Return(interfaces.FoodSearchResult{
				ID:   "food-2",
				Name: "Wantan Mee",
			}, true, nil).
			Once()

		candidates, err := service.GenerateCandidates(ctx, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].MatchedQuery).To(Equal("Wan Tan Mee"))
	})

	It("should discard a meal when every search term returns no result", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{
						Name:                   "Unknown Meal",
						AlternativeSearchTerms: []string{"Unknown Food"},
					},
				},
			}, nil).
			Once()

		foodSearcher.EXPECT().SearchFood(mock.Anything, "Unknown Meal").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()
		foodSearcher.EXPECT().SearchFood(mock.Anything, "Unknown Food").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()

		candidates, err := service.GenerateCandidates(ctx, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(BeEmpty())
	})

	It("should return a controlled error when Gemini fails", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{}, errors.New("Gemini unavailable")).
			Once()

		_, err := service.GenerateCandidates(ctx, input)

		Expect(err).To(MatchError("generated meal candidates: Gemini unavailable"))
	})

	It("should return a controlled error when food search fails", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Nasi Lemak"},
				},
			}, nil).
			Once()

		foodSearcher.EXPECT().SearchFood(mock.Anything, "Nasi Lemak").
			Return(interfaces.FoodSearchResult{}, false, errors.New("food API unavailable")).
			Once()

		_, err := service.GenerateCandidates(ctx, input)

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
