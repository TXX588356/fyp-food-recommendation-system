package recommendation

import (
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"sort"
	"strings"
	"time"
)

func buildMealHistoryContext(logs []model.MealLog, now time.Time) interfaces.MealHistoryContext {
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].EatenAt.After(logs[j].EatenAt)
	})

	recentStart := now.AddDate(0, 0, -30)
	fatigueStart := now.AddDate(0, 0, -7)
	learnedStart := now.AddDate(0, 0, -90)
	recentNames := make([]string, 0, len(logs))
	categoryCounts := map[string]int{}
	fatiguedCategoryCounts := map[string]int{}
	learnedCategoryCounts := map[string]int{}
	nameCounts := map[string]int{}
	recentlyEatenByName := map[string]int{}

	for _, log := range logs {
		if log.EatenAt.Before(learnedStart) {
			continue
		}
		for _, category := range log.MealCategory {
			if shouldUseCategoryForRecency(category) {
				learnedCategoryCounts[category]++
			}
		}

		if log.EatenAt.Before(recentStart) {
			continue
		}

		recentNames = appendUniqueLimited(recentNames, log.MealName, 10)
		normalizedName := normalizeHistoryName(log.MealName)
		nameCounts[normalizedName]++
		if _, exists := recentlyEatenByName[normalizedName]; !exists {
			recentlyEatenByName[normalizedName] = daysBetween(log.EatenAt, now)
		}
		for _, category := range log.MealCategory {
			if shouldUseCategoryForRecency(category) {
				categoryCounts[category]++
				if !log.EatenAt.Before(fatigueStart) {
					fatiguedCategoryCounts[category]++
				}
			}
		}
	}

	repeated := make([]string, 0)
	for _, name := range recentNames {
		if nameCounts[normalizeHistoryName(name)] > 1 {
			repeated = append(repeated, name)
		}
	}

	return interfaces.MealHistoryContext{
		RecentMealNames:        recentNames,
		RecentCategoryCounts:   categoryCounts,
		RepeatedMealNames:      repeated,
		RecentlyEatenByName:    recentlyEatenByName,
		LearnedCategoryCounts:  learnedCategoryCounts,
		FatiguedCategoryCounts: fatiguedCategoryCounts,
	}
}

func daysBetween(past time.Time, now time.Time) int {
	pastDay := time.Date(past.Year(), past.Month(), past.Day(), 0, 0, 0, 0, past.Location())
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	days := int(nowDay.Sub(pastDay).Hours() / 24)
	if days < 0 {
		return 0
	}

	return days
}

func appendUniqueLimited(values []string, value string, limit int) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if strings.EqualFold(existing, value) {
			return values
		}
	}
	if len(values) >= limit {
		return values
	}
	return append(values, value)
}

func normalizeHistoryName(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}
