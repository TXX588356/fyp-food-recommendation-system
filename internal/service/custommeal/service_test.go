package custommeal

import (
	"context"
	"testing"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

// TestCustomMealService runs the custom-meal service test suite.
func TestCustomMealService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Custom Meal Service Suite")
}

var _ = Describe("Custom meal service", func() {
	var (
		ctx    context.Context
		repo   *mocks.CustomMealRepository
		svc    interfaces.CustomMealService
		userID uuid.UUID
	)

	BeforeEach(func() {
		ctx = context.Background()
		userID = uuid.New()
		repo = mocks.NewCustomMealRepository(GinkgoT())
		svc = NewService(repo)
	})

	// validInput returns a complete valid request used as the base for tests.
	validInput := func() interfaces.CustomMealInput {
		return interfaces.CustomMealInput{
			Name:           "Dubai Chocolate",
			Price:          9.50,
			Calories:       400,
			FatG:           50,
			ProteinG:       20,
			CarbsG:         30,
			State:          "Kuala Lumpur",
			District:       "Bangsar",
			RestaurantName: "FamilyMart",
			DietaryRestrictionTags: []string{
				"halal",
			},
			MealCategoryTags: []string{
				"nuts",
				"snacks",
			},
		}
	}

	// customMeal builds a persisted custom-meal model for repository responses.
	customMeal := func(id uuid.UUID, ownerID uuid.UUID, name string) model.CustomMealItem {
		return model.CustomMealItem{
			ID:             id,
			Name:           name,
			Price:          9.50,
			Calories:       400,
			FatG:           50,
			ProteinG:       20,
			CarbsG:         30,
			State:          "Kuala Lumpur",
			District:       "Bangsar",
			RestaurantName: "FamilyMart",
			CreatedBy:      ownerID,
			DietaryRestrictionTags: []model.CustomMealDietaryRestrictionTag{
				{DietaryRestrictionTag: "halal"},
			},
			MealCategoryTags: []model.CustomMealCategoryTag{
				{MealCategory: "nuts"},
				{MealCategory: "snacks"},
			},
		}
	}

	Describe("Create", func() {
		It("should save a valid custom meal", func() {
			savedMeal := customMeal(uuid.New(), userID, "Dubai Chocolate")

			repo.EXPECT().Create(
				mock.Anything,
				mock.MatchedBy(func(meal *model.CustomMealItem) bool {
					return meal.Name == "Dubai Chocolate" &&
						meal.CreatedBy == userID &&
						len(meal.DietaryRestrictionTags) == 1 &&
						len(meal.MealCategoryTags) == 2
				}),
			).
				Return(&savedMeal, nil).
				Once()

			response, err := svc.Create(ctx, userID, validInput())

			Expect(err).NotTo(HaveOccurred())
			Expect(response).NotTo(BeNil())
			Expect(response.Name).To(Equal("Dubai Chocolate"))
			Expect(response.IsOwner).To(BeTrue())
			Expect(response.IsShared).To(BeFalse())
			Expect(response.DietaryRestrictionTags).To(Equal([]string{"halal"}))
			Expect(response.MealCategoryTags).To(Equal([]string{"nuts", "snacks"}))
		})

		It("should reject missing name", func() {
			input := validInput()
			input.Name = " "

			response, err := svc.Create(ctx, userID, input)

			Expect(err).To(MatchError("custom meal name is required"))
			Expect(response).To(BeNil())
		})

		It("should reject negative price", func() {
			input := validInput()
			input.Price = -0.10

			response, err := svc.Create(ctx, userID, input)

			Expect(err).To(MatchError("price cannot be negative"))
			Expect(response).To(BeNil())
		})

		It("should reject missing state, district, and restaurant", func() {
			input := validInput()
			input.State = " "

			response, err := svc.Create(ctx, userID, input)

			Expect(err).To(MatchError("state is required"))
			Expect(response).To(BeNil())

			input = validInput()
			input.District = " "

			response, err = svc.Create(ctx, userID, input)

			Expect(err).To(MatchError("district is required"))
			Expect(response).To(BeNil())

			input = validInput()
			input.RestaurantName = " "

			response, err = svc.Create(ctx, userID, input)

			Expect(err).To(MatchError("restaurant name is required"))
			Expect(response).To(BeNil())
		})

		It("should reject unsupported dietary restriction tag", func() {
			input := validInput()
			input.DietaryRestrictionTags = []string{"invalid_diet"}

			response, err := svc.Create(ctx, userID, input)

			Expect(err).To(MatchError("unsupported dietary restriction tag: invalid_diet"))
			Expect(response).To(BeNil())
		})

		It("should reject unsupported meal category tag", func() {
			input := validInput()
			input.MealCategoryTags = []string{"invalid_category"}

			response, err := svc.Create(ctx, userID, input)

			Expect(err).To(MatchError("unsupported meal preference tag: invalid_category"))
			Expect(response).To(BeNil())
		})
	})

	Describe("ListVisible", func() {
		It("should combine owned and shared custom meals", func() {
			otherUserID := uuid.New()
			ownedMeals := []model.CustomMealItem{
				customMeal(uuid.New(), userID, "My Dubai Chocolate"),
			}
			sharedMeals := []model.CustomMealItem{
				customMeal(uuid.New(), otherUserID, "Shared Chicken Rice"),
			}

			repo.EXPECT().ListOwnedByUser(mock.Anything, userID, "chicken").
				Return(ownedMeals, nil).
				Once()
			repo.EXPECT().ListSharedFromOtherUsers(mock.Anything, userID, "chicken").
				Return(sharedMeals, nil).
				Once()

			responses, err := svc.ListVisible(ctx, userID, "chicken")

			Expect(err).NotTo(HaveOccurred())
			Expect(responses).To(HaveLen(2))

			Expect(responses[0].Name).To(Equal("My Dubai Chocolate"))
			Expect(responses[0].IsOwner).To(BeTrue())
			Expect(responses[0].IsShared).To(BeFalse())

			Expect(responses[1].Name).To(Equal("Shared Chicken Rice"))
			Expect(responses[1].IsOwner).To(BeFalse())
			Expect(responses[1].IsShared).To(BeTrue())
		})
	})

	Describe("FindVisibleByID", func() {
		It("should map one visible custom meal response", func() {
			mealID := uuid.New()
			visibleMeal := customMeal(mealID, userID, "My Dubai Chocolate")

			repo.EXPECT().FindVisibleByID(mock.Anything, userID, mealID).
				Return(&visibleMeal, nil).
				Once()

			response, err := svc.FindVisibleByID(ctx, userID, mealID)

			Expect(err).NotTo(HaveOccurred())
			Expect(response.ID).To(Equal(mealID.String()))
			Expect(response.Name).To(Equal("My Dubai Chocolate"))
			Expect(response.IsOwner).To(BeTrue())
			Expect(response.IsShared).To(BeFalse())
			Expect(response.DietaryRestrictionTags).To(Equal([]string{"halal"}))
			Expect(response.MealCategoryTags).To(Equal([]string{"nuts", "snacks"}))
		})
	})
})
