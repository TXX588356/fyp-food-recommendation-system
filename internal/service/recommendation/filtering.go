package recommendation

import "fyp/food-rs/internal/interfaces"

type FilteredOutCandidate struct {
	Candidate interfaces.MatchedMealCandidate
	Reason    string
}

type FilterResult struct {
	Filtered []interfaces.MatchedMealCandidate
	Removed  []FilteredOutCandidate
}

func filterCandidates(candidates []interfaces.MatchedMealCandidate, input interfaces.MealPromptInput) FilterResult {
	result := FilterResult{
		Filtered: make([]interfaces.MatchedMealCandidate, 0, len(candidates)),
		Removed:  make([]FilteredOutCandidate, 0)}

	for _, candidate := range candidates {
		if reason := dietaryConflictReason(candidate, input.DietaryRestrictions); reason != "" {
			result.Removed = append(result.Removed, FilteredOutCandidate{Candidate: candidate, Reason: reason})
			continue
		}
		if reason := healthConflictReason(candidate, input.HealthConcerns); reason != "" {
			result.Removed = append(result.Removed, FilteredOutCandidate{Candidate: candidate, Reason: reason})
			continue
		}
		result.Filtered = append(result.Filtered, candidate)
	}
	return result
}

func dietaryConflictReason(candidate interfaces.MatchedMealCandidate, restrictions []string) string {
	tags := map[string]bool{}
	for _, tag := range candidate.Food.Tags {
		tags[tag] = true
	}

	for _, restriction := range restrictions {
		switch restriction {
		case "halal":
			if tags["pork"] {
				return "contains pork, not suitable for halal restriction"
			}
		case "non_beef":
			if tags["beef"] {
				return "contains beef"
			}
		case "vegetarian", "vegan":
			if tags["beef"] || tags["pork"] || tags["poultry"] || tags["seafood"] || tags["lamb"] {
				return "contains animal protein"
			}
		case "seafood_free":
			if tags["seafood"] {
				return "contains seafood"
			}
		case "nut_free":
			if tags["nuts_seeds"] {
				return "contains nuts or seeds"
			}
		}
	}
	return ""
}

func healthConflictReason(candidate interfaces.MatchedMealCandidate, concerns []string) string {
	for _, concern := range concerns {
		if candidate.GeneratedMeal.HealthFlags[concern] == "AVOID" {
			return "AVOID for " + concern
		}
	}
	return ""
}
