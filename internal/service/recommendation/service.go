package recommendation

import (
	"context"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type service struct {
	mealGenerator           interfaces.MealGenerator
	foodSearcher            interfaces.FoodSearcher
	catalogService          interfaces.CatalogService
	customMealAutocompleter interfaces.CustomMealAutocompleter
	mealLogRepository       interfaces.MealLogRepository
	now                     func() time.Time

	autoCreateLimiter chan struct{}
}

type generatedMealCreateResult struct {
	index     int
	mealName  string
	candidate interfaces.MatchedMealCandidate
	err       error
}

type generatedMealCreateInput struct {
	index int
	meal  interfaces.GeneratedMeal
}

type indexedMealCandidate struct {
	index     int
	candidate interfaces.MatchedMealCandidate
}

func NewService(mealGenerator interfaces.MealGenerator, foodSearcher interfaces.FoodSearcher, catalogService interfaces.CatalogService, customMealAutocompleter interfaces.CustomMealAutocompleter, mealLogRepository interfaces.MealLogRepository) interfaces.RecommendationService {
	return &service{
		mealGenerator:           mealGenerator,
		foodSearcher:            foodSearcher,
		catalogService:          catalogService,
		customMealAutocompleter: customMealAutocompleter,
		mealLogRepository:       mealLogRepository,
		autoCreateLimiter:       make(chan struct{}, 2),
		now:                     time.Now,
	}
}

// GenerateRecommendationResult requests meal suggestions from Gemini, enriches
// them with catalog data, filters unsuitable meals, and ranks the valid results.
func (s *service) GenerateRecommendationResult(ctx context.Context, userID uuid.UUID, input interfaces.MealPromptInput) (interfaces.RecommendationResult, error) {
	slog.Info("recommendation generation started",
		"user_id", userID,
		"meal_category", input.MealCategory,
		"goal", input.Goal,
	)

	slog.Info("calling Gemini meal generator",
		"user_id", userID,
		"meal_category", input.MealCategory,
	)

	started := time.Now()

	historyStartTime := time.Now()

	// Load history
	historyStart := s.now().AddDate(0, 0, -30)
	historyLogs, err := s.mealLogRepository.ListByUserAndRange(ctx, userID, historyStart, s.now().AddDate(0, 0, 1))
	if err != nil {
		return interfaces.RecommendationResult{}, fmt.Errorf("load recommendation history: %w", err)
	}
	slog.Info("recommendation timing", "stage", "load_history", "duration_ms", time.Since(historyStartTime).Milliseconds())
	input.History = buildMealHistoryContext(historyLogs, s.now())

	geminiStart := time.Now()

	// Call Gemini
	response, err := s.mealGenerator.GenerateMeals(ctx, input)
	if err != nil {
		slog.Error("Gemini meal generation failed",
			"user_id", userID,
			"meal_category", input.MealCategory,
			"error", err,
		)
		return interfaces.RecommendationResult{}, fmt.Errorf("generated meal candidates: %w", err)
	}

	slog.Info("recommendation timing", "stage", "gemini_generate_meals", "duration_ms", time.Since(geminiStart).Milliseconds(), "generated_count", len(response.Meals))

	slog.Info("Gemini meal generation completed",
		"user_id", userID,
		"meal_category", input.MealCategory,
		"generated_count", len(response.Meals),
	)

	candidates := make([]indexedMealCandidate, 0, len(response.Meals))
	missingMeals := make([]generatedMealCreateInput, 0)

	for index, meal := range response.Meals {
		matchStart := time.Now()

		slog.Info("matching generated meal",
			"user_id", userID,
			"index", index,
			"meal_name", meal.Name,
			"alternative_terms_count", len(meal.AlternativeSearchTerms),
		)

		// Match Gemini response with existing meals
		candidate, found, err := s.matchFood(ctx, userID, meal)
		if err != nil {
			slog.Error("generated meal matching failed",
				"user_id", userID,
				"meal_name", meal.Name,
				"error", err,
			)
			return interfaces.RecommendationResult{}, err
		}

		slog.Info("recommendation timing", "stage", "match_food", "index", index, "meal_name", meal.Name, "found", found, "duration_ms", time.Since(matchStart).Milliseconds())

		if !found {
			slog.Info("generated meal unmatched; attempting prebuilt catalog auto-create",
				"user_id", userID,
				"meal_name", meal.Name,
			)

			missingMeals = append(missingMeals, generatedMealCreateInput{
				index: index,
				meal:  meal,
			})
			continue
		}

		candidates = append(candidates, indexedMealCandidate{
			index:     index,
			candidate: candidate,
		})
		slog.Info("generated meal matched",
			"user_id", userID,
			"meal_name", meal.Name,
			"matched_query", candidate.MatchedQuery,
			"matched_food", candidate.Food.Name,
		)
	}

	// concurrently create missing generated meals
	createResults := s.createMissingGeneratedCatalogMealsConcurrently(ctx, userID, missingMeals)
	for _, result := range createResults {
		if result.err != nil {
			slog.Error("generated meal auto-create failed",
				"user_id", userID,
				"index", result.index,
				"meal_name", result.mealName,
				"error", result.err,
			)
			continue
		}

		candidates = append(candidates, indexedMealCandidate{
			index:     result.index,
			candidate: result.candidate,
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].index < candidates[j].index
	})
	matchedCandidates := make([]interfaces.MatchedMealCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		matchedCandidates = append(matchedCandidates, candidate.candidate)
	}

	filterStart := time.Now()

	// filter
	filterResult := filterCandidates(matchedCandidates, input)
	slog.Info("recommendation timing", "stage", "filter_candidates", "candidate_count", len(matchedCandidates), "duration_ms", time.Since(filterStart).Milliseconds())

	rankStart := time.Now()

	// rank
	rankedCandidates := rankCandidates(filterResult.Filtered, input, input.History)
	slog.Info("recommendation timing", "stage", "rank_candidates", "candidate_count", len(filterResult.Filtered), "duration_ms", time.Since(rankStart).Milliseconds())
	filteredOut := make([]interfaces.FilteredMealCandidate, 0, len(filterResult.Removed))
	for _, removed := range filterResult.Removed {
		filteredOut = append(filteredOut, interfaces.FilteredMealCandidate{
			Candidate: removed.Candidate,
			Reason:    removed.Reason,
		})
	}

	slog.Info("recommendation generation completed",
		"user_id", userID,
		"meal_category", input.MealCategory,
		"candidate_count", len(matchedCandidates),
	)

	slog.Info("recommendation timing", "stage", "total", "duration_ms", time.Since(started).Milliseconds())

	return interfaces.RecommendationResult{
		Candidates:       rankedCandidates,
		FilteredOut:      filteredOut,
		FilteringApplied: true,
	}, nil
}

func (s *service) createMissingGeneratedCatalogMeal(ctx context.Context, userID uuid.UUID, meal interfaces.GeneratedMeal) (interfaces.MatchedMealCandidate, error) {
	details, err := s.customMealAutocompleter.AutocompleteCustomMeal(ctx, interfaces.CustomMealAutocompleteInput{
		Name: meal.Name,
	})

	if err != nil {
		return interfaces.MatchedMealCandidate{}, fmt.Errorf("autocomplete generated catalog meal %q: %w", meal.Name, err)
	}

	catalogMeal, err := s.catalogService.CreateGeneratedMeal(ctx, interfaces.GeneratedCatalogMealInput{
		Name:               meal.Name,
		CategoryCodes:      details.MealCategoryTags,
		ServingDescription: "1 serving",
		Calories:           details.Calories,
		ProteinG:           details.ProteinG,
		CarbsG:             details.CarbsG,
		FatG:               details.FatG,
	})

	if err != nil {
		return interfaces.MatchedMealCandidate{}, fmt.Errorf("create generated catalog meal %q: %w", meal.Name, err)
	}

	return interfaces.MatchedMealCandidate{
		GeneratedMeal: meal,
		MatchedQuery:  meal.Name,
		Food:          catalogMealToFoodSearchResult(catalogMeal),
	}, nil
}

// matchFood tries the normalized Gemini name first, followed by each fallback search term.
// A meal is discarded if every exact lookup returns no match.
func (s *service) matchFood(ctx context.Context, userID uuid.UUID, meal interfaces.GeneratedMeal) (interfaces.MatchedMealCandidate, bool, error) {
	searchTerms := buildSearchTerms(meal)

	for _, query := range searchTerms {
		slog.Info("searching meal candidate",
			"user_id", userID,
			"generated_meal", meal.Name,
			"query", query,
		)

		food, found, err := s.foodSearcher.SearchFood(ctx, userID, query)
		if err != nil {
			return interfaces.MatchedMealCandidate{}, false, fmt.Errorf("search food %q: %w", query, err)
		}

		if found {
			return interfaces.MatchedMealCandidate{
				GeneratedMeal: meal,
				Food:          food,
				MatchedQuery:  query,
			}, true, nil
		}
	}

	return interfaces.MatchedMealCandidate{}, false, nil
}

func catalogMealToFoodSearchResult(meal interfaces.CatalogMeal) interfaces.FoodSearchResult {
	result := interfaces.FoodSearchResult{
		ID:   meal.ID.String(),
		Name: meal.Name,
		Tags: meal.Categories,
	}
	if meal.SelectedNutrition.Calories != nil {
		result.Calories = *meal.SelectedNutrition.Calories
	}
	if meal.SelectedNutrition.FatG != nil {
		result.FatG = *meal.SelectedNutrition.FatG
	}
	if meal.SelectedNutrition.ProteinG != nil {
		result.ProteinG = *meal.SelectedNutrition.ProteinG
	}
	if meal.SelectedNutrition.CarbsG != nil {
		result.CarbsG = *meal.SelectedNutrition.CarbsG
	}
	if meal.Image != nil {
		result.ImageURL = meal.Image.URL
	}

	return result
}

func (s *service) createMissingGeneratedCatalogMealsConcurrently(ctx context.Context, userID uuid.UUID, meals []generatedMealCreateInput) []generatedMealCreateResult {
	results := make([]generatedMealCreateResult, len(meals))
	var wg sync.WaitGroup

	for resultIndex, input := range meals {
		resultIndex := resultIndex
		input := input

		wg.Add(1)
		go func() {
			defer wg.Done()

			select {
			case s.autoCreateLimiter <- struct{}{}:
				defer func() { <-s.autoCreateLimiter }()
			case <-ctx.Done():
				results[resultIndex] = generatedMealCreateResult{
					index:    input.index,
					mealName: input.meal.Name,
					err:      ctx.Err(),
				}
				return
			}

			createStart := time.Now()
			candidate, err := s.createMissingGeneratedCatalogMeal(ctx, userID, input.meal)
			slog.Info("recommendation timing",
				"stage", "auto_create_generated_meal",
				"index", input.index,
				"meal_name", input.meal.Name,
				"duration_ms", time.Since(createStart).Milliseconds(),
			)

			results[resultIndex] = generatedMealCreateResult{
				index:     input.index,
				mealName:  input.meal.Name,
				candidate: candidate,
				err:       err,
			}
		}()
	}

	wg.Wait()
	return results
}

// buildSearchTerms returns unique non-empty lookup terms in priority order.
func buildSearchTerms(meal interfaces.GeneratedMeal) []string {
	searchTerms := make([]string, 0, 1+len(meal.AlternativeSearchTerms))

	seen := make(map[string]bool)

	addTerm := func(term string) {
		trimmed := strings.TrimSpace(term)
		normalized := strings.ToLower(trimmed)

		if trimmed == "" || seen[normalized] {
			return
		}

		seen[normalized] = true
		searchTerms = append(searchTerms, trimmed)
	}

	addTerm(meal.Name)

	for _, term := range meal.AlternativeSearchTerms {
		addTerm(term)
	}

	return searchTerms
}
