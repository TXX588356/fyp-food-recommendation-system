package recommendation

import "fyp/food-rs/internal/interfaces"

const (
	// matchDecisionMatch means the adjudicator selected one supplied candidate
	matchDecisionMatch = "MATCH"

	// matchDecisionNoMatch means the adjudicator intentionally selected no candidate
	matchDecisionNoMatch = "NO_MATCH"
)

// candidateAllowList stores backend-owned candidate records by generated meal
// index and candidate food ID.
//
// Gemini is allowed to return only IDs that exists in this allowlist. The final
// accepted candidate is hydrated from this map, not from Gemini output
type candidateAllowList map[int]map[string]interfaces.FoodMatchCandidate

// classifyCandidates accepts a candidate group locally only when there is
// exactly one exact-name or exact-alias candidate.
//
// This is the fast path. It avoids a second Gemini call for safe exact matches.
//
// Returns:
// - candidate, true: exactly one exact candidate exists
// - zero value, false: no exact candidate or multiple exact candidates exist
func classifyCandidates(candidates []interfaces.FoodMatchCandidate) (interfaces.FoodMatchCandidate, bool) {
	exact := make([]interfaces.FoodMatchCandidate, 0, len(candidates))

	// Collect only exact match kinds. Fuzzy and partial matches are still ambiguous
	// and must not be accepted locally.
	for _, candidate := range candidates {
		if candidate.MatchKind == interfaces.FoodMatchExactName ||
			candidate.MatchKind == interfaces.FoodMatchExactAlias {
			exact = append(exact, candidate)
		}
	}

	// A single exact match is safe enough to accept
	if len(exact) == 1 {
		return exact[0], true
	}

	// Zero exact matches or multiple exact matches are ambiguous
	return interfaces.FoodMatchCandidate{}, false
}

// buildMatchTask creates one adjudication task for an ambiguous generated meal.
//
// index must be the original index from Gemini's generated meal slice. Do not
// use the task slice index, because validation later relies on this stable
// original meal index.
func buildMatchTask(index int, meal interfaces.GeneratedMeal, candidates []interfaces.FoodMatchCandidate) interfaces.MealMatchTask {
	return interfaces.MealMatchTask{
		MealIndex:     index,
		GeneratedMeal: meal,
		Candidates:    candidates,
	}
}

// buildCandidateAllowlist builds the per-meal candidate ID allowlist used to
// validate adjudicator decisions.
//
// The result shape is:
//
//	meal_index -> candidate_id -> full backend-owned candidate
//
// If duplicate candidate IDs appear inside the same task, the first one is kept.
// Candidate search should already deduplicate, so duplicate IDs here are treated
// defensively.
func buildCandidateAllowlist(tasks []interfaces.MealMatchTask) candidateAllowList {
	allowList := make(candidateAllowList, len(tasks))

	for _, task := range tasks {
		// Each generated meal gets its own independent candidate ID map
		byID := make(map[string]interfaces.FoodMatchCandidate, len(task.Candidates))

		for _, candidate := range task.Candidates {
			candidateID := candidate.Food.ID
			if candidateID == "" {
				continue
			}

			// Keep the first candidate for this ID. Candidate search ranking has
			// already decided order before task construction.
			if _, exists := byID[candidateID]; !exists {
				byID[candidateID] = candidate
			}
		}

		allowList[task.MealIndex] = byID
	}

	return allowList
}

func validateDecisions(tasks []interfaces.MealMatchTask, decisions []interfaces.MealMatchDecision) map[int]interfaces.FoodMatchCandidate {
	allowList := buildCandidateAllowlist(tasks)

	// Track how many decisions were returned for each known meal index.
	// Any duplicate decision invalidates that meal index completely.
	decisionCountByIndex := make(map[int]int, len(decisions))
	for _, decision := range decisions {
		if _, knownIndex := allowList[decision.MealIndex]; !knownIndex {
			continue
		}

		decisionCountByIndex[decision.MealIndex]++
	}

	accepted := make(map[int]interfaces.FoodMatchCandidate)

	for _, decision := range decisions {
		candidatesByID, knownIndex := allowList[decision.MealIndex]
		if !knownIndex {
			// Ignore decisions for meal indexes the backend never sent
			continue
		}

		if decisionCountByIndex[decision.MealIndex] != 1 {
			// Duplicate decisions are ambigious. Reject the whole index
			delete(accepted, decision.MealIndex)
			continue
		}

		switch decision.Decision {
		case matchDecisionNoMatch:
			// NO_MATCH is valid and intentionally produces no candidate.
			continue
		case matchDecisionMatch:
			// MATCH must select one candidate from this meal's allowlist
			candidate, exists := candidatesByID[decision.CandidateID]
			if !exists {
				continue
			}

			accepted[decision.MealIndex] = candidate
		default:
			// Unknown decision values are ignored
			continue
		}
	}
	return accepted
}
