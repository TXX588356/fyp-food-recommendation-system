package llm

import (
	"fyp/food-rs/internal/interfaces"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gemini meal match adjudication", func() {
	Describe("BuildMealMatchingPrompt", func() {
		It("should include all ambiguous tasks in one prompt", func() {
			tasks := []interfaces.MealMatchTask{
				matchTask(0, "Chicken Sausage", []interfaces.FoodMatchCandidate{
					matchCandidate("ordinary-sausage", "Chicken Breakfast Sausage", []string{"breakfast", "poultry"}),
					matchCandidate("grilled-sausage", "Grilled Chicken Sausage", []string{"poultry"}),
				}),
				matchTask(2, "Fried Rice", []interfaces.FoodMatchCandidate{
					matchCandidate("fried-rice", "Fried Rice", []string{"rice_dishes"}),
				}),
			}

			prompt, err := BuildMealMatchingPrompt(tasks)

			Expect(err).NotTo(HaveOccurred())

			// The prompt should contain both tasks, preserving original meal indexes.
			Expect(prompt).To(ContainSubstring(`"meal_index": 0`))
			Expect(prompt).To(ContainSubstring(`"meal_index": 2`))

			// The generated meal and supplied candidates should appear in the prompt.
			Expect(prompt).To(ContainSubstring(`"name": "Chicken Sausage"`))
			Expect(prompt).To(ContainSubstring(`"id": "ordinary-sausage"`))
			Expect(prompt).To(ContainSubstring(`"id": "grilled-sausage"`))
			Expect(prompt).To(ContainSubstring(`"name": "Fried Rice"`))

			// The prompt should require the key safety behavior.
			Expect(prompt).To(ContainSubstring("semantic food identity"))
			Expect(prompt).To(ContainSubstring("Return NO_MATCH"))
			Expect(prompt).To(ContainSubstring("Return only candidate IDs"))
			Expect(prompt).To(ContainSubstring("exactly one decision for every meal_index"))
		})

		It("should include alternative search terms for generated meals", func() {
			tasks := []interfaces.MealMatchTask{
				{
					MealIndex: 0,
					GeneratedMeal: interfaces.GeneratedMeal{
						Name:                   "Wantan Mee",
						AlternativeSearchTerms: []string{"Wonton Mee", "Wan Tan Mee"},
					},
					Candidates: []interfaces.FoodMatchCandidate{
						matchCandidate("wantan-mee", "Wantan Mee", []string{"noodle_dishes"}),
					},
				},
			}

			prompt, err := BuildMealMatchingPrompt(tasks)

			Expect(err).NotTo(HaveOccurred())
			Expect(prompt).To(ContainSubstring(`"alternative_search_terms": [`))
			Expect(prompt).To(ContainSubstring(`"Wonton Mee"`))
			Expect(prompt).To(ContainSubstring(`"Wan Tan Mee"`))
		})

		It("should not send calories or macros in the prompt payload", func() {
			tasks := []interfaces.MealMatchTask{
				matchTask(0, "Nasi Lemak", []interfaces.FoodMatchCandidate{
					matchCandidate("nasi-lemak", "Nasi Lemak", []string{"rice_dishes"}),
				}),
			}

			prompt, err := BuildMealMatchingPrompt(tasks)

			Expect(err).NotTo(HaveOccurred())

			// Nutrition values should not be used for identity adjudication.
			Expect(prompt).NotTo(ContainSubstring("calories"))
			Expect(prompt).NotTo(ContainSubstring("protein"))
			Expect(prompt).NotTo(ContainSubstring("carbs"))
			Expect(prompt).NotTo(ContainSubstring("fat"))
		})
	})

	Describe("ParseMealMatchDecisions", func() {
		It("should accept valid MATCH and NO_MATCH decisions", func() {
			response := `{
				"decisions": [
					{"meal_index": 0, "decision": "MATCH", "candidate_id": "ordinary-sausage"},
					{"meal_index": 1, "decision": "NO_MATCH"}
				]
			}`

			decisions, err := ParseMealMatchDecisions(response)

			Expect(err).NotTo(HaveOccurred())
			Expect(decisions).To(HaveLen(2))

			Expect(decisions[0].MealIndex).To(Equal(0))
			Expect(decisions[0].Decision).To(Equal(MatchDecisionMatch))
			Expect(decisions[0].CandidateID).To(Equal("ordinary-sausage"))

			Expect(decisions[1].MealIndex).To(Equal(1))
			Expect(decisions[1].Decision).To(Equal(MatchDecisionNoMatch))
			Expect(decisions[1].CandidateID).To(BeEmpty())
		})

		It("should accept JSON wrapped in markdown fences", func() {
			response := "```json\n" + `{
				"decisions": [
					{"meal_index": 0, "decision": "NO_MATCH"}
				]
			}` + "\n```"

			decisions, err := ParseMealMatchDecisions(response)

			Expect(err).NotTo(HaveOccurred())
			Expect(decisions).To(HaveLen(1))
			Expect(decisions[0].Decision).To(Equal(MatchDecisionNoMatch))
		})

		It("should reject invalid JSON", func() {
			_, err := ParseMealMatchDecisions("not-json")

			Expect(err).To(HaveOccurred())
		})

		It("should reject unknown response fields", func() {
			response := `{
				"decisions": [
					{"meal_index": 0, "decision": "NO_MATCH"}
				],
				"extra": true
			}`

			_, err := ParseMealMatchDecisions(response)

			Expect(err).To(MatchError(ContainSubstring("unknown field")))
		})

		It("should reject negative meal indexes", func() {
			response := `{
				"decisions": [
					{"meal_index": -1, "decision": "NO_MATCH"}
				]
			}`

			_, err := ParseMealMatchDecisions(response)

			Expect(err).To(MatchError(ContainSubstring("meal_index cannot be negative")))
		})

		It("should reject unknown decision values", func() {
			response := `{
				"decisions": [
					{"meal_index": 0, "decision": "MAYBE"}
				]
			}`

			_, err := ParseMealMatchDecisions(response)

			Expect(err).To(MatchError(ContainSubstring("unsupported decision")))
		})

		It("should reject MATCH without candidate ID", func() {
			response := `{
				"decisions": [
					{"meal_index": 0, "decision": "MATCH"}
				]
			}`

			_, err := ParseMealMatchDecisions(response)

			Expect(err).To(MatchError(ContainSubstring("MATCH requires candidate_id")))
		})

		It("should reject NO_MATCH with candidate ID", func() {
			response := `{
				"decisions": [
					{"meal_index": 0, "decision": "NO_MATCH", "candidate_id": "food-1"}
				]
			}`

			_, err := ParseMealMatchDecisions(response)

			Expect(err).To(MatchError(ContainSubstring("NO_MATCH cannot include candidate_id")))
		})
	})

	Describe("ResolveMatches", func() {
		It("should return no decisions without calling Gemini when no tasks exist", func() {
			client := Client{}

			decisions, err := client.ResolveMatches(nil, nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(decisions).To(BeEmpty())
		})
	})
})

// matchTask creates one ambiguous meal-matching task for prompt tests.
func matchTask(
	index int,
	generatedName string,
	candidates []interfaces.FoodMatchCandidate,
) interfaces.MealMatchTask {
	return interfaces.MealMatchTask{
		MealIndex: index,
		GeneratedMeal: interfaces.GeneratedMeal{
			Name: generatedName,
		},
		Candidates: candidates,
	}
}

// matchCandidate creates one backend-owned food candidate for matcher tests.
func matchCandidate(
	id string,
	name string,
	tags []string,
) interfaces.FoodMatchCandidate {
	return interfaces.FoodMatchCandidate{
		Food: interfaces.FoodSearchResult{
			ID:       id,
			Name:     name,
			Tags:     tags,
			Calories: 500,
			FatG:     15,
			ProteinG: 25,
			CarbsG:   60,
		},
		MatchKind:   interfaces.FoodMatchFuzzy,
		MatchedTerm: name,
		Score:       0.85,
	}
}
