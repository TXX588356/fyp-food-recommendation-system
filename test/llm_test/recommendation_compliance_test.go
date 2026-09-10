//go:build external

package llm_test

import (
	"fmt"
	"strings"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/service/llm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type complianceScenario struct {
	id                  string
	description         string
	input               interfaces.MealPromptInput
	checkDietary        func(interfaces.GeneratedMeal) bool
	checkBudget         func(interfaces.GeneratedMeal, float64) bool
	expectedDietaryRate float64
	expectedBudgetRate  float64
}

type complianceResult struct {
	total          int
	dietaryOK      int
	budgetOK       int
	dietaryFailed  []string
	budgetFailed   []string
	dietaryRatePct float64
	budgetRatePct  float64
}

var _ = Describe("Recommendation compliance using real LLM API", Ordered, func() {
	var client llm.Client

	BeforeAll(func() {
		ctx, cancel := llmTestContext()
		defer cancel()

		client = newRealGeminiClient(ctx)
	})

	DescribeTable("RC scenarios",
		func(scenario complianceScenario) {
			ctx, cancel := llmTestContext()
			defer cancel()

			response, err := client.GenerateMeals(ctx, scenario.input)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Meals).To(HaveLen(6))

			result := evaluateCompliance(response.Meals, scenario)

			GinkgoWriter.Printf(
				"\n%s %s\nDietary Compliance = %d / %d x 100%% = %.2f%%\nBudget Compliance = %d / %d x 100%% = %.2f%%\n",
				scenario.id,
				scenario.description,
				result.dietaryOK,
				result.total,
				result.dietaryRatePct,
				result.budgetOK,
				result.total,
				result.budgetRatePct,
			)

			Expect(result.dietaryFailed).To(BeEmpty(), strings.Join(result.dietaryFailed, "\n"))
			Expect(result.budgetFailed).To(BeEmpty(), strings.Join(result.budgetFailed, "\n"))
			Expect(result.dietaryRatePct).To(BeNumerically(">=", scenario.expectedDietaryRate))
			Expect(result.budgetRatePct).To(BeNumerically(">=", scenario.expectedBudgetRate))
		},
		Entry("RC-01 non-beef restriction", complianceScenario{
			id:          "RC-01",
			description: "Non-beef restriction",
			input: basePromptInput(
				[]string{"non_beef"},
				nil,
				[]string{"malaysian", "rice_dishes", "poultry", "vegetables"},
				12,
			),
			checkDietary:        containsNone(beefTerms),
			checkBudget:         alwaysBudgetCompliant,
			expectedDietaryRate: 100,
			expectedBudgetRate:  100,
		}),
		Entry("RC-02 vegetarian restriction", complianceScenario{
			id:          "RC-02",
			description: "Vegetarian restriction",
			input: basePromptInput(
				[]string{"vegetarian"},
				nil,
				[]string{"vegetarian", "tofu_soy", "legumes", "vegetables"},
				12,
			),
			checkDietary:        containsNone(meatTerms),
			checkBudget:         alwaysBudgetCompliant,
			expectedDietaryRate: 100,
			expectedBudgetRate:  100,
		}),
		Entry("RC-03 gout health concern", complianceScenario{
			id:          "RC-03",
			description: "Gout health concern",
			input: basePromptInput(
				nil,
				[]string{"gout"},
				[]string{"malaysian", "rice_dishes", "vegetables"},
				12,
			),
			checkDietary:        healthFlagIsNotAvoid("gout"),
			checkBudget:         alwaysBudgetCompliant,
			expectedDietaryRate: 100,
			expectedBudgetRate:  100,
		}),
		Entry("RC-04 diabetes health concern", complianceScenario{
			id:          "RC-04",
			description: "Diabetes health concern",
			input: basePromptInput(
				nil,
				[]string{"diabetes"},
				[]string{"low_sugar", "soups", "salads", "vegetables"},
				12,
			),
			checkDietary:        healthFlagIsNotAvoid("diabetes"),
			checkBudget:         alwaysBudgetCompliant,
			expectedDietaryRate: 100,
			expectedBudgetRate:  100,
		}),
		Entry("RC-05 per-meal budget RM10", complianceScenario{
			id:          "RC-05",
			description: "Per-meal budget RM10",
			input: basePromptInput(
				nil,
				nil,
				[]string{"malaysian", "rice_dishes", "noodle_dishes"},
				10,
			),
			checkDietary:        alwaysDietaryCompliant,
			checkBudget:         budgetFitsImplementedRule,
			expectedDietaryRate: 100,
			expectedBudgetRate:  100,
		}),
		Entry("RC-06 multiple restrictions plus budget", complianceScenario{
			id:          "RC-06",
			description: "Multiple restrictions + budget",
			input: basePromptInput(
				[]string{"non_beef", "vegetarian"},
				[]string{"diabetes", "gout"},
				[]string{"vegetarian", "low_sugar", "tofu_soy", "legumes", "vegetables"},
				10,
			),
			checkDietary: func(meal interfaces.GeneratedMeal) bool {
				return containsNone(meatTerms)(meal) &&
					healthFlagIsNotAvoid("diabetes")(meal) &&
					healthFlagIsNotAvoid("gout")(meal)
			},
			checkBudget:         budgetFitsImplementedRule,
			expectedDietaryRate: 100,
			expectedBudgetRate:  100,
		}),
	)
})

func basePromptInput(
	dietaryRestrictions []string,
	healthConcerns []string,
	preferredMealTags []string,
	perMealBudget float64,
) interfaces.MealPromptInput {
	return interfaces.MealPromptInput{
		Goal:                "eat_healthier",
		DietaryRestrictions: dietaryRestrictions,
		HealthConcerns:      healthConcerns,
		PreferredMealTags:   preferredMealTags,
		MealCategory:        "lunch",
		MonthlyMealBudget:   300,
		CurrentMonthSpent:   0,
		RemainingBudget:     300,
		PerMealBudget:       perMealBudget,
		PriceMarketLocation: "Kuala Lumpur",
		History: interfaces.MealHistoryContext{
			RecentMealNames:        []string{"Nasi Lemak"},
			RepeatedMealNames:      []string{"Nasi Lemak"},
			LearnedCategoryCounts:  map[string]int{"malaysian": 2},
			FatiguedCategoryCounts: map[string]int{"fried_foods": 2},
		},
	}
}

func evaluateCompliance(meals []interfaces.GeneratedMeal, scenario complianceScenario) complianceResult {
	result := complianceResult{total: len(meals)}

	for _, meal := range meals {
		if scenario.checkDietary(meal) {
			result.dietaryOK++
		} else {
			result.dietaryFailed = append(result.dietaryFailed, fmt.Sprintf(
				"%s dietary violation: %q health_flags=%v price=RM%.2f-RM%.2f",
				scenario.id,
				meal.Name,
				meal.HealthFlags,
				meal.EstimatedPriceRange.Min,
				meal.EstimatedPriceRange.Max,
			))
		}

		if scenario.checkBudget(meal, scenario.input.PerMealBudget) {
			result.budgetOK++
		} else {
			result.budgetFailed = append(result.budgetFailed, fmt.Sprintf(
				"%s budget violation: %q minimum estimated price RM%.2f exceeds per-meal budget RM%.2f; price=RM%.2f-RM%.2f",
				scenario.id,
				meal.Name,
				meal.EstimatedPriceRange.Min,
				scenario.input.PerMealBudget,
				meal.EstimatedPriceRange.Min,
				meal.EstimatedPriceRange.Max,
			))
		}
	}

	if result.total > 0 {
		result.dietaryRatePct = float64(result.dietaryOK) / float64(result.total) * 100
		result.budgetRatePct = float64(result.budgetOK) / float64(result.total) * 100
	}

	return result
}

func alwaysDietaryCompliant(interfaces.GeneratedMeal) bool {
	return true
}

func alwaysBudgetCompliant(interfaces.GeneratedMeal, float64) bool {
	return true
}

func containsNone(blockedTerms []string) func(interfaces.GeneratedMeal) bool {
	return func(meal interfaces.GeneratedMeal) bool {
		haystack := strings.ToLower(meal.Name + " " + strings.Join(meal.AlternativeSearchTerms, " "))
		for _, term := range blockedTerms {
			if strings.Contains(haystack, term) {
				return false
			}
		}
		return true
	}
}

func healthFlagIsNotAvoid(concern string) func(interfaces.GeneratedMeal) bool {
	return func(meal interfaces.GeneratedMeal) bool {
		return meal.HealthFlags[concern] != "AVOID"
	}
}

func budgetFitsImplementedRule(meal interfaces.GeneratedMeal, perMealBudget float64) bool {
	if perMealBudget <= 0 {
		return true
	}
	return meal.EstimatedPriceRange.Min <= perMealBudget
}

var beefTerms = []string{
	"beef",
	"rendang daging",
	"daging",
}

var meatTerms = []string{
	"beef",
	"chicken",
	"duck",
	"pork",
	"lamb",
	"mutton",
	"fish",
	"seafood",
	"shrimp",
	"prawn",
	"squid",
	"anchovy",
	"anchovies",
	"ikan",
	"ayam",
	"daging",
	"kambing",
	"udang",
	"sotong",
}
