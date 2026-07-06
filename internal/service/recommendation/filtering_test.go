package recommendation

import (
	"fyp/food-rs/internal/interfaces"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("filterCandidates", func() {
	It("should remove dietary restriction conflicts meals", func() {
		candidates := []interfaces.MatchedMealCandidate{
			{
				Food: interfaces.FoodSearchResult{
					ID:   "safe",
					Name: "Vegetable Soup",
					Tags: []string{"vegetables", "soups"},
				}},
			{
				Food: interfaces.FoodSearchResult{
					ID:   "pork-food",
					Name: "Pork Noodles",
					Tags: []string{"pork", "noodle_dishes"},
				},
			},
		}
		got := filterCandidates(candidates, interfaces.MealPromptInput{
			DietaryRestrictions: []string{"halal"},
		})

		Expect(got.Filtered).To(HaveLen(1))
		Expect(got.Filtered[0].Food.ID).To(Equal("safe"))

		Expect(got.Removed).To(HaveLen(1))
		Expect(got.Removed[0].Reason).NotTo(BeEmpty())
	})

	It("should remove meals with AVOID health flags", func() {
		candidates := []interfaces.MatchedMealCandidate{
			{
				GeneratedMeal: interfaces.GeneratedMeal{
					Name:        "Sweet Dessert",
					HealthFlags: map[string]string{"diabetes": "AVOID"},
				},
				Food: interfaces.FoodSearchResult{
					ID:   "dessert",
					Name: "Sweet Dessert",
					Tags: []string{"dessert"},
				},
			},
		}
		got := filterCandidates(candidates, interfaces.MealPromptInput{
			HealthConcerns: []string{"diabetes"},
		})

		Expect(got.Filtered).To(BeEmpty())
		Expect(got.Removed).To(HaveLen(1))
	})
})
