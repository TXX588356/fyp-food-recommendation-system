package recommendation

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

type testCatalogService struct {
	createdInput *interfaces.GeneratedCatalogMealInput
	createdMeal  interfaces.CatalogMeal
	createErr    error
	mealsByID    map[uuid.UUID]interfaces.CatalogMeal
	getErr       error
}

type testMealLogRepository struct {
	logs []model.MealLog
	err  error
}

type testCustomMealRepository struct {
	meal *model.CustomMealItem
	err  error
}

type testCandidateSearchCall struct {
	userID  uuid.UUID
	queries []string
	limit   int
}

type testCandidateSearcher struct {
	candidatesByQuery map[string][]interfaces.FoodMatchCandidate
	err               error
	calls             []testCandidateSearchCall
}

type testPreferenceService struct {
	response *interfaces.PreferenceResponse
	err      error
}

type testMealDetailExplainer struct {
	explanation string
	err         error
	input       *interfaces.MealDetailExplanationInput
}

type testRestaurantSearcher struct {
	result interfaces.RestaurantSearchResult
	err    error
}

func (s *testCandidateSearcher) SearchFoodCandidates(ctx context.Context, userID uuid.UUID, queries []string, limit int) ([]interfaces.FoodMatchCandidate, error) {
	s.calls = append(s.calls, testCandidateSearchCall{
		userID:  userID,
		queries: append([]string(nil), queries...),
		limit:   limit,
	})

	if s.err != nil {
		return nil, s.err
	}

	return s.candidatesByQuery[strings.Join(queries, "|")], nil
}

type testMatchAdjudicator struct {
	decisions []interfaces.MealMatchDecision
	err       error
	calls     [][]interfaces.MealMatchTask
}

func (a *testMatchAdjudicator) ResolveMatches(ctx context.Context, tasks []interfaces.MealMatchTask) ([]interfaces.MealMatchDecision, error) {
	a.calls = append(a.calls, append([]interfaces.MealMatchTask(nil), tasks...))

	if a.err != nil {
		return nil, a.err
	}

	return a.decisions, nil
}

func (s *testPreferenceService) CompleteOnboarding(context.Context, uuid.UUID, interfaces.PreferenceInput) (*interfaces.PreferenceResponse, error) {
	return s.response, s.err
}

func (s *testPreferenceService) GetByUserID(context.Context, uuid.UUID) (*interfaces.PreferenceResponse, error) {
	if s.err != nil {
		return nil, s.err
	}

	if s.response != nil {
		return s.response, nil
	}

	return &interfaces.PreferenceResponse{}, nil
}

func (s *testPreferenceService) Update(context.Context, uuid.UUID, interfaces.PreferenceInput) (*interfaces.PreferenceResponse, error) {
	return s.response, s.err
}

func (s *testPreferenceService) UpdateDataSharingConsent(context.Context, uuid.UUID, bool) error {
	return s.err
}

func (e *testMealDetailExplainer) ExplainMealRecommendation(_ context.Context, input interfaces.MealDetailExplanationInput) (string, error) {
	e.input = &input

	if e.err != nil {
		return "", e.err
	}

	if e.explanation != "" {
		return e.explanation, nil
	}

	return "This meal fits the user's current preferences.", nil
}

func (s *testRestaurantSearcher) SearchRestaurants(context.Context, interfaces.RestaurantSearchInput) (interfaces.RestaurantSearchResult, error) {
	if s.err != nil {
		return interfaces.RestaurantSearchResult{}, s.err
	}

	if s.result.Status != "" {
		return s.result, nil
	}

	return interfaces.RestaurantSearchResult{
		Status:      interfaces.RestaurantLookupUnavailable,
		Restaurants: []interfaces.RestaurantResult{},
	}, nil
}

func newTestCatalogService() *testCatalogService {
	return &testCatalogService{}
}

func (r *testMealLogRepository) Create(context.Context, *model.MealLog) (*model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogRepository) ListByUserAndMonth(context.Context, uuid.UUID, time.Time, time.Time) ([]model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogRepository) ListByUserAndRange(context.Context, uuid.UUID, time.Time, time.Time) ([]model.MealLog, error) {
	return r.logs, r.err
}

func (r *testMealLogRepository) FindByIDAndUser(context.Context, uuid.UUID, uuid.UUID) (*model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogRepository) Update(context.Context, *model.MealLog) (*model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogRepository) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (r *testCustomMealRepository) Create(context.Context, *model.CustomMealItem) (*model.CustomMealItem, error) {
	return nil, nil
}

func (r *testCustomMealRepository) ListOwnedByUser(context.Context, uuid.UUID, string) ([]model.CustomMealItem, error) {
	return nil, nil
}

func (r *testCustomMealRepository) ListSharedFromOtherUsers(context.Context, uuid.UUID, string) ([]model.CustomMealItem, error) {
	return nil, nil
}

func (r *testCustomMealRepository) FindVisibleByID(context.Context, uuid.UUID, uuid.UUID) (*model.CustomMealItem, error) {
	return r.meal, r.err
}

func (r *testCustomMealRepository) UpdateOwned(context.Context, uuid.UUID, *model.CustomMealItem) (*model.CustomMealItem, error) {
	return nil, nil
}

func (r *testCustomMealRepository) DeleteOwned(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (s *testCatalogService) CreateGeneratedMeal(ctx context.Context, input interfaces.GeneratedCatalogMealInput) (interfaces.CatalogMeal, error) {
	s.createdInput = &input
	if s.createErr != nil {
		return interfaces.CatalogMeal{}, s.createErr
	}

	return s.createdMeal, nil
}

func (s *testCatalogService) SearchMeals(context.Context, interfaces.CatalogQuery) (interfaces.CatalogMealPage, error) {
	return interfaces.CatalogMealPage{}, nil
}

func (s *testCatalogService) GetMeal(_ context.Context, id uuid.UUID) (interfaces.CatalogMeal, error) {
	if s.getErr != nil {
		return interfaces.CatalogMeal{}, s.getErr
	}

	if s.mealsByID != nil {
		if meal, ok := s.mealsByID[id]; ok {
			return meal, nil
		}
	}

	return interfaces.CatalogMeal{}, nil
}

func (s *testCatalogService) ListCategories(context.Context) ([]interfaces.CatalogCategory, error) {
	return nil, nil
}

func testFloatPtr(value float64) *float64 {
	return &value
}

// TestRecommendation runs the recommendation service test suite.
func TestRecommendation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Recommendation Suite")
}

var _ = Describe("Recommendation candidate generation", func() {
	var (
		ctx                     context.Context
		mealGenerator           *mocks.MealGenerator
		candidateSearcher       *testCandidateSearcher
		matchAdjudicator        *testMatchAdjudicator
		svc                     *service
		catalogService          interfaces.CatalogService
		customMealAutocompleter *mocks.CustomMealAutocompleter
		input                   interfaces.MealPromptInput
		userID                  uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		input = interfaces.MealPromptInput{
			History: interfaces.MealHistoryContext{
				RecentMealNames:        []string{},
				RecentCategoryCounts:   map[string]int{},
				RepeatedMealNames:      []string{},
				RecentlyEatenByName:    map[string]int{},
				LearnedCategoryCounts:  map[string]int{},
				FatiguedCategoryCounts: map[string]int{},
			},
		}
		userID = uuid.New()
		mealGenerator = mocks.NewMealGenerator(GinkgoT())
		candidateSearcher = &testCandidateSearcher{
			candidatesByQuery: map[string][]interfaces.FoodMatchCandidate{},
		}
		matchAdjudicator = &testMatchAdjudicator{}
		customMealAutocompleter = mocks.NewCustomMealAutocompleter(GinkgoT())
		catalogService = newTestCatalogService()
		svc = NewService(
			mealGenerator,
			candidateSearcher,
			matchAdjudicator,
			catalogService,
			customMealAutocompleter,
			&testMealLogRepository{},
			&testCustomMealRepository{},
			&testPreferenceService{},
			&testMealDetailExplainer{},
			&testRestaurantSearcher{},
		).(*service)
	})

	It("should accept one exact candidate without adjudication", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Nasi Lemak"},
				},
			}, nil).
			Once()

		candidateSearcher.candidatesByQuery["Nasi Lemak"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidate("food-1", "Nasi Lemak", interfaces.FoodMatchExactName, "Nasi Lemak", 1),
		}

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(HaveLen(1))
		Expect(result.Candidates[0].Food.ID).To(Equal("food-1"))
		Expect(result.Candidates[0].MatchedQuery).To(Equal("Nasi Lemak"))
		Expect(result.FilteringApplied).To(BeTrue())
		Expect(matchAdjudicator.calls).To(BeEmpty())
	})

	It("should pass generated name and alternative search terms to candidate search", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{
						Name: "Wantan Mee",
						AlternativeSearchTerms: []string{
							"Wonton Mee",
							"Wan Tan Mee",
						},
					},
				},
			}, nil).
			Once()

		candidateSearcher.candidatesByQuery["Wantan Mee|Wonton Mee|Wan Tan Mee"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidate("food-2", "Wantan Mee", interfaces.FoodMatchExactAlias, "Wan Tan Mee", 0.98),
		}

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(HaveLen(1))
		Expect(result.Candidates[0].MatchedQuery).To(Equal("Wan Tan Mee"))
		Expect(candidateSearcher.calls).To(HaveLen(1))
		Expect(candidateSearcher.calls[0].queries).To(Equal([]string{"Wantan Mee", "Wonton Mee", "Wan Tan Mee"}))
		Expect(candidateSearcher.calls[0].limit).To(Equal(5))
	})

	It("should auto-create an unmatched meal and return it in the final candidates", func() {
		generatedMeal := interfaces.GeneratedMeal{
			Name:                   "Unknown Meal",
			AlternativeSearchTerms: []string{"Unknown Food"},
			EstimatedPriceRange: interfaces.PriceRange{
				Min: 8,
				Max: 12,
			},
		}

		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{generatedMeal},
			}, nil).
			Once()

		customMealAutocompleter.EXPECT().AutocompleteCustomMeal(
			mock.Anything,
			interfaces.CustomMealAutocompleteInput{Name: "Unknown Meal"},
		).Return(interfaces.CustomMealAutocompleteResponse{
			Calories:               500,
			FatG:                   18,
			ProteinG:               24,
			CarbsG:                 62,
			DietaryRestrictionTags: []string{"halal"},
			MealCategoryTags:       []string{"rice_dishes"},
		}, nil).Once()

		catalogService.(*testCatalogService).createdMeal = interfaces.CatalogMeal{
			ID:             uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:           "Unknown Meal",
			SourceCode:     "ai_generated",
			SourceRecordID: "gemini:unknown meal",
			Categories:     []string{"rice_dishes"},
			SelectedNutrition: interfaces.CatalogNutrition{
				Calories: testFloatPtr(500),
				FatG:     testFloatPtr(18),
				ProteinG: testFloatPtr(24),
				CarbsG:   testFloatPtr(62),
			},
			SelectedPortion: interfaces.CatalogPortion{
				Amount:      1,
				Description: "1 serving",
			},
		}

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(HaveLen(1))
		Expect(result.Candidates[0].MatchedQuery).To(Equal("Unknown Meal"))
		Expect(result.Candidates[0].Food.ID).To(Equal("11111111-1111-1111-1111-111111111111"))

		Expect(catalogService.(*testCatalogService).createdInput).To(Equal(&interfaces.GeneratedCatalogMealInput{
			Name:               "Unknown Meal",
			CategoryCodes:      []string{"rice_dishes"},
			ServingDescription: "1 serving",
			Calories:           500,
			ProteinG:           24,
			CarbsG:             62,
			FatG:               18,
		}))
	})

	It("should preserve Gemini order for equal-score matched and auto-created meals", func() {
		input.PerMealBudget = 10
		generatedMeals := []interfaces.GeneratedMeal{
			{
				Name:                "Unknown Meal",
				EstimatedPriceRange: interfaces.PriceRange{Min: 5, Max: 5},
			},
			{
				Name:                "Existing Meal",
				EstimatedPriceRange: interfaces.PriceRange{Min: 5, Max: 5},
			},
		}

		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{Meals: generatedMeals}, nil).
			Once()

		candidateSearcher.candidatesByQuery["Existing Meal"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidate("existing", "Existing Meal", interfaces.FoodMatchExactName, "Existing Meal", 1),
		}

		customMealAutocompleter.EXPECT().AutocompleteCustomMeal(
			mock.Anything,
			interfaces.CustomMealAutocompleteInput{Name: "Unknown Meal"},
		).Return(interfaces.CustomMealAutocompleteResponse{
			Calories:         400,
			FatG:             10,
			ProteinG:         20,
			CarbsG:           40,
			MealCategoryTags: []string{"rice_dishes"},
		}, nil).Once()

		catalogService.(*testCatalogService).createdMeal = interfaces.CatalogMeal{
			ID:         uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Name:       "Unknown Meal",
			Categories: []string{},
			SelectedNutrition: interfaces.CatalogNutrition{
				Calories: testFloatPtr(400),
				FatG:     testFloatPtr(10),
				ProteinG: testFloatPtr(20),
				CarbsG:   testFloatPtr(40),
			},
		}

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(HaveLen(2))
		Expect(result.Candidates[0].Food.Name).To(Equal("Unknown Meal"))
		Expect(result.Candidates[1].Food.Name).To(Equal("Existing Meal"))
	})

	It("should skip an unmatched meal when generated catalog details fail", func() {
		generatedMeal := interfaces.GeneratedMeal{Name: "Unknown Meal"}

		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{generatedMeal},
			}, nil).
			Once()

		customMealAutocompleter.EXPECT().AutocompleteCustomMeal(
			mock.Anything,
			interfaces.CustomMealAutocompleteInput{Name: "Unknown Meal"},
		).Return(interfaces.CustomMealAutocompleteResponse{}, errors.New("unable to generate meal details")).Once()

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(BeEmpty())
	})

	It("should skip an unmatched meal when saving the generated prebuilt meal fails", func() {
		generatedMeal := interfaces.GeneratedMeal{Name: "Unknown Meal"}

		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{generatedMeal},
			}, nil).
			Once()

		customMealAutocompleter.EXPECT().AutocompleteCustomMeal(
			mock.Anything,
			interfaces.CustomMealAutocompleteInput{Name: "Unknown Meal"},
		).Return(interfaces.CustomMealAutocompleteResponse{
			Calories:         500,
			FatG:             18,
			ProteinG:         24,
			CarbsG:           62,
			MealCategoryTags: []string{"rice_dishes"},
		}, nil).Once()

		catalogService.(*testCatalogService).createErr = errors.New("database unavailable")

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(BeEmpty())
	})

	It("should adjudicate ambiguous candidates once", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Chicken Sausage"},
					{Name: "Fried Rice"},
				},
			}, nil).
			Once()

		candidateSearcher.candidatesByQuery["Chicken Sausage"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidate("sausage-1", "Chicken Breakfast Sausage", interfaces.FoodMatchFuzzy, "Chicken Sausage", 0.83),
			serviceFoodCandidate("sausage-2", "Grilled Chicken Sausage", interfaces.FoodMatchFuzzy, "Chicken Sausage", 0.83),
		}
		candidateSearcher.candidatesByQuery["Fried Rice"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidate("rice-1", "Fried Rice", interfaces.FoodMatchFuzzy, "Fried Rice", 0.85),
		}

		matchAdjudicator.decisions = []interfaces.MealMatchDecision{
			{MealIndex: 0, Decision: matchDecisionMatch, CandidateID: "sausage-1"},
			{MealIndex: 1, Decision: matchDecisionMatch, CandidateID: "rice-1"},
		}

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(matchAdjudicator.calls).To(HaveLen(1))
		Expect(matchAdjudicator.calls[0]).To(HaveLen(2))
		Expect(result.Candidates).To(HaveLen(2))
		Expect(result.Candidates[0].Food.ID).To(Equal("sausage-1"))
		Expect(result.Candidates[1].Food.ID).To(Equal("rice-1"))
	})

	It("should auto-create a meal when adjudication returns NO_MATCH", func() {
		generatedMeal := interfaces.GeneratedMeal{Name: "Unknown Sausage"}

		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{generatedMeal},
			}, nil).
			Once()

		candidateSearcher.candidatesByQuery["Unknown Sausage"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidate("candidate-1", "Chicken Sausage", interfaces.FoodMatchFuzzy, "Unknown Sausage", 0.80),
		}

		matchAdjudicator.decisions = []interfaces.MealMatchDecision{
			{MealIndex: 0, Decision: matchDecisionNoMatch},
		}

		customMealAutocompleter.EXPECT().AutocompleteCustomMeal(
			mock.Anything,
			interfaces.CustomMealAutocompleteInput{Name: "Unknown Sausage"},
		).Return(interfaces.CustomMealAutocompleteResponse{
			Calories:         500,
			FatG:             18,
			ProteinG:         24,
			CarbsG:           62,
			MealCategoryTags: []string{"rice_dishes"},
		}, nil).Once()

		catalogService.(*testCatalogService).createdMeal = interfaces.CatalogMeal{
			ID:         uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			Name:       "Unknown Sausage",
			Categories: []string{"rice_dishes"},
			SelectedNutrition: interfaces.CatalogNutrition{
				Calories: testFloatPtr(500),
				FatG:     testFloatPtr(18),
				ProteinG: testFloatPtr(24),
				CarbsG:   testFloatPtr(62),
			},
		}

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(HaveLen(1))
		Expect(result.Candidates[0].Food.ID).To(Equal("33333333-3333-3333-3333-333333333333"))
	})

	It("should auto-create ambiguous meals when adjudication fails", func() {
		generatedMeal := interfaces.GeneratedMeal{Name: "Ambiguous Meal"}

		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{generatedMeal},
			}, nil).
			Once()

		candidateSearcher.candidatesByQuery["Ambiguous Meal"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidate("candidate-1", "Ambiguous Meal A", interfaces.FoodMatchFuzzy, "Ambiguous Meal", 0.80),
		}
		matchAdjudicator.err = errors.New("adjudication unavailable")

		customMealAutocompleter.EXPECT().AutocompleteCustomMeal(
			mock.Anything,
			interfaces.CustomMealAutocompleteInput{Name: "Ambiguous Meal"},
		).Return(interfaces.CustomMealAutocompleteResponse{
			Calories:         400,
			FatG:             10,
			ProteinG:         20,
			CarbsG:           40,
			MealCategoryTags: []string{"rice_dishes"},
		}, nil).Once()

		catalogService.(*testCatalogService).createdMeal = interfaces.CatalogMeal{
			ID:         uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			Name:       "Ambiguous Meal",
			Categories: []string{"rice_dishes"},
			SelectedNutrition: interfaces.CatalogNutrition{
				Calories: testFloatPtr(400),
				FatG:     testFloatPtr(10),
				ProteinG: testFloatPtr(20),
				CarbsG:   testFloatPtr(40),
			},
		}

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(matchAdjudicator.calls).To(HaveLen(1))
		Expect(result.Candidates).To(HaveLen(1))
		Expect(result.Candidates[0].Food.ID).To(Equal("44444444-4444-4444-4444-444444444444"))
	})

	It("should return a controlled error when Gemini fails", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{}, errors.New("Gemini unavailable")).
			Once()

		_, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).To(MatchError("generated meal candidates: Gemini unavailable"))
	})

	It("should return a controlled error when candidate search fails", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Nasi Lemak"},
				},
			}, nil).
			Once()

		candidateSearcher.err = errors.New("candidate search unavailable")

		_, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).To(MatchError(ContainSubstring("candidate search unavailable")))
	})

	It("should expose filtered-out meals with reasons in the recommendation result", func() {
		input.DietaryRestrictions = []string{"halal"}

		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Vegetable Soup"},
					{Name: "Pork Noodles"},
				},
			}, nil).
			Once()

		candidateSearcher.candidatesByQuery["Vegetable Soup"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidateWithTags("safe", "Vegetable Soup", []string{"vegetables", "soups"}, interfaces.FoodMatchExactName, "Vegetable Soup", 1),
		}
		candidateSearcher.candidatesByQuery["Pork Noodles"] = []interfaces.FoodMatchCandidate{
			serviceFoodCandidateWithTags("pork-food", "Pork Noodles", []string{"pork", "noodle_dishes"}, interfaces.FoodMatchExactName, "Pork Noodles", 1),
		}

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.FilteringApplied).To(BeTrue())
		Expect(result.Candidates).To(HaveLen(1))
		Expect(result.Candidates[0].Food.ID).To(Equal("safe"))
		Expect(result.FilteredOut).To(HaveLen(1))
		Expect(result.FilteredOut[0].Candidate.Food.ID).To(Equal("pork-food"))
		Expect(result.FilteredOut[0].Reason).To(ContainSubstring("halal"))
	})

	It("should remove duplicate and blank fallback search terms", func() {
		terms := buildSearchTerms(interfaces.GeneratedMeal{
			Name: "Wantan Mee",
			AlternativeSearchTerms: []string{
				" ",
				"wantan mee",
				"Wan Tan Mee",
			},
		})

		Expect(terms).To(Equal([]string{
			"Wantan Mee",
			"Wan Tan Mee",
		}))
	})
})

var _ = Describe("Recommendation meal detail", func() {
	It("should hydrate serving description from catalog for saved candidates without it", func() {
		ctx := context.Background()
		userID := uuid.New()
		mealID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
		catalogService := newTestCatalogService()
		catalogService.mealsByID = map[uuid.UUID]interfaces.CatalogMeal{
			mealID: {
				ID:   mealID,
				Name: "Chicken Rice",
				SelectedPortion: interfaces.CatalogPortion{
					Amount:      1,
					Description: "1 bowl (350g)",
				},
			},
		}
		explainer := &testMealDetailExplainer{}
		svc := NewService(
			mocks.NewMealGenerator(GinkgoT()),
			&testCandidateSearcher{},
			&testMatchAdjudicator{},
			catalogService,
			mocks.NewCustomMealAutocompleter(GinkgoT()),
			&testMealLogRepository{},
			&testCustomMealRepository{},
			&testPreferenceService{},
			explainer,
			&testRestaurantSearcher{},
		).(*service)

		result, err := svc.BuildMealDetail(ctx, userID, interfaces.MealDetailInput{
			MealCategory: "lunch",
			Candidate: interfaces.MatchedMealCandidate{
				GeneratedMeal: interfaces.GeneratedMeal{
					EstimatedPriceRange: interfaces.PriceRange{Min: 8, Max: 12},
				},
				Food: interfaces.FoodSearchResult{
					ID:       mealID.String(),
					Name:     "Chicken Rice",
					Calories: 500,
					FatG:     12,
					ProteinG: 28,
					CarbsG:   60,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Meal.ServingDescription).To(Equal("1 bowl (350g)"))
		Expect(explainer.input).NotTo(BeNil())
		Expect(explainer.input.MealName).To(Equal("Chicken Rice"))
	})

	It("should include a custom meal restaurant when its state matches the current location state", func() {
		ctx := context.Background()
		userID := uuid.New()
		mealID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
		customMealRepository := &testCustomMealRepository{
			meal: &model.CustomMealItem{
				ID:             mealID,
				Name:           "Custom Chicken Rice",
				State:          "Selangor",
				District:       "Shah Alam",
				RestaurantName: "User Chicken Shop",
			},
		}
		svc := NewService(
			mocks.NewMealGenerator(GinkgoT()),
			&testCandidateSearcher{},
			&testMatchAdjudicator{},
			newTestCatalogService(),
			mocks.NewCustomMealAutocompleter(GinkgoT()),
			&testMealLogRepository{},
			customMealRepository,
			&testPreferenceService{},
			&testMealDetailExplainer{},
			&testRestaurantSearcher{},
		).(*service)

		result, err := svc.BuildMealDetail(ctx, userID, interfaces.MealDetailInput{
			MealCategory: "lunch",
			Location:     "Kajang, Selangor",
			Candidate: interfaces.MatchedMealCandidate{
				GeneratedMeal: interfaces.GeneratedMeal{
					EstimatedPriceRange: interfaces.PriceRange{Min: 8, Max: 12},
				},
				Food: interfaces.FoodSearchResult{
					ID:       mealID.String(),
					Name:     "Custom Chicken Rice",
					Source:   "custom",
					Calories: 500,
					FatG:     12,
					ProteinG: 28,
					CarbsG:   60,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Restaurants).To(HaveLen(1))
		Expect(result.Restaurants[0].Name).To(Equal("User Chicken Shop"))
		Expect(result.Restaurants[0].Source).To(Equal("user_recommended"))
	})

	It("should exclude a custom meal restaurant when its state does not match the current location state", func() {
		ctx := context.Background()
		userID := uuid.New()
		mealID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
		customMealRepository := &testCustomMealRepository{
			meal: &model.CustomMealItem{
				ID:             mealID,
				Name:           "Sabah Custom Meal",
				State:          "Sabah",
				District:       "Kota Kinabalu",
				RestaurantName: "Sabah Food Place",
			},
		}
		svc := NewService(
			mocks.NewMealGenerator(GinkgoT()),
			&testCandidateSearcher{},
			&testMatchAdjudicator{},
			newTestCatalogService(),
			mocks.NewCustomMealAutocompleter(GinkgoT()),
			&testMealLogRepository{},
			customMealRepository,
			&testPreferenceService{},
			&testMealDetailExplainer{},
			&testRestaurantSearcher{},
		).(*service)

		result, err := svc.BuildMealDetail(ctx, userID, interfaces.MealDetailInput{
			MealCategory: "lunch",
			Location:     "Kajang, Selangor",
			Candidate: interfaces.MatchedMealCandidate{
				GeneratedMeal: interfaces.GeneratedMeal{
					EstimatedPriceRange: interfaces.PriceRange{Min: 8, Max: 12},
				},
				Food: interfaces.FoodSearchResult{
					ID:       mealID.String(),
					Name:     "Sabah Custom Meal",
					Source:   "custom",
					Calories: 500,
					FatG:     12,
					ProteinG: 28,
					CarbsG:   60,
				},
			},
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Restaurants).To(BeEmpty())
	})
})

func serviceFoodCandidate(id string, name string, kind interfaces.FoodMatchKind, matchedTerm string, score float64) interfaces.FoodMatchCandidate {
	return serviceFoodCandidateWithTags(id, name, []string{"test_category"}, kind, matchedTerm, score)
}

func serviceFoodCandidateWithTags(id string, name string, tags []string, kind interfaces.FoodMatchKind, matchedTerm string, score float64) interfaces.FoodMatchCandidate {
	return interfaces.FoodMatchCandidate{
		Food: interfaces.FoodSearchResult{
			ID:       id,
			Name:     name,
			Tags:     tags,
			Calories: 500,
			FatG:     15,
			ProteinG: 25,
			CarbsG:   60,
		},
		MatchKind:   kind,
		MatchedTerm: matchedTerm,
		Score:       score,
	}
}
