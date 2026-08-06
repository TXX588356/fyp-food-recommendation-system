package recommendation

import (
	"fyp/food-rs/internal/interfaces"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("recommendation ranking", func() {
	It("should score candidates with goal, budget, recency penalty, and preferences", func() {
		history := interfaces.MealHistoryContext{
			RecentlyEatenByName: map[string]int{},
		}
		input := interfaces.MealPromptInput{
			Goal:              "eat_healthier",
			PerMealBudget:     10,
			PreferredMealTags: []string{"soups", "vegetables"},
		}
		candidate := interfaces.MatchedMealCandidate{
			Food: interfaces.FoodSearchResult{
				ID:       "soup",
				Name:     "Vegetable Soup",
				Tags:     []string{"soups", "vegetables"},
				FatG:     8,
				ProteinG: 18,
				CarbsG:   35,
			},
			GeneratedMeal: interfaces.GeneratedMeal{
				EstimatedPriceRange: interfaces.PriceRange{Min: 8, Max: 8},
			},
		}

		got := scoreCandidate(candidate, input, history)

		Expect(got.Total).To(BeNumerically(">=", 80))
		Expect(got.PreferenceScore).To(Equal(float64(30)))
	})

	It("should prioritize protein for muscle gain goal", func() {
		candidates := []interfaces.MatchedMealCandidate{
			{
				Food: interfaces.FoodSearchResult{
					ID:       "low_protein_food",
					Name:     "Plain porridge",
					ProteinG: 5,
					FatG:     3,
					CarbsG:   40,
				},
				GeneratedMeal: interfaces.GeneratedMeal{
					EstimatedPriceRange: interfaces.PriceRange{Min: 6, Max: 6},
				},
			},
			{
				Food: interfaces.FoodSearchResult{
					ID:       "high_protein_food",
					Name:     "Grilled Chicken",
					ProteinG: 35,
					FatG:     10,
					CarbsG:   30,
				},
				GeneratedMeal: interfaces.GeneratedMeal{
					EstimatedPriceRange: interfaces.PriceRange{Min: 9, Max: 9},
				},
			},
		}

		got := rankCandidates(candidates, interfaces.MealPromptInput{
			Goal:          "muscle_gain",
			PerMealBudget: 10,
		}, interfaces.MealHistoryContext{})

		Expect(got[0].Food.ID).To(Equal("high_protein_food"))
		Expect(got[0].Score).To(BeNumerically(">", got[1].Score))
		Expect(got[0].ScoreBreakdown.GoalAlignment).To(BeNumerically(">", got[1].ScoreBreakdown.GoalAlignment))
		Expect(got[0].ScoreBreakdown.BudgetFit).To(BeNumerically(">", 0))
	})

	It("should rank recently repeated meals lower", func() {
		candidates := []interfaces.MatchedMealCandidate{
			{Food: interfaces.FoodSearchResult{ID: "repeat_food", Name: "Nasi Lemak"}},
			{Food: interfaces.FoodSearchResult{ID: "new_food", Name: "Chicken Chop"}},
		}

		got := rankCandidates(candidates, interfaces.MealPromptInput{PerMealBudget: 10}, interfaces.MealHistoryContext{
			RecentlyEatenByName: map[string]int{"nasi lemak": 1},
		})

		Expect(got[0].Food.ID).To(Equal("new_food"))
	})

	Describe("budgetFitScore", func() {
		It("should keep partial budget credit for a modestly over-budget price range", func() {
			got := budgetFitScore(8.5, 12.5, 10.13)

			Expect(got).To(BeNumerically("~", 11.61, 0.01))
		})

		It("should keep a floor score at twenty-five percent over budget", func() {
			got := budgetFitScore(12.5, 12.5, 10)

			Expect(got).To(Equal(float64(8)))
		})

		It("should return zero at fifty percent over budget", func() {
			got := budgetFitScore(15, 15, 10)

			Expect(got).To(Equal(float64(0)))
		})
	})
})
