package recommendation

import (
	"fyp/food-rs/internal/interfaces"
	"math"
	"sort"
)

type CandidateScore struct {
	GoalScore           float64
	BudgetScore         float64
	RecencyPenaltyScore float64
	PreferenceScore     float64
	Total               float64
}

func rankCandidates(candidates []interfaces.MatchedMealCandidate, input interfaces.MealPromptInput, history interfaces.MealHistoryContext) []interfaces.MatchedMealCandidate {
	scored := make([]struct {
		candidate interfaces.MatchedMealCandidate
		score     float64
		index     int
	}, len(candidates))

	for index, candidate := range candidates {
		if candidate.Score == 0 {
			applyCandidateScore(&candidate, scoreCandidate(candidate, input, history))
		}
		scored[index].candidate = candidate
		scored[index].score = candidate.Score
		scored[index].index = index
	}

	// Rank by score in descending order
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score == scored[j].score {
			return scored[i].index < scored[j].index
		}
		return scored[i].score > scored[j].score
	})

	result := make([]interfaces.MatchedMealCandidate, 0, len(candidates))
	for _, item := range scored {
		result = append(result, item.candidate)
	}

	return result
}

func applyCandidateScore(candidate *interfaces.MatchedMealCandidate, score CandidateScore) {
	candidate.Score = score.Total
	candidate.ScoreBreakdown = interfaces.CandidateScoreBreakdown{
		GoalAlignment:  score.GoalScore,
		BudgetFit:      score.BudgetScore,
		RecencyPenalty: score.RecencyPenaltyScore,
		Preference:     score.PreferenceScore,
	}
}

func scoreCandidate(candidate interfaces.MatchedMealCandidate, input interfaces.MealPromptInput, history interfaces.MealHistoryContext) CandidateScore {
	goal := goalAlignmentScore(candidate.Food, input.Goal)
	budget := budgetFitScore(
		candidate.GeneratedMeal.EstimatedPriceRange.Min,
		candidate.GeneratedMeal.EstimatedPriceRange.Max,
		input.PerMealBudget,
	)
	recencyPenalty := recencyPenaltyScore(candidate.Food.Name, history)
	preference := preferenceScore(candidate.Food.Tags, input.PreferredMealTags)

	return CandidateScore{
		GoalScore:           goal,
		BudgetScore:         budget,
		RecencyPenaltyScore: recencyPenalty,
		PreferenceScore:     preference,
		Total:               goal + budget + recencyPenalty + preference,
	}
}

func goalAlignmentScore(food interfaces.FoodSearchResult, goal string) float64 {
	fat := math.Max(0, 1-(food.FatG/30))
	protein := math.Min(food.ProteinG/20, 1)
	carbs := math.Max(0, 1-(food.CarbsG/80))

	// Socring rules for different goals
	switch goal {
	case "muscle_gain":
		return protein*20 + fat*5 + carbs*5
	case "quick_recommendation":
		return 25
	default:
		return fat*10 + protein*12 + carbs*8
	}
}

func budgetFitScore(minPrice, maxPrice, perMealBudget float64) float64 {
	// No valid per-meal budget
	if perMealBudget <= 0 {
		return 20
	}

	price := expectedBudgetPrice(minPrice, maxPrice)
	if price <= 0 {
		return 20
	}

	ratio := price / perMealBudget
	if ratio <= 1 {
		return 20
	}
	if ratio <= 1.25 {
		return 20 - ((ratio - 1) / 0.25 * 12)
	}
	if ratio <= 1.5 {
		return 8 * (1 - ((ratio - 1.25) / 0.25))
	}
	return 0
}

func expectedBudgetPrice(minPrice, maxPrice float64) float64 {
	if maxPrice <= 0 {
		return minPrice
	}

	if minPrice <= 0 || minPrice > maxPrice {
		return maxPrice
	}

	midpoint := (minPrice + maxPrice) / 2
	return maxPrice*0.7 + midpoint*0.3
}

func recencyPenaltyScore(name string, history interfaces.MealHistoryContext) float64 {
	daysAgo, found := history.RecentlyEatenByName[normalizeHistoryName(name)]
	if !found {
		return 20
	}
	switch {
	case daysAgo <= 1:
		return 0
	case daysAgo == 2:
		return 8
	case daysAgo == 3:
		return 14
	default:
		return 20
	}
}

func preferenceScore(tags, preferredTags []string) float64 {
	if len(preferredTags) == 0 {
		return 30
	}
	tagSet := map[string]bool{}
	for _, tag := range tags {
		tagSet[tag] = true
	}

	matches := 0
	for _, preferred := range preferredTags {
		if tagSet[preferred] {
			matches++
		}
	}
	return float64(matches) / float64(len(preferredTags)) * 30
}
