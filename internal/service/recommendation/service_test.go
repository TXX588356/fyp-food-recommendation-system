package recommendation

import (
	"context"
	"errors"
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
}

type testMealLogRepository struct {
	logs []model.MealLog
	err  error
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

func (s *testCatalogService) GetMeal(context.Context, uuid.UUID) (interfaces.CatalogMeal, error) {
	return interfaces.CatalogMeal{}, nil
}

func (s *testCatalogService) ListCategories(context.Context) ([]interfaces.CatalogCategory, error) {
	return nil, nil
}

func (s *testCatalogService) ListImages(context.Context, interfaces.CatalogImageQuery) ([]model.PrebuiltMealImage, string, error) {
	return nil, "", nil
}

func (s *testCatalogService) GetImage(context.Context, uuid.UUID) (*model.PrebuiltMealImage, error) {
	return nil, nil
}

func (s *testCatalogService) ApplyImageAction(context.Context, uuid.UUID, interfaces.CatalogImageAction) (*model.PrebuiltMealImage, error) {
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
		foodSearcher            *mocks.FoodSearcher
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
				RecentMealNames:      []string{},
				RecentCategoryCounts: map[string]int{},
				RepeatedMealNames:    []string{},
				RecentlyEatenByName:  map[string]int{},
			},
		}
		userID = uuid.New()
		mealGenerator = mocks.NewMealGenerator(GinkgoT())
		foodSearcher = mocks.NewFoodSearcher(GinkgoT())
		customMealAutocompleter = mocks.NewCustomMealAutocompleter(GinkgoT())
		catalogService = newTestCatalogService()
		svc = NewService(mealGenerator, foodSearcher, catalogService, customMealAutocompleter, &testMealLogRepository{}).(*service)
	})

	It("should match a meal using its normalized name first", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Nasi Lemak"},
				},
			}, nil).
			Once()

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Nasi Lemak").
			Return(interfaces.FoodSearchResult{
				ID:   "food-1",
				Name: "Nasi Lemak",
			}, true, nil).
			Once()

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(HaveLen(1))
		Expect(result.Candidates[0].MatchedQuery).To(Equal("Nasi Lemak"))
		Expect(result.FilteringApplied).To(BeTrue())
	})

	It("should try alternative search terms when the normalized name is not found", func() {
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

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Wantan Mee").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()
		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Wonton Mee").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()
		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Wan Tan Mee").
			Return(interfaces.FoodSearchResult{
				ID:   "food-2",
				Name: "Wantan Mee",
			}, true, nil).
			Once()

		result, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(result.Candidates).To(HaveLen(1))
		Expect(result.Candidates[0].MatchedQuery).To(Equal("Wan Tan Mee"))
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

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Unknown Meal").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()
		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Unknown Food").
			Return(interfaces.FoodSearchResult{}, false, nil).
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

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Unknown Meal").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()
		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Existing Meal").
			Return(interfaces.FoodSearchResult{
				ID:       "existing",
				Name:     "Existing Meal",
				FatG:     10,
				ProteinG: 20,
				CarbsG:   40,
			}, true, nil).
			Once()

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

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Unknown Meal").
			Return(interfaces.FoodSearchResult{}, false, nil).
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

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Unknown Meal").
			Return(interfaces.FoodSearchResult{}, false, nil).
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

	It("should return a controlled error when Gemini fails", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{}, errors.New("Gemini unavailable")).
			Once()

		_, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).To(MatchError("generated meal candidates: Gemini unavailable"))
	})

	It("should return a controlled error when food search fails", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{
				Meals: []interfaces.GeneratedMeal{
					{Name: "Nasi Lemak"},
				},
			}, nil).
			Once()

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Nasi Lemak").
			Return(interfaces.FoodSearchResult{}, false, errors.New("food API unavailable")).
			Once()

		_, err := svc.GenerateRecommendationResult(ctx, userID, input)

		Expect(err).To(MatchError(`search food "Nasi Lemak": food API unavailable`))
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

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Vegetable Soup").
			Return(interfaces.FoodSearchResult{
				ID:   "safe",
				Name: "Vegetable Soup",
				Tags: []string{"vegetables", "soups"},
			}, true, nil).
			Once()

		foodSearcher.EXPECT().SearchFood(mock.Anything, userID, "Pork Noodles").
			Return(interfaces.FoodSearchResult{
				ID:   "pork-food",
				Name: "Pork Noodles",
				Tags: []string{"pork", "noodle_dishes"},
			}, true, nil).
			Once()

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
