package catalog

import (
	"context"
	"testing"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testRepository struct {
	meals        []model.PrebuiltMeal
	meal         *model.PrebuiltMeal
	createdInput *interfaces.GeneratedCatalogMealInput
	createdMeal  *model.PrebuiltMeal
	categoriesOK bool
}

func (r *testRepository) SearchMeals(context.Context, interfaces.CatalogQuery) ([]model.PrebuiltMeal, error) {
	return r.meals, nil
}
func (r *testRepository) FindMeal(context.Context, uuid.UUID) (*model.PrebuiltMeal, error) {
	return r.meal, nil
}
func (r *testRepository) ListCategories(context.Context) ([]interfaces.CatalogCategory, error) {
	return nil, nil
}
func (r *testRepository) CategoryCodesExist(_ context.Context, codes []string) (bool, error) {
	if len(codes) == 0 {
		return true, nil
	}
	return r.categoriesOK, nil
}

func (r *testRepository) CreateGeneratedMeal(ctx context.Context, input interfaces.GeneratedCatalogMealInput) (*model.PrebuiltMeal, error) {
	r.createdInput = &input
	if r.createdMeal != nil {
		return r.createdMeal, nil
	}
	return &model.PrebuiltMeal{
		ID:                 uuid.New(),
		SourceCode:         "ai_generated",
		SourceRecordID:     "gemini:test-meal",
		Name:               input.Name,
		NormalizedName:     "test meal",
		CategoryCodes:      model.StringArray(input.CategoryCodes),
		ServingDescription: input.ServingDescription,
		Calories:           &input.Calories,
		ProteinG:           &input.ProteinG,
		CarbsG:             &input.CarbsG,
		FatG:               &input.FatG,
	}, nil
}

type testResolver struct{}

func (testResolver) Resolve(key string) string { return "https://images.test/" + key }

func float(value float64) *float64 { return &value }
func TestCatalogService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Catalog Service Suite")
}

var _ = Describe("Catalog service", func() {
	It("should return stored serving nutrition", func() {
		mealID := uuid.New()
		repo := &testRepository{meal: &model.PrebuiltMeal{
			ID: mealID, Name: "Fried Rice", SourceCode: "usda_fndds", SourceRecordID: "270001",
			CategoryCodes: []string{"rice_dishes", "indonesian"}, ServingDescription: "plate",
			Calories: float(500), ProteinG: float(25), CarbsG: float(75), FatG: float(12.5),
		}}
		service := NewService(repo, testResolver{})

		got, err := service.GetMeal(context.Background(), mealID)

		Expect(err).NotTo(HaveOccurred())
		Expect(got.SelectedNutrition.Calories).NotTo(BeNil())
		Expect(*got.SelectedNutrition.Calories).To(Equal(float64(500)))
		Expect(got.SelectedPortion.Description).To(Equal("plate"))
		Expect(got.SourceCode).To(Equal("usda_fndds"))
		Expect(got.Image).To(BeNil())
	})

	It("should reject excessive search limits", func() {
		service := NewService(&testRepository{}, testResolver{})

		_, err := service.SearchMeals(context.Background(), interfaces.CatalogQuery{Limit: 101})

		Expect(err).To(MatchError(interfaces.ErrInvalidCatalogQuery))
	})

	It("should create and map generated catalog meals", func() {
		repo := &testRepository{categoriesOK: true}
		service := NewService(repo, testResolver{})

		got, err := service.CreateGeneratedMeal(context.Background(), interfaces.GeneratedCatalogMealInput{
			Name:               "Unknown Meal",
			CategoryCodes:      []string{"rice_dishes"},
			ServingDescription: "1 serving",
			Calories:           500,
			ProteinG:           24,
			CarbsG:             62,
			FatG:               18,
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(repo.createdInput).NotTo(BeNil())
		Expect(repo.createdInput.Name).To(Equal("Unknown Meal"))
		Expect(repo.createdInput.CategoryCodes).To(Equal([]string{"rice_dishes"}))
		Expect(got.SourceCode).To(Equal("ai_generated"))
		Expect(got.SourceRecordID).To(Equal("gemini:test-meal"))
		Expect(got.SelectedNutrition.Calories).NotTo(BeNil())
		Expect(*got.SelectedNutrition.Calories).To(Equal(float64(500)))
		Expect(got.SelectedPortion.Description).To(Equal("1 serving"))
	})

	It("should reject invalid generated meal input", func() {
		valid := interfaces.GeneratedCatalogMealInput{
			Name:               "Unknown Meal",
			CategoryCodes:      []string{"rice_dishes"},
			ServingDescription: "1 serving",
			Calories:           500,
			ProteinG:           24,
			CarbsG:             62,
			FatG:               18,
		}

		tests := []struct {
			name         string
			input        interfaces.GeneratedCatalogMealInput
			categoriesOK bool
		}{
			{name: "empty category codes", input: func() interfaces.GeneratedCatalogMealInput {
				input := valid
				input.CategoryCodes = nil
				return input
			}(), categoriesOK: true},
			{name: "unsupported category codes", input: valid, categoriesOK: false},
			{name: "empty serving description", input: func() interfaces.GeneratedCatalogMealInput {
				input := valid
				input.ServingDescription = " "
				return input
			}(), categoriesOK: true},
		}

		for _, test := range tests {
			repo := &testRepository{categoriesOK: test.categoriesOK}
			service := NewService(repo, testResolver{})

			_, err := service.CreateGeneratedMeal(context.Background(), test.input)

			Expect(err).To(MatchError(interfaces.ErrInvalidCatalogQuery), test.name)
			Expect(repo.createdInput).To(BeNil(), test.name)
		}
	})
})
