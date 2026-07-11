package catalog

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/service/mealsearch"
	"strings"

	"github.com/google/uuid"
)

// CandidateSearcher adapts CatalogService.SearchMeals into the ranked candidate
// interface used by recommendation matching.
//
// This is different from FoodSearcher in food_searcher.go:
//   - FoodSearcher returns only one accepted result.
//   - CandidateSearcher returns multiple ranked candidates so the recommendation
//     flow can detect ambiguity and send candidates to Gemini adjudication.
type CandidateSearcher struct {
	service interfaces.CatalogService
}

var _ interfaces.FoodCandidateSearcher = (*CandidateSearcher)(nil)

// The final recommendation candidate searcher will combine:
// - visible custom meals
// - catalog meals from this searcher
func NewCandidateSearcher(service interfaces.CatalogService) interfaces.FoodCandidateSearcher {
	return &CandidateSearcher{
		service: service,
	}
}

// SearchFoodCandidates returns ranked catalog candidates for generated meal
// terms.
//
// userID is accepted to satisfy the shared FoodCandidateSearcher interface.
// Catalog meals are global, so this implementation does not use userID for
// filtering.
//
// The method searches each generated meal term with CatalogService.SearchMeals,
// ranks every returned catalog meal using mealsearch.RankFoodName, deduplicates
// repeated catalog IDs, sorts by ranking strength, and returns at most limit
// candidates.
func (s *CandidateSearcher) SearchFoodCandidates(ctx context.Context, userID uuid.UUID, queries []string, limit int) ([]interfaces.FoodMatchCandidate, error) {
	if limit <= 0 {
		return nil, nil
	}

	// Store only the best candidate for each catalog food ID.
	// The same food may appear from multiple generated search terms.
	bestByID := make(map[string]interfaces.FoodMatchCandidate)

	// Search each unique query once.
	// Example:
	//   ["Nasi Lemak", " nasi   lemak ", "NASI LEMAK"]
	// becomes one catalog search.
	for _, query := range uniqueCandidateQueries(queries) {
		// Ask the existing catalog service for several possible matches.
		// This is the key difference from FoodSearcher, which uses Limit: 1.
		page, err := s.service.SearchMeals(ctx, interfaces.CatalogQuery{
			Query: query,
			Limit: limit,
		})

		if err != nil {
			return nil, err
		}

		// Convert every returned catalog meal into a recommendation candidate.
		for _, meal := range page.Items {
			// Skip meals that cannot produce complate FoodSearchResult nutrition.
			food, ok := catalogMealToCandidateFood(meal)
			if !ok {
				continue
			}

			match, found := mealsearch.RankFoodName(meal.Name, nil, queries)
			if !found {
				continue
			}

			candidate := interfaces.FoodMatchCandidate{
				Food:        food,
				MatchKind:   match.Kind,
				MatchedTerm: match.MatchedTerm,
				Score:       match.Score,
			}

			existing, exists := bestByID[food.ID]
			if !exists || mealsearch.CompareFoodMatchCandidates(candidate, existing) < 0 {
				bestByID[food.ID] = candidate
			}
		}
	}

	// Convert the dedupe map back into a slice for sorting and limiting.
	candidates := make([]interfaces.FoodMatchCandidate, 0, len(bestByID))
	for _, candidate := range bestByID {
		candidates = append(candidates, candidate)
	}

	// Sort once after dedupe so the final list is deterministic.
	mealsearch.SortFoodMatchCandidates(candidates)

	// Enforce the caller's requested limit.
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

// catalogMealToCandidateFood converts a CatalogMeal into FoodSearchResult.
//
// It intentionally follows the same macro completeness rule as FoodSearcher:
// the recommendation matcher should only return backend records with complete
// calories/protein/carbs/fat values. Meals missing required macros are skipped.
func catalogMealToCandidateFood(meal interfaces.CatalogMeal) (interfaces.FoodSearchResult, bool) {
	nutrition := meal.SelectedNutrition

	// Required macro values must all be present.
	// If any pointer is nil, skip the meal instead of returning partial data.
	if nutrition.Calories == nil ||
		nutrition.ProteinG == nil ||
		nutrition.CarbsG == nil ||
		nutrition.FatG == nil {
		return interfaces.FoodSearchResult{}, false
	}

	// Image is optional.
	imageURL := ""
	if meal.Image != nil {
		imageURL = meal.Image.URL
	}

	return interfaces.FoodSearchResult{
		ID:       meal.ID.String(),
		Name:     meal.Name,
		Tags:     meal.Categories,
		Calories: *nutrition.Calories,
		ProteinG: *nutrition.ProteinG,
		CarbsG:   *nutrition.CarbsG,
		FatG:     *nutrition.FatG,
		ImageURL: imageURL,
	}, true
}

// uniqueCandidateQueries removes empty and duplicate generated meal search
// terms while preserving the first spelling of each term.
//
// This prevents repeated calls to CatalogService.SearchMeals for equivalent
// terms such as "Nasi Lemak", " nasi   lemak ", and "NASI LEMAK".
func uniqueCandidateQueries(queries []string) []string {
	unique := make([]string, 0, len(queries))
	seen := make(map[string]bool)

	for _, query := range queries {
		// Normalize only for deduplication.
		// Keep the original query string in the returned slice so SearchMeals
		// receives a human-readable query.
		normalized := normalizeCandidateQuery(query)

		if normalized == "" || seen[normalized] {
			continue
		}

		seen[normalized] = true
		unique = append(unique, query)
	}

	return unique
}

// normalizeCandidateQuery lowercases and collapses whitespace for query
// deduplication.
func normalizeCandidateQuery(query string) string {
	// Trim leading/trailing whitespace, lowercase, split on any whitespace, then
	// join with a single space.
	fields := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	return strings.Join(fields, " ")
}
