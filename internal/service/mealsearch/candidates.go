package mealsearch

import (
	"context"
	"fmt"

	"fyp/food-rs/internal/interfaces"

	"github.com/google/uuid"
)

// CandidateSearcher merges visible custom meals with catalog candidates for the
// recommendation matcher.
//
// This intentionally does not replace CombinedSearcher:
//   - CombinedSearcher is still for single-result manual/simple search.
//   - CandidateSearcher is for recommendations, where we need multiple ranked
//     candidates so ambiguous matches can be adjudicated safely.
type CandidateSearcher struct {
	customMealService interfaces.CustomMealService
	catalogSearcher   interfaces.FoodCandidateSearcher
}

var _ interfaces.FoodCandidateSearcher = (*CandidateSearcher)(nil)

// NewCandidateSearcher creates the recommendation candidate aggregator.
//
// Use this from app.GetRecommendationService with:
// - the existing CustomMealService
// - the catalog candidate searcher from internal/service/catalog
func NewCandidateSearcher(
	customMealService interfaces.CustomMealService,
	catalogSearcher interfaces.FoodCandidateSearcher,
) interfaces.FoodCandidateSearcher {
	return &CandidateSearcher{
		customMealService: customMealService,
		catalogSearcher:   catalogSearcher,
	}
}

// SearchFoodCandidates searches all unique generated terms against custom meals
// and catalog meals, deduplicates the combined records, and returns the top
// ranked candidates for adjudication.
//
// The recommendation service should call this once per generated meal with that
// meal's name and alternative search terms.
func (s *CandidateSearcher) SearchFoodCandidates(
	ctx context.Context,
	userID uuid.UUID,
	queries []string,
	limit int,
) ([]interfaces.FoodMatchCandidate, error) {
	// A non-positive limit means the caller does not want any results.
	// Returning early also avoids unnecessary custom/catalog queries.
	if limit <= 0 {
		return nil, nil
	}

	// bestByKey stores the best candidate for each source-specific identity.
	//
	// We namespace keys because custom meals and catalog meals may both have the
	// same public string ID format in the future. Namespacing prevents accidental
	// cross-source deduplication.
	bestByKey := make(map[string]interfaces.FoodMatchCandidate)

	// Search custom meals once per unique generated search term.
	//
	// Example:
	//   ["Nasi Lemak", " nasi   lemak ", "NASI LEMAK"]
	// should call ListVisible only once.
	for _, query := range uniqueNonEmptyTerms(queries) {
		meals, err := s.customMealService.ListVisible(ctx, userID, query)
		if err != nil {
			return nil, fmt.Errorf("search custom meal candidates %q: %w", query, err)
		}

		for _, meal := range meals {
			if meal == nil {
				continue
			}

			// Convert the custom meal API shape into the shared food-search shape.
			food := customMealToCandidateFood(meal)

			// Rank the custom meal against all generated terms, not only the term
			// that produced this ListVisible result.
			//
			// This lets the strongest generated term decide the candidate's final
			// MatchKind/MatchedTerm/Score.
			match, found := RankFoodName(meal.Name, nil, queries)
			if !found {
				continue
			}

			candidate := interfaces.FoodMatchCandidate{
				Food:        food,
				MatchKind:   match.Kind,
				MatchedTerm: match.MatchedTerm,
				Score:       match.Score,
			}

			// Custom meal IDs are deduped within the custom source only.
			upsertBestCandidate(bestByKey, "custom:"+meal.ID, candidate)
		}
	}

	// Ask the catalog candidate searcher for ranked catalog candidates.
	//
	// The catalog searcher already handles:
	// - querying CatalogService.SearchMeals
	// - skipping incomplete nutrition
	// - ranking catalog meals
	// - deduplicating catalog meal IDs
	catalogCandidates, err := s.catalogSearcher.SearchFoodCandidates(ctx, userID, queries, limit)
	if err != nil {
		return nil, fmt.Errorf("search catalog meal candidates: %w", err)
	}

	// Merge catalog candidates into the same dedupe map using catalog namespace.
	for _, candidate := range catalogCandidates {
		if candidate.Food.ID == "" {
			continue
		}

		upsertBestCandidate(bestByKey, "catalog:"+candidate.Food.ID, candidate)
	}

	// Convert the dedupe map into a slice for deterministic sorting.
	candidates := make([]interfaces.FoodMatchCandidate, 0, len(bestByKey))
	for _, candidate := range bestByKey {
		candidates = append(candidates, candidate)
	}

	// Sort by score, match kind, normalized name, then stable ID.
	SortFoodMatchCandidates(candidates)

	// Enforce the requested candidate limit after custom and catalog candidates
	// have been merged into one ranked list.
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return candidates, nil
}

// customMealToCandidateFood converts a visible custom meal into the shared
// FoodSearchResult shape used by recommendation matching.
//
// This mirrors CombinedSearcher.searchCustomMeals so custom meal behavior stays
// consistent between single-result search and candidate search.
func customMealToCandidateFood(meal *interfaces.CustomMealResponse) interfaces.FoodSearchResult {
	return interfaces.FoodSearchResult{
		ID:       meal.ID,
		Name:     meal.Name,
		Tags:     meal.MealCategoryTags,
		Calories: meal.Calories,
		FatG:     meal.FatG,
		ProteinG: meal.ProteinG,
		CarbsG:   meal.CarbsG,
		ImageURL: meal.ImageURL,
	}
}

// upsertBestCandidate stores candidate under key when no candidate exists yet,
// or replaces the existing candidate when the new one ranks higher.
//
// The comparison uses the same deterministic ordering as final sorting.
func upsertBestCandidate(
	bestByKey map[string]interfaces.FoodMatchCandidate,
	key string,
	candidate interfaces.FoodMatchCandidate,
) {
	existing, exists := bestByKey[key]
	if !exists {
		bestByKey[key] = candidate
		return
	}

	// CompareFoodMatchCandidates returns -1 when candidate should sort before
	// existing, meaning candidate is the stronger result.
	if CompareFoodMatchCandidates(candidate, existing) < 0 {
		bestByKey[key] = candidate
	}
}
