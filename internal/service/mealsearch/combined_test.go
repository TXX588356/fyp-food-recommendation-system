package mealsearch

import (
	"context"
	"errors"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestMealSearch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Meal Search Suite")
}

var _ = Describe("Combined meal searcher", func() {
	var (
		ctx               context.Context
		userID            uuid.UUID
		customMealService *mocks.CustomMealService
		prebuiltSearcher  *mocks.FoodSearcher
		searcher          interfaces.FoodSearcher
	)

	BeforeEach(func() {
		ctx = context.Background()
		userID = uuid.New()
		customMealService = mocks.NewCustomMealService(GinkgoT())
		prebuiltSearcher = mocks.NewFoodSearcher(GinkgoT())
		searcher = NewCombinedSearcher(customMealService, prebuiltSearcher)
	})

	It("should return a custom meal before searching prebuilt meals", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Nasi Lemak").
			Return([]*interfaces.CustomMealResponse{
				{
					ID:               "1",
					Name:             "Nasi Lemak",
					Calories:         530,
					FatG:             18,
					ProteinG:         20,
					CarbsG:           75,
					MealCategoryTags: []string{"Rice"},
				},
			}, nil).Once()
		food, found, err := searcher.SearchFood(ctx, userID, "Nasi Lemak")

		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(food.ID).To(Equal("1"))
		Expect(food.Name).To(Equal("Nasi Lemak"))
		Expect(food.Source).To(Equal("custom"))
		Expect(food.Calories).To(Equal(float64(530)))
		Expect(food.FatG).To(Equal(float64(18)))
		Expect(food.ProteinG).To(Equal(float64(20)))
		Expect(food.CarbsG).To(Equal(float64(75)))
		Expect(food.Tags).To(Equal([]string{"Rice"}))
	})

	// TODO:
	It("should fall back to prebuilt meals when no custom meal matches", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Nasi Lemak").
			Return([]*interfaces.CustomMealResponse{}, nil).
			Once()

		prebuiltSearcher.EXPECT().SearchFood(mock.Anything, userID, "Nasi Lemak").
			Return(interfaces.FoodSearchResult{
				ID:       "100",
				Name:     "Nasi Lemak",
				Tags:     []string{"Rice"},
				Calories: 494,
				FatG:     14,
				ProteinG: 13,
				CarbsG:   80,
			}, true, nil).Once()

		food, found, err := searcher.SearchFood(ctx, userID, "Nasi Lemak")

		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(food.ID).To(Equal("100"))
		Expect(food.Name).To(Equal("Nasi Lemak"))
		Expect(food.Calories).To(Equal(float64(494)))
	})

	It("should return found false when neither custom nor prebuilt meals match", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Unknown meal").
			Return([]*interfaces.CustomMealResponse{}, nil).
			Once()

		prebuiltSearcher.EXPECT().SearchFood(mock.Anything, userID, "Unknown meal").
			Return(interfaces.FoodSearchResult{}, false, nil).
			Once()

		_, found, err := searcher.SearchFood(ctx, userID, "Unknown meal")

		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeFalse())
	})

	It("should return an error when custom meal search fails", func() {
		customMealService.EXPECT().ListVisible(mock.Anything, userID, "Nasi Lemak").
			Return(nil, errors.New("database unavailable")).
			Once()

		_, found, err := searcher.SearchFood(ctx, userID, "Nasi Lemak")

		Expect(found).To(BeFalse())
		Expect(err).To(MatchError("database unavailable"))
	})
})
