package llm

import (
	"strings"
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
	Describe("ParseMeals", func() {
		It("accepts a valid JSON response", func() {
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

		It("rejects invalid JSON", func() {
			_, err := ParseMeals("not-json", nil)

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ValidateMeals", func() {
		It("rejects an invalid price range", func() {
			response := validResponse()
			response.Meals[0].EstimatedPriceRange = interfaces.PriceRange{
				Min: 10,
				Max: 5,
			}

			err := ValidateMeals(response, []string{"diabetes"})

			Expect(err).To(HaveOccurred())
		})

		It("rejects an invalid risk level", func() {
			response := validResponse()
			response.Meals[0].SodiumLevel = "VERY_HIGH"

			err := ValidateMeals(response, []string{"diabetes"})

			Expect(err).To(HaveOccurred())
		})

		It("rejects an unexpected health flag", func() {
			response := validResponse()
			response.Meals[0].HealthFlags["gout"] = "SAFE"

			err := ValidateMeals(response, []string{"diabetes"})

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("BuildMealRecommendationPrompt", func() {
		It("includes the supplied user context", func() {
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
				Expect(strings.Contains(prompt, expected)).To(
					BeTrue(),
					"expected prompt to contain %q",
					expected,
				)
			}
		})
	})
})
