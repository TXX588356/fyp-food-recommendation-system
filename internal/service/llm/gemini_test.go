package llm

import (
	"testing"

	"fyp/food-rs/internal/interfaces"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLLM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LLM Suite")
}

func validMeal(name string) interfaces.GeneratedMeal {
	return interfaces.GeneratedMeal{
		Name:                   name,
		AlternativeSearchTerms: []string{},
		EstimatedPriceRange: interfaces.PriceRange{
			Min: 5,
			Max: 8,
		},
		SodiumLevel: "LOW",
		SugarLevel:  "MEDIUM",
		PurineRisk:  "LOW",
		HealthFlags: map[string]string{
			"diabetes": "SAFE",
		},
	}
}

func validResponse() interfaces.GeminiMealsResponse {
	return interfaces.GeminiMealsResponse{
		Meals: []interfaces.GeneratedMeal{
			validMeal("Nasi Lemak"),
			validMeal("Roti Canai"),
			validMeal("Wantan Mee"),
			validMeal("Oatmeal"),
			validMeal("Tom Yum"),
			validMeal("Bubur Ayam"),
		},
	}
}

var _ = Describe("Gemini meal generation", func() {
	Describe("BuildMealRecommendationPrompt", func() {
		It("should include recent meal context", func() {
			prompt := BuildMealRecommendationPrompt(interfaces.MealPromptInput{
				Goal:         "eat_healthier",
				MealCategory: "dinner",
				History: interfaces.MealHistoryContext{
					RecentMealNames:   []string{"Nasi Lemak", "Fried Chicken"},
					RepeatedMealNames: []string{"Nasi Lemak"},
				},
			})

			Expect(prompt).To(ContainSubstring("Recent meals: Nasi Lemak, Fried Chicken"))
			Expect(prompt).To(ContainSubstring("Recently repeated meals: Nasi Lemak"))
		})
	})

	Describe("ParseMeals", func() {
		It("should accept a valid JSON response", func() {
			response := `{
				"meals": [
					{"name":"Nasi Lemak","alternative_search_terms":[],"estimated_price_range":{"min":5,"max":8},"sodium_level":"LOW","sugar_level":"MEDIUM","purine_risk":"LOW","health_flags":{"diabetes":"SAFE"}},
					{"name":"Roti Canai","alternative_search_terms":[],"estimated_price_range":{"min":2,"max":4},"sodium_level":"MEDIUM","sugar_level":"LOW","purine_risk":"LOW","health_flags":{"diabetes":"SAFE"}},
					{"name":"Wantan Mee","alternative_search_terms":[],"estimated_price_range":{"min":6,"max":9},"sodium_level":"HIGH","sugar_level":"LOW","purine_risk":"LOW","health_flags":{"diabetes":"SAFE"}},
					{"name":"Oatmeal","alternative_search_terms":[],"estimated_price_range":{"min":4,"max":7},"sodium_level":"LOW","sugar_level":"LOW","purine_risk":"LOW","health_flags":{"diabetes":"SAFE"}},
					{"name":"Tom Yum","alternative_search_terms":[],"estimated_price_range":{"min":8,"max":14},"sodium_level":"HIGH","sugar_level":"MEDIUM","purine_risk":"MEDIUM","health_flags":{"diabetes":"CAUTION"}},
					{"name":"Bubur Ayam","alternative_search_terms":[],"estimated_price_range":{"min":5,"max":8},"sodium_level":"MEDIUM","sugar_level":"LOW","purine_risk":"LOW","health_flags":{"diabetes":"SAFE"}}
				]
			}`

			result, err := ParseMeals(response, []string{"diabetes"})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Meals).To(HaveLen(6))
		})

		It("should reject invalid JSON", func() {
			_, err := ParseMeals("not-json", nil)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ParseCustomMealAutocomplete", func() {
		It("should accept a valid autocomplete response", func() {
			response := `{
				"calories": 150,
				"fatG": 3,
				"proteinG": 5,
				"carbsG": 22,
				"dietaryRestrictionTags": ["vegetarian"],
				"mealCategoryTags": ["korean", "vegetables"]
			}`

			result, err := ParseCustomMealAutocomplete(response)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Calories).To(Equal(150.0))
			Expect(result.FatG).To(Equal(3.0))
			Expect(result.ProteinG).To(Equal(5.0))
			Expect(result.CarbsG).To(Equal(22.0))
			Expect(result.DietaryRestrictionTags).To(Equal([]string{"vegetarian"}))
			Expect(result.MealCategoryTags).To(Equal([]string{"korean", "vegetables"}))
		})

		It("should reject wrong schema keys instead of silently accepting zero macros", func() {
			response := `{
				"mealName": "Grilled Chicken Salad",
				"calories": 350,
				"carbs": 12,
				"fat": 14,
				"protein": 42,
				"dietaryRestrictionTags": ["gluten-free"],
				"mealCategoryTags": ["lunch"]
			}`

			_, err := ParseCustomMealAutocomplete(response)

			Expect(err).To(MatchError(ContainSubstring("unknown field")))
		})

		It("should reject unsupported tags", func() {
			response := `{
				"calories": 150,
				"fatG": 3,
				"proteinG": 5,
				"carbsG": 22,
				"dietaryRestrictionTags": ["gluten-free"],
				"mealCategoryTags": ["korean"]
			}`

			_, err := ParseCustomMealAutocomplete(response)

			Expect(err).To(MatchError(ContainSubstring("unsupported dietary restriction tag")))
		})

		It("should reject missing meal category tags", func() {
			response := `{
				"calories": 150,
				"fatG": 3,
				"proteinG": 5,
				"carbsG": 22,
				"dietaryRestrictionTags": [],
				"mealCategoryTags": []
			}`

			_, err := ParseCustomMealAutocomplete(response)

			Expect(err).To(MatchError(ContainSubstring("meal category tags are required")))
		})

		It("returns the AI unable-to-generate message", func() {
			response := `{"error":"unable to generate meal details"}`

			_, err := ParseCustomMealAutocomplete(response)

			Expect(err).To(MatchError("unable to generate meal details"))
		})
	})

	Describe("ValidateMeals", func() {
		It("should reject an invalid price range", func() {
			response := validResponse()
			response.Meals[0].EstimatedPriceRange = interfaces.PriceRange{
				Min: 10,
				Max: 5,
			}

			err := ValidateMeals(response, []string{"diabetes"})

			Expect(err).To(HaveOccurred())
		})

		It("should reject an invalid risk level", func() {
			response := validResponse()
			response.Meals[0].SodiumLevel = "VERY_HIGH"

			err := ValidateMeals(response, []string{"diabetes"})

			Expect(err).To(HaveOccurred())
		})

		It("should reject an unexpected health flag", func() {
			response := validResponse()
			response.Meals[0].HealthFlags["gout"] = "SAFE"

			err := ValidateMeals(response, []string{"diabetes"})

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("BuildMealRecommendationPrompt", func() {
		It("should include the supplied user context", func() {
			prompt := BuildMealRecommendationPrompt(interfaces.MealPromptInput{
				Goal:                "eat_healthier",
				DietaryRestrictions: []string{"halal"},
				HealthConcerns:      []string{"diabetes"},
				PreferredMealTags:   []string{"noodles", "chinese"},
				MealCategory:        "dinner",
				MonthlyMealBudget:   300,
				CurrentMonthSpent:   50,
				RemainingBudget:     250,
				PerMealBudget:       8.33,
			})

			expectedValues := []string{
				"Goal: eat_healthier",
				"Dietary restrictions: halal",
				"Health concerns: diabetes",
				"Preferred meal tags: noodles, chinese",
				"Meal category: dinner",
				"Monthly meal budget: RM300.00",
				"Per-meal budget: RM8.33",
			}

			for _, expected := range expectedValues {
				Expect(prompt).To(ContainSubstring(expected))
			}
		})

		It("includes meal timing guidance for lunch and dinner", func() {
			prompt := BuildMealRecommendationPrompt(interfaces.MealPromptInput{
				MealCategory: "dinner",
			})

			expectedValues := []string{
				"Lunch can include more filling meals with higher calories, carbohydrates, and digestive load",
				"Dinner should be lighter than lunch",
				"lower calories, lower carbohydrates, and easier digestion",
			}

			for _, expected := range expectedValues {
				Expect(prompt).To(ContainSubstring(expected))
			}
		})
	})

	Describe("BuildCustomMealAutocompletePrompt", func() {
		It("includes the meal name, exact schema, and supported tag values", func() {
			prompt := BuildCustomMealAutocompletePrompt("Kimchi")

			expectedValues := []string{
				"Meal name: Kimchi",
				`"fatG": 0`,
				`"proteinG": 0`,
				`"carbsG": 0`,
				`{"error":"unable to generate meal details"}`,
				"gluten-free is not valid",
				"korean",
				"vegetables",
			}

			for _, expected := range expectedValues {
				Expect(prompt).To(ContainSubstring(expected))
			}
		})
	})
})
