package mealdataset

import (
	"context"
	"encoding/json"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"log/slog"
	"os"
	"strings"

	"github.com/google/uuid"
)

type PrebuiltMeal struct {
	ID                     string       `json:"id"`
	Name                   string       `json:"name"`
	AlternativeSearchTerms []string     `json:"alternative_search_terms"`
	Calories               float64      `json:"calories"`
	Protein                float64      `json:"protein"`
	Carbs                  float64      `json:"carbs"`
	Fat                    float64      `json:"fat"`
	Serving                string       `json:"serving"`
	Category               CategoryTags `json:"category"`
	ImageURL               string       `json:"image_url"`
}

type CategoryTags []string

func (c *CategoryTags) UnmarshalJSON(data []byte) error {
	var categories []string
	if err := json.Unmarshal(data, &categories); err == nil {
		*c = cleanCategoryTags(categories)
		return nil
	}

	var category string
	if err := json.Unmarshal(data, &category); err != nil {
		return fmt.Errorf("parse category as string or string array: %w", err)
	}

	*c = cleanCategoryTags([]string{category})
	return nil
}

func cleanCategoryTags(categories []string) []string {
	tags := make([]string, 0, len(categories))
	seen := make(map[string]bool)

	for _, category := range categories {
		trimmed := strings.TrimSpace(category)
		normalized := strings.ToLower(trimmed)
		if trimmed == "" || seen[normalized] {
			continue
		}

		seen[normalized] = true
		tags = append(tags, trimmed)
	}

	return tags
}

type PrebuiltSearcher struct {
	meals []PrebuiltMeal
}

var _ interfaces.FoodSearcher = (*PrebuiltSearcher)(nil)

func NewPrebuiltSearcher(datasetPath string) (*PrebuiltSearcher, error) {
	data, err := os.ReadFile(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("read prebuilt meal dataset: %w", err)
	}

	var meals []PrebuiltMeal
	if err := json.Unmarshal(data, &meals); err != nil {
		return nil, fmt.Errorf("parse prebuilt meal dataset: %w", err)
	}

	slog.Info("prebuilt meal dataset loaded",
		"dataset_path", datasetPath,
		"meal_count", len(meals),
	)

	return &PrebuiltSearcher{
		meals: meals,
	}, nil
}

func (s *PrebuiltSearcher) Search(ctx context.Context, query string) ([]PrebuiltMeal, error) {
	normalizedQuery := normalizeMealName(query)
	if normalizedQuery == "" {
		return nil, nil
	}

	var exactMatches []PrebuiltMeal
	var fuzzyMatches []PrebuiltMeal
	var partialMatches []PrebuiltMeal
	queryWordCount := mealNameWordCount(normalizedQuery)

	for _, meal := range s.meals {
		if prebuiltMealExactMatch(meal, normalizedQuery) {
			exactMatches = append(exactMatches, meal)
			continue
		}

		if prebuiltMealFuzzyMatch(meal, normalizedQuery) {
			fuzzyMatches = append(fuzzyMatches, meal)
			continue
		}

		if prebuiltMealPartialMatch(meal, normalizedQuery, queryWordCount) {
			partialMatches = append(partialMatches, meal)
		}
	}

	if len(exactMatches) > 0 {
		return exactMatches, nil
	}

	if len(fuzzyMatches) > 0 {
		return fuzzyMatches, nil
	}

	return partialMatches, nil
}

func prebuiltMealExactMatch(meal PrebuiltMeal, normalizedQuery string) bool {
	for _, candidate := range prebuiltMealSearchNames(meal) {
		if normalizeMealName(candidate) == normalizedQuery {
			return true
		}
	}

	return false
}

func prebuiltMealPartialMatch(meal PrebuiltMeal, normalizedQuery string, queryWordCount int) bool {
	for _, candidate := range prebuiltMealSearchNames(meal) {
		normalizedName := normalizeMealName(candidate)
		nameWordCount := mealNameWordCount(normalizedName)

		if isMealPhraseMatch(normalizedName, nameWordCount, normalizedQuery, queryWordCount) {
			return true
		}
	}

	return false
}

func prebuiltMealFuzzyMatch(meal PrebuiltMeal, normalizedQuery string) bool {
	for _, candidate := range prebuiltMealSearchNames(meal) {
		if FuzzyMealNameMatch(candidate, normalizedQuery) {
			return true
		}
	}

	return false
}

func prebuiltMealSearchNames(meal PrebuiltMeal) []string {
	names := make([]string, 0, 1+len(meal.AlternativeSearchTerms))
	names = append(names, meal.Name)
	names = append(names, meal.AlternativeSearchTerms...)

	return names
}

func isMealPhraseMatch(normalizedName string, nameWordCount int, normalizedQuery string, queryWordCount int) bool {
	if strings.Contains(normalizedName, normalizedQuery) && queryWordCount >= 2 {
		return true
	}

	if strings.Contains(normalizedQuery, normalizedName) && nameWordCount >= 2 {
		return true
	}

	return false
}

func mealNameWordCount(value string) int {
	if value == "" {
		return 0
	}

	return len(strings.Fields(value))
}

func normalizeMealName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ToLower(value)
	value = strings.Join(strings.Fields(value), " ")

	return value
}

func FuzzyMealNameMatch(candidate string, query string) bool {
	candidateWords := strings.Fields(normalizeMealName(candidate))
	queryWords := strings.Fields(normalizeMealName(query))

	if len(candidateWords) == 0 || len(queryWords) == 0 {
		return false
	}

	usedCandidateWords := make([]bool, len(candidateWords))
	for _, queryWord := range queryWords {
		matched := false

		for index, candidateWord := range candidateWords {
			if usedCandidateWords[index] {
				continue
			}

			if mealWordFuzzyMatch(candidateWord, queryWord) {
				usedCandidateWords[index] = true
				matched = true
				break
			}
		}

		if !matched {
			return false
		}
	}

	return true
}

func mealWordFuzzyMatch(candidateWord string, queryWord string) bool {
	if candidateWord == queryWord {
		return true
	}

	if len(queryWord) < 4 || len(candidateWord) < 4 {
		return false
	}

	distance := levenshteinDistance(candidateWord, queryWord)
	allowedDistance := 1
	if len(queryWord) >= 8 {
		allowedDistance = 2
	}

	return distance <= allowedDistance
}

func levenshteinDistance(a string, b string) int {
	if a == b {
		return 0
	}

	if len(a) == 0 {
		return len(b)
	}

	if len(b) == 0 {
		return len(a)
	}

	previous := make([]int, len(b)+1)
	current := make([]int, len(b)+1)

	for column := range previous {
		previous[column] = column
	}

	for row := 1; row <= len(a); row++ {
		current[0] = row

		for column := 1; column <= len(b); column++ {
			substitutionCost := 0
			if a[row-1] != b[column-1] {
				substitutionCost = 1
			}

			current[column] = minInt(
				current[column-1]+1,
				previous[column]+1,
				previous[column-1]+substitutionCost,
			)
		}

		previous, current = current, previous
	}

	return previous[len(b)]
}

func minInt(values ...int) int {
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}

	return minimum
}

func (s *PrebuiltSearcher) SearchFood(ctx context.Context, userID uuid.UUID, query string) (interfaces.FoodSearchResult, bool, error) {
	results, err := s.Search(ctx, query)
	if err != nil {
		return interfaces.FoodSearchResult{}, false, err
	}

	slog.Info("prebuilt meal search completed",
		"user_id", userID,
		"query", query,
		"match_count", len(results),
	)

	if len(results) == 0 {
		return interfaces.FoodSearchResult{}, false, nil
	}

	meal := results[0]

	return interfaces.FoodSearchResult{
		ID:       prebuiltMealID(meal),
		Name:     meal.Name,
		Tags:     []string(meal.Category),
		Calories: meal.Calories,
		FatG:     meal.Fat,
		ProteinG: meal.Protein,
		CarbsG:   meal.Carbs,
		ImageURL: meal.ImageURL,
	}, true, nil
}

func prebuiltMealID(meal PrebuiltMeal) string {
	return PrebuiltMealID(meal)
}

func PrebuiltMealID(meal PrebuiltMeal) string {
	id := strings.TrimSpace(meal.ID)
	if id != "" {
		return id
	}

	return normalizeMealName(meal.Name)
}
