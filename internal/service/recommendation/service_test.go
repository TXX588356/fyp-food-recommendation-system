package recommendation

import (
	"context"
	"errors"
	"testing"

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

func newTestCatalogService() *testCatalogService {
	return &testCatalogService{}
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
		service                 interfaces.RecommendationService
		catalogService          interfaces.CatalogService
		customMealAutocompleter *mocks.CustomMealAutocompleter
		input                   interfaces.MealPromptInput
		userID                  uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		input = interfaces.MealPromptInput{}
		userID = uuid.New()
		mealGenerator = mocks.NewMealGenerator(GinkgoT())
		foodSearcher = mocks.NewFoodSearcher(GinkgoT())
		customMealAutocompleter = mocks.NewCustomMealAutocompleter(GinkgoT())
		catalogService = newTestCatalogService()
		service = NewService(mealGenerator, foodSearcher, catalogService, customMealAutocompleter)
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

		candidates, err := service.GenerateCandidates(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].MatchedQuery).To(Equal("Nasi Lemak"))
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

		candidates, err := service.GenerateCandidates(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].MatchedQuery).To(Equal("Wan Tan Mee"))
	})

	It("should auto-create a prebuilt meal when every search term returns no result", func() {
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

		candidates, err := service.GenerateCandidates(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(HaveLen(1))
		Expect(candidates[0].MatchedQuery).To(Equal("Unknown Meal"))
		Expect(candidates[0].Food.ID).To(Equal("11111111-1111-1111-1111-111111111111"))
		Expect(candidates[0].Food.Name).To(Equal("Unknown Meal"))
		Expect(candidates[0].Food.Calories).To(Equal(float64(500)))
		Expect(candidates[0].Food.Tags).To(Equal([]string{"rice_dishes"}))
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

		candidates, err := service.GenerateCandidates(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(BeEmpty())
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

		candidates, err := service.GenerateCandidates(ctx, userID, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(candidates).To(BeEmpty())
	})

	It("should return a controlled error when Gemini fails", func() {
		mealGenerator.EXPECT().GenerateMeals(mock.Anything, input).
			Return(interfaces.GeminiMealsResponse{}, errors.New("Gemini unavailable")).
			Once()

		_, err := service.GenerateCandidates(ctx, userID, input)

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

		_, err := service.GenerateCandidates(ctx, userID, input)

		Expect(err).To(MatchError(`search food "Nasi Lemak": food API unavailable`))
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
