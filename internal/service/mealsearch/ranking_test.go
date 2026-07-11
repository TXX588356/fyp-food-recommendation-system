package mealsearch

import (
	"fyp/food-rs/internal/interfaces"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Meal search ranking", func() {
	Describe("RankFoodName", func() {
		It("should rank an exact canonical name match with the highest score", func() {
			match, found := RankFoodName("Nasi Lemak", nil, []string{" nasi   lemak "})

			Expect(found).To(BeTrue())
			Expect(match.Kind).To(Equal(interfaces.FoodMatchExactName))
			Expect(match.MatchedTerm).To(Equal("nasi   lemak"))
			Expect(match.Score).To(Equal(1.00))
		})

		It("should rank an exact alias match below an exact canonical name match", func() {
			match, found := RankFoodName("Wantan Mee", []string{"Wonton Mee"}, []string{"wonton mee"})

			Expect(found).To(BeTrue())
			Expect(match.Kind).To(Equal(interfaces.FoodMatchExactAlias))
			Expect(match.MatchedTerm).To(Equal("wonton mee"))
			Expect(match.Score).To(Equal(0.98))
		})

		It("should choose the strongest match across multiple generated search terms", func() {
			match, found := RankFoodName(
				"Wantan Mee",
				[]string{"Wonton Mee"},
				[]string{"Wantan Mee Soup", "Wonton Mee", "Wantan Mee"},
			)

			Expect(found).To(BeTrue())
			Expect(match.Kind).To(Equal(interfaces.FoodMatchExactName))
			Expect(match.MatchedTerm).To(Equal("Wantan Mee"))
			Expect(match.Score).To(Equal(1.00))
		})

		It("should rank fuzzy full-query matches below exact alias matches", func() {
			match, found := RankFoodName("Nasi Lemak Ayam Goreng", nil, []string{"nasi lemak ayam gorng"})

			Expect(found).To(BeTrue())
			Expect(match.Kind).To(Equal(interfaces.FoodMatchFuzzy))
			Expect(match.MatchedTerm).To(Equal("nasi lemak ayam gorng"))
			Expect(match.Score).To(BeNumerically("<", 0.98))
			Expect(match.Score).To(BeNumerically(">", 0.70))
		})

		It("should rank partial phrase matches below fuzzy full-query matches", func() {
			match, found := RankFoodName("Fried Rice", nil, []string{"Fried Rice Chicken"})

			Expect(found).To(BeTrue())
			Expect(match.Kind).To(Equal(interfaces.FoodMatchPartial))
			Expect(match.MatchedTerm).To(Equal("Fried Rice Chicken"))
			Expect(match.Score).To(BeNumerically("<", 0.85))
		})

		It("should keep multiple plausible sausage candidates eligible for adjudication", func() {
			breakfastMatch, breakfastFound := RankFoodName("Chicken Breakfast Sausage", nil, []string{"Chicken Sausage"})
			grilledMatch, grilledFound := RankFoodName("Grilled Chicken Sausage", nil, []string{"Chicken Sausage"})

			Expect(breakfastFound).To(BeTrue())
			Expect(grilledFound).To(BeTrue())
			Expect(breakfastMatch.Kind).To(Equal(interfaces.FoodMatchFuzzy))
			Expect(grilledMatch.Kind).To(Equal(interfaces.FoodMatchFuzzy))
		})

		It("should return false when no generated term plausibly matches the food name", func() {
			match, found := RankFoodName("Tofu", nil, []string{"Chicken Sausage", "   "})

			Expect(found).To(BeFalse())
			Expect(match).To(Equal(RankedMatch{}))
		})

		It("should ignore duplicate and empty generated terms", func() {
			match, found := RankFoodName(
				"Roti Canai",
				nil,
				[]string{"", " roti   canai ", "ROTI CANAI"},
			)

			Expect(found).To(BeTrue())
			Expect(match.Kind).To(Equal(interfaces.FoodMatchExactName))
			Expect(match.MatchedTerm).To(Equal("roti   canai"))
		})
	})

	Describe("SortFoodMatchCandidates", func() {
		It("should sort candidates by score, match kind, normalized name, and stable ID", func() {
			candidates := []interfaces.FoodMatchCandidate{
				{
					Food:      interfaces.FoodSearchResult{ID: "z", Name: "Zucchini Soup"},
					MatchKind: interfaces.FoodMatchFuzzy,
					Score:     0.80,
				},
				{
					Food:      interfaces.FoodSearchResult{ID: "b", Name: "Banana Pancake"},
					MatchKind: interfaces.FoodMatchFuzzy,
					Score:     0.85,
				},
				{
					Food:      interfaces.FoodSearchResult{ID: "a", Name: "Apple Pancake"},
					MatchKind: interfaces.FoodMatchFuzzy,
					Score:     0.85,
				},
				{
					Food:      interfaces.FoodSearchResult{ID: "c", Name: "Apple Pancake"},
					MatchKind: interfaces.FoodMatchExactAlias,
					Score:     0.85,
				},
			}

			SortFoodMatchCandidates(candidates)

			Expect([]string{
				candidates[0].Food.ID,
				candidates[1].Food.ID,
				candidates[2].Food.ID,
				candidates[3].Food.ID,
			}).To(Equal([]string{"c", "a", "b", "z"}))
		})

		It("should compare equivalent candidates as equal", func() {
			candidate := interfaces.FoodMatchCandidate{
				Food:      interfaces.FoodSearchResult{ID: "same", Name: "Same Food"},
				MatchKind: interfaces.FoodMatchExactName,
				Score:     1.00,
			}

			Expect(CompareFoodMatchCandidates(candidate, candidate)).To(Equal(0))
		})
	})

	Describe("internal matching helpers", func() {
		It("should not fuzzy-match short words with one-character differences", func() {
			Expect(mealWordFuzzyMatch("mee", "bee")).To(BeFalse())
		})

		It("should fuzzy-match longer words with small typos", func() {
			Expect(mealWordFuzzyMatch("spaghetti", "spagheti")).To(BeTrue())
		})

		It("should not match a single ingredient as a partial phrase", func() {
			Expect(partialPhraseMatch("Tofu", "Tofu Ramen")).To(BeFalse())
		})
	})
})
