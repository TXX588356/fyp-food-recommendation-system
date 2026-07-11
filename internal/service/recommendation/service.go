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
	mealGenerator interfaces.MealGenerator

	// candidateSearcher returns up to N ranked backend-owned candidates for one
	// generated meal.
	candidateSearcher interfaces.FoodCandidateSearcher

	// matchAdjudicator resolves ambiguous candidate groups in one batched LLM
	// call. Exact match skips this dependency.
	matchAdjudicator        interfaces.MealMatchAdjudicator
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

func NewService(mealGenerator interfaces.MealGenerator, candidateSearcher interfaces.FoodCandidateSearcher, matchAdjudicator interfaces.MealMatchAdjudicator, catalogService interfaces.CatalogService, customMealAutocompleter interfaces.CustomMealAutocompleter, mealLogRepository interfaces.MealLogRepository) interfaces.RecommendationService {
	return &service{
		mealGenerator:           mealGenerator,
		candidateSearcher:       candidateSearcher,
		matchAdjudicator:        matchAdjudicator,
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

	// Ambiguous meals are collected first, then resolved in one LLM call.
	matchTasks := make([]interfaces.MealMatchTask, 0)

	for index, meal := range response.Meals {
		matchStart := time.Now()

		slog.Info("matching generated meal",
			"user_id", userID,
			"index", index,
			"meal_name", meal.Name,
			"alternative_terms_count", len(meal.AlternativeSearchTerms),
		)

		// Build the same unique search terms: generated name first, then
		// alternative_search_terms.
		searchTerms := buildSearchTerms(meal)

		// Ask the new candidate searcher for several ranked options instead of
		// one first result. This is the core fuzzy-search upgrade.
		foodCandidates, err := s.candidateSearcher.SearchFoodCandidates(ctx, userID, searchTerms, 5)
		if err != nil {
			slog.Error("generated meal candidate search failed",
				"user_id", userID,
				"index", index,
				"meal_name", meal.Name,
				"error", err,
			)
			return interfaces.RecommendationResult{}, fmt.Errorf("search generated meal candidates %q: %w", meal.Name, err)
		}

		slog.Info("recommendation_timing",
			"stage", "search_food_candidates",
			"index", index,
			"meal_name", meal.Name,
			"candidate_count", len(foodCandidates),
			"duration_ms", time.Since(matchStart).Milliseconds(),
		)

		// If no backend candidates exist, keep existing behavior: auto-create a
		// generated catalog meal later.
		if len(foodCandidates) == 0 {
			slog.Info("generated meal has no candidates; attempting generated catalog auto-create",
				"user_id", userID,
				"index", index,
				"meal_name", meal.Name,
			)

			missingMeals = append(missingMeals, generatedMealCreateInput{
				index: index,
				meal:  meal,
			})
			continue
		}

		// Unique exact match fast path:
		//
		// if exactly one exact-name/exact-alias candidate exists, accept it
		// locally without calling Gemini again.
		if exactCandidate, ok := classifyCandidates(foodCandidates); ok {
			candidates = append(candidates, indexedMealCandidate{
				index:     index,
				candidate: matchedMealCandidateFromFoodCandidate(meal, exactCandidate),
			})

			slog.Info("generated meal exact matched",
				"user_id", userID,
				"index", index,
				"meal_name", meal.Name,
				"matched_query", exactCandidate.MatchedTerm,
				"matched_food", exactCandidate.Food.Name,
			)
			continue
		}

		// Multiple plausible candidates, or only fuzzy/partial candidates, remain
		// ambiguous. Defer them into one batched adjudication call.
		task := buildMatchTask(index, meal, foodCandidates)
		matchTasks = append(matchTasks, task)

		slog.Info("generated meal requires adjudication",
			"user_id", userID,
			"index", index,
			"meal_name", meal.Name,
			"candidate_count", len(foodCandidates),
		)
	}

	// Resolve all ambiguous meals in at most one additional LLM call.
	if len(matchTasks) > 0 {
		adjudicationStart := time.Now()

		decisions, err := s.matchAdjudicator.ResolveMatches(ctx, matchTasks)
		if err != nil {
			// Adjudication failure should not accept fuzzy matches and should not
			// fail the whole recommendation request. Treat ambiguous meals as
			// unmatched so the auto-create path can handle them.
			slog.Error("meal match adjudication failed; ambiguous meals will be auto-generated",
				"user_id", userID,
				"task_count", len(matchTasks),
				"error", err,
				"duration_ms", time.Since(adjudicationStart).Milliseconds(),
			)

			for _, task := range matchTasks {
				missingMeals = append(missingMeals, generatedMealCreateInput{
					index: task.MealIndex,
					meal:  task.GeneratedMeal,
				})
			}
		} else {
			// Validate every Gemini decision against the backend-owned candidates
			// that were supplied for that exact meal index.
			validatedMatches := validateDecisions(matchTasks, decisions)

			slog.Info("meal match adjudication completed",
				"user_id", userID,
				"task_count", len(matchTasks),
				"decision_count", len(decisions),
				"valid_match_count", len(validatedMatches),
				"duration_ms", time.Since(adjudicationStart).Milliseconds(),
			)

			for _, task := range matchTasks {
				matchedCandidate, ok := validatedMatches[task.MealIndex]
				if !ok {
					// NO_MATCH, invalid decision, missing decision, duplicate
					// decision, or invented/cross-meal candidate ID all land here.
					missingMeals = append(missingMeals, generatedMealCreateInput{
						index: task.MealIndex,
						meal:  task.GeneratedMeal,
					})
					continue
				}

				candidates = append(candidates, indexedMealCandidate{
					index: task.MealIndex,
					candidate: matchedMealCandidateFromFoodCandidate(
						task.GeneratedMeal,
						matchedCandidate,
					),
				})

				slog.Info("generated meal adjudicated matched",
					"user_id", userID,
					"index", task.MealIndex,
					"meal_name", task.GeneratedMeal.Name,
					"matched_query", matchedCandidate.MatchedTerm,
					"matched_food", matchedCandidate.Food.Name,
				)
			}
		}
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

// matchedMealCandidateFromFoodCandidate converts an accepted FoodMatchCandidate
// into the public recommendation candidate shape.
//
// The accepted candidate must already be backend-owned and validated:
// - exact fast path candidates come from FoodCandidateSearcher
// - adjudicated candidates come from validateDecisions
func matchedMealCandidateFromFoodCandidate(meal interfaces.GeneratedMeal, candidate interfaces.FoodMatchCandidate) interfaces.MatchedMealCandidate {
	return interfaces.MatchedMealCandidate{
		GeneratedMeal: meal,
		Food:          candidate.Food,

		// Preserve backward-compatible MatchedQuery behavior by using the term
		// that actually produced the accepted candidate.
		MatchedQuery: candidate.MatchedTerm,
	}
}
