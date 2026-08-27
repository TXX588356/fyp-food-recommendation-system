//go:build external

package external_test

import (
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/service/llm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/genai"
)

var _ = Describe("Gemini external integration", func() {
	Describe("Generate meal candidates using real Gemini API", func() {
		It("should return generated meals that can be parsed into the expected structure", func() {
			ctx, cancel := externalTestContext()
			defer cancel()

			client := newRealGeminiClient(ctx)

			result, err := client.GenerateMeals(ctx, interfaces.MealPromptInput{
				Goal:                "eat_healthier",
				DietaryRestrictions: []string{"halal"},
				HealthConcerns:      []string{"high_blood_pressure"},
				PreferredMealTags:   []string{"malaysian", "rice_dishes"},
				MealCategory:        "lunch",
				MonthlyMealBudget:   500,
				CurrentMonthSpent:   120,
				RemainingBudget:     380,
				PerMealBudget:       12,
				PriceMarketLocation: "Pavilion Kuala Lumpur",
				History: interfaces.MealHistoryContext{
					RecentMealNames:        []string{"Nasi Lemak"},
					RepeatedMealNames:      []string{"Nasi Lemak"},
					LearnedCategoryCounts:  map[string]int{"malaysian": 3},
					FatiguedCategoryCounts: map[string]int{"rice_dishes": 2},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Meals).NotTo(BeEmpty())

			first := result.Meals[0]
			Expect(first.Name).NotTo(BeEmpty())
			Expect(first.EstimatedPriceRange.Min).To(BeNumerically(">=", 0))
			Expect(first.EstimatedPriceRange.Max).To(BeNumerically(">=", first.EstimatedPriceRange.Min))
			Expect(first.SodiumLevel).To(Or(Equal("LOW"), Equal("MEDIUM"), Equal("HIGH")))
			Expect(first.SugarLevel).To(Or(Equal("LOW"), Equal("MEDIUM"), Equal("HIGH")))
			Expect(first.PurineRisk).To(Or(Equal("LOW"), Equal("MEDIUM"), Equal("HIGH")))
		})
	})

	Describe("Resolve ambiguous meal match using real Gemini API", func() {
		It("should return a valid MATCH or NO_MATCH decision", func() {
			ctx, cancel := externalTestContext()
			defer cancel()

			client := newRealGeminiClient(ctx)

			decisions, err := client.ResolveMatches(ctx, []interfaces.MealMatchTask{
				{
					MealIndex: 0,
					GeneratedMeal: interfaces.GeneratedMeal{
						Name:                   "Chicken Noodles",
						AlternativeSearchTerms: []string{"Chicken Mee"},
					},
					Candidates: []interfaces.FoodMatchCandidate{
						{
							Food: interfaces.FoodSearchResult{
								ID:   "candidate-1",
								Name: "Chicken Noodle Soup",
								Tags: []string{"chinese", "noodle_dishes", "poultry"},
							},
						},
						{
							Food: interfaces.FoodSearchResult{
								ID:   "candidate-2",
								Name: "Chicken Fried Noodles",
								Tags: []string{"chinese", "noodle_dishes", "poultry"},
							},
						},
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(decisions).To(HaveLen(1))
			Expect(decisions[0].MealIndex).To(Equal(0))
			Expect(decisions[0].Decision).To(Or(Equal("MATCH"), Equal("NO_MATCH")))

			if decisions[0].Decision == "MATCH" {
				Expect(decisions[0].CandidateID).To(Or(Equal("candidate-1"), Equal("candidate-2")))
			} else {
				Expect(decisions[0].CandidateID).To(BeEmpty())
			}
		})
	})

	Describe("Generate recommendation explanation using real Gemini API", func() {
		It("should return a non-empty explanation", func() {
			ctx, cancel := externalTestContext()
			defer cancel()

			client := newRealGeminiClient(ctx)

			explanation, err := client.ExplainMealRecommendation(ctx, interfaces.MealDetailExplanationInput{
				MealName:     "Chicken Rice",
				MealCategory: "lunch",
				Nutrition: interfaces.MealDetailNutritionInput{
					Calories: 620,
					FatG:     18,
					ProteinG: 32,
					CarbsG:   72,
				},
				PriceRange: interfaces.PriceRange{
					Min: 8,
					Max: 12,
				},
				SodiumLevel: "MEDIUM",
				SugarLevel:  "LOW",
				PurineRisk:  "LOW",
				HealthFlags: map[string]string{
					"high_blood_pressure": "CAUTION",
				},
				UserGoal:            "eat_healthier",
				DietaryRestrictions: []string{"halal"},
				HealthConcerns:      []string{"high_blood_pressure"},
				PreferredMealTags:   []string{"malaysian", "rice_dishes"},
				MonthlyMealBudget:   500,
				CurrentMonthSpent:   120,
				RemainingBudget:     380,
				PerMealBudget:       12,
				Location:            "Pavilion Kuala Lumpur",
				LocationBasis:       "work_school",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(explanation).NotTo(BeEmpty())
		})
	})

	Describe("Handle unavailable or invalid Gemini API response", func() {
		It("should return an error without crashing", func() {
			ctx, cancel := externalTestContext()
			defer cancel()

			// Intentionally create a bad client by requiring no real key here.
			// This verifies backend error handling when Gemini rejects the request.
			client, err := genai.NewClient(ctx, &genai.ClientConfig{
				APIKey:  "invalid-api-key",
				Backend: genai.BackendGeminiAPI,
			})
			Expect(err).NotTo(HaveOccurred())

			llmClient := llm.NewClient(client)

			result, err := llmClient.GenerateMeals(ctx, interfaces.MealPromptInput{
				Goal:                "eat_healthier",
				DietaryRestrictions: []string{"halal"},
				HealthConcerns:      []string{"high_blood_pressure"},
				PreferredMealTags:   []string{"malaysian"},
				MealCategory:        "lunch",
				MonthlyMealBudget:   500,
				RemainingBudget:     500,
				PerMealBudget:       12,
				PriceMarketLocation: "Pavilion Kuala Lumpur",
			})

			Expect(err).To(HaveOccurred())
			Expect(result.Meals).To(BeEmpty())
		})
	})
})
