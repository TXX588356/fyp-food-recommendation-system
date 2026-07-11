package recommendation

import (
	"fyp/food-rs/internal/interfaces"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recommendation matching helpers", func() {
	Describe("classifyCandidates", func() {
		It("should accept one exact canonical candidate locally", func() {
			candidate := matchCandidate("food-1", "Nasi Lemak", interfaces.FoodMatchExactName)

			got, ok := classifyCandidates([]interfaces.FoodMatchCandidate{candidate})

			Expect(ok).To(BeTrue())
			Expect(got.Food.ID).To(Equal("food-1"))
			Expect(got.MatchKind).To(Equal(interfaces.FoodMatchExactName))
		})

		It("should accept one exact alias candidate locally", func() {
			candidate := matchCandidate("food-1", "Wantan Mee", interfaces.FoodMatchExactAlias)

			got, ok := classifyCandidates([]interfaces.FoodMatchCandidate{candidate})

			Expect(ok).To(BeTrue())
			Expect(got.Food.ID).To(Equal("food-1"))
			Expect(got.MatchKind).To(Equal(interfaces.FoodMatchExactAlias))
		})

		It("should keep two exact candidates ambiguous", func() {
			candidates := []interfaces.FoodMatchCandidate{
				matchCandidate("food-1", "Chicken Rice", interfaces.FoodMatchExactName),
				matchCandidate("food-2", "Chicken Rice Bowl", interfaces.FoodMatchExactAlias),
			}

			got, ok := classifyCandidates(candidates)

			Expect(ok).To(BeFalse())
			Expect(got).To(Equal(interfaces.FoodMatchCandidate{}))
		})

		It("should keep fuzzy candidates ambiguous", func() {
			candidate := matchCandidate("food-1", "Chicken Breakfast Sausage", interfaces.FoodMatchFuzzy)

			got, ok := classifyCandidates([]interfaces.FoodMatchCandidate{candidate})

			Expect(ok).To(BeFalse())
			Expect(got).To(Equal(interfaces.FoodMatchCandidate{}))
		})

		It("should keep partial candidates ambiguous", func() {
			candidate := matchCandidate("food-1", "Fried Rice Chicken", interfaces.FoodMatchPartial)

			got, ok := classifyCandidates([]interfaces.FoodMatchCandidate{candidate})

			Expect(ok).To(BeFalse())
			Expect(got).To(Equal(interfaces.FoodMatchCandidate{}))
		})

		It("should return false when no candidates exist", func() {
			got, ok := classifyCandidates(nil)

			Expect(ok).To(BeFalse())
			Expect(got).To(Equal(interfaces.FoodMatchCandidate{}))
		})
	})

	Describe("buildMatchTask", func() {
		It("should preserve the original generated meal index", func() {
			meal := interfaces.GeneratedMeal{
				Name:                   "Chicken Sausage",
				AlternativeSearchTerms: []string{"Chicken Breakfast Sausage"},
			}
			candidates := []interfaces.FoodMatchCandidate{
				matchCandidate("food-1", "Chicken Breakfast Sausage", interfaces.FoodMatchFuzzy),
			}

			task := buildMatchTask(3, meal, candidates)

			Expect(task.MealIndex).To(Equal(3))
			Expect(task.GeneratedMeal.Name).To(Equal("Chicken Sausage"))
			Expect(task.Candidates).To(HaveLen(1))
			Expect(task.Candidates[0].Food.ID).To(Equal("food-1"))
		})
	})

	Describe("buildCandidateAllowlist", func() {
		It("should map candidate IDs under their own meal index", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Meal A"}, []interfaces.FoodMatchCandidate{
					matchCandidate("a-1", "Meal A Candidate", interfaces.FoodMatchFuzzy),
				}),
				buildMatchTask(1, interfaces.GeneratedMeal{Name: "Meal B"}, []interfaces.FoodMatchCandidate{
					matchCandidate("b-1", "Meal B Candidate", interfaces.FoodMatchFuzzy),
				}),
			}

			allowlist := buildCandidateAllowlist(tasks)

			Expect(allowlist).To(HaveLen(2))
			Expect(allowlist[0]).To(HaveKey("a-1"))
			Expect(allowlist[1]).To(HaveKey("b-1"))
			Expect(allowlist[0]).NotTo(HaveKey("b-1"))
		})

		It("should ignore candidates with empty IDs", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Meal A"}, []interfaces.FoodMatchCandidate{
					matchCandidate("", "Missing ID", interfaces.FoodMatchFuzzy),
					matchCandidate("a-1", "Meal A Candidate", interfaces.FoodMatchFuzzy),
				}),
			}

			allowlist := buildCandidateAllowlist(tasks)

			Expect(allowlist[0]).To(HaveLen(1))
			Expect(allowlist[0]).To(HaveKey("a-1"))
			Expect(allowlist[0]).NotTo(HaveKey(""))
		})
	})

	Describe("validateDecisions", func() {
		It("should accept a valid MATCH decision", func() {
			candidate := matchCandidate("food-1", "Chicken Breakfast Sausage", interfaces.FoodMatchFuzzy)
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Chicken Sausage"}, []interfaces.FoodMatchCandidate{
					candidate,
				}),
			}

			accepted := validateDecisions(tasks, []interfaces.MealMatchDecision{
				{
					MealIndex:   0,
					Decision:    matchDecisionMatch,
					CandidateID: "food-1",
				},
			})

			Expect(accepted).To(HaveLen(1))
			Expect(accepted[0].Food.ID).To(Equal("food-1"))
		})

		It("should treat NO_MATCH as valid without accepting a candidate", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Chicken Sausage"}, []interfaces.FoodMatchCandidate{
					matchCandidate("food-1", "Babyfood Chicken Sausage", interfaces.FoodMatchFuzzy),
				}),
			}

			accepted := validateDecisions(tasks, []interfaces.MealMatchDecision{
				{
					MealIndex: 0,
					Decision:  matchDecisionNoMatch,
				},
			})

			Expect(accepted).To(BeEmpty())
		})

		It("should reject an invented candidate ID", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Chicken Sausage"}, []interfaces.FoodMatchCandidate{
					matchCandidate("food-1", "Chicken Breakfast Sausage", interfaces.FoodMatchFuzzy),
				}),
			}

			accepted := validateDecisions(tasks, []interfaces.MealMatchDecision{
				{
					MealIndex:   0,
					Decision:    matchDecisionMatch,
					CandidateID: "invented-id",
				},
			})

			Expect(accepted).To(BeEmpty())
		})

		It("should reject a candidate ID selected under a different meal index", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Meal A"}, []interfaces.FoodMatchCandidate{
					matchCandidate("a-1", "Meal A Candidate", interfaces.FoodMatchFuzzy),
				}),
				buildMatchTask(1, interfaces.GeneratedMeal{Name: "Meal B"}, []interfaces.FoodMatchCandidate{
					matchCandidate("b-1", "Meal B Candidate", interfaces.FoodMatchFuzzy),
				}),
			}

			accepted := validateDecisions(tasks, []interfaces.MealMatchDecision{
				{
					MealIndex:   0,
					Decision:    matchDecisionMatch,
					CandidateID: "b-1",
				},
			})

			Expect(accepted).To(BeEmpty())
		})

		It("should reject an unknown meal index", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Meal A"}, []interfaces.FoodMatchCandidate{
					matchCandidate("a-1", "Meal A Candidate", interfaces.FoodMatchFuzzy),
				}),
			}

			accepted := validateDecisions(tasks, []interfaces.MealMatchDecision{
				{
					MealIndex:   99,
					Decision:    matchDecisionMatch,
					CandidateID: "a-1",
				},
			})

			Expect(accepted).To(BeEmpty())
		})

		It("should reject duplicate decisions for one meal index", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Meal A"}, []interfaces.FoodMatchCandidate{
					matchCandidate("a-1", "Meal A Candidate", interfaces.FoodMatchFuzzy),
				}),
			}

			accepted := validateDecisions(tasks, []interfaces.MealMatchDecision{
				{
					MealIndex:   0,
					Decision:    matchDecisionMatch,
					CandidateID: "a-1",
				},
				{
					MealIndex: 0,
					Decision:  matchDecisionNoMatch,
				},
			})

			Expect(accepted).To(BeEmpty())
		})

		It("should produce no candidate when a decision is missing", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Meal A"}, []interfaces.FoodMatchCandidate{
					matchCandidate("a-1", "Meal A Candidate", interfaces.FoodMatchFuzzy),
				}),
			}

			accepted := validateDecisions(tasks, nil)

			Expect(accepted).To(BeEmpty())
		})

		It("should reject unknown decision values", func() {
			tasks := []interfaces.MealMatchTask{
				buildMatchTask(0, interfaces.GeneratedMeal{Name: "Meal A"}, []interfaces.FoodMatchCandidate{
					matchCandidate("a-1", "Meal A Candidate", interfaces.FoodMatchFuzzy),
				}),
			}

			accepted := validateDecisions(tasks, []interfaces.MealMatchDecision{
				{
					MealIndex:   0,
					Decision:    "MAYBE",
					CandidateID: "a-1",
				},
			})

			Expect(accepted).To(BeEmpty())
		})
	})
})

// matchCandidate creates a recommendation matching candidate fixture.
//
// The helper keeps tests compact while still constructing the real
// FoodMatchCandidate shape used by production code.
func matchCandidate(
	id string,
	name string,
	kind interfaces.FoodMatchKind,
) interfaces.FoodMatchCandidate {
	return interfaces.FoodMatchCandidate{
		Food: interfaces.FoodSearchResult{
			ID:       id,
			Name:     name,
			Tags:     []string{"test_category"},
			Calories: 500,
			FatG:     15,
			ProteinG: 25,
			CarbsG:   60,
		},
		MatchKind:   kind,
		MatchedTerm: name,
		Score:       matchCandidateScore(kind),
	}
}

// matchCandidateScore returns a realistic default score for each match kind.
//
// Tests that need exact score behavior can still override the returned fixture.
func matchCandidateScore(kind interfaces.FoodMatchKind) float64 {
	switch kind {
	case interfaces.FoodMatchExactName:
		return 1.00
	case interfaces.FoodMatchExactAlias:
		return 0.98
	case interfaces.FoodMatchFuzzy:
		return 0.85
	case interfaces.FoodMatchPartial:
		return 0.70
	default:
		return 0
	}
}
