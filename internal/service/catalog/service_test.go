package catalog

import (
	"context"
	"testing"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
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
func (r *testRepository) ListImages(context.Context, interfaces.CatalogImageQuery) ([]model.PrebuiltMealImage, error) {
	return nil, nil
}
func (r *testRepository) FindImage(context.Context, uuid.UUID) (*model.PrebuiltMealImage, error) {
	return nil, nil
}
func (r *testRepository) ApplyImageAction(context.Context, uuid.UUID, interfaces.CatalogImageAction) (*model.PrebuiltMealImage, error) {
	return nil, nil
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
func str(value string) *string     { return &value }

func TestGetMealReturnsStoredServingNutritionAndImageURL(t *testing.T) {
	mealID, imageID := uuid.New(), uuid.New()
	repo := &testRepository{meal: &model.PrebuiltMeal{
		ID: mealID, Name: "Fried Rice", SourceCode: "usda_fndds", SourceRecordID: "270001",
		CategoryCodes: []string{"rice_dishes", "indonesian"}, ServingDescription: "plate",
		Calories: float(500), ProteinG: float(25), CarbsG: float(75), FatG: float(12.5),
		Images: []model.PrebuiltMealImage{{ID: imageID, MinioObjectKey: str("catalog-meals/x.jpg"), IsPrimary: true}},
	}}
	service := NewService(repo, testResolver{})

	got, err := service.GetMeal(context.Background(), mealID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SelectedNutrition.Calories == nil || *got.SelectedNutrition.Calories != 500 {
		t.Fatalf("expected 500 calories, got %#v", got.SelectedNutrition.Calories)
	}
	if got.SelectedPortion.Description != "plate" || got.SourceCode != "usda_fndds" {
		t.Fatalf("unexpected flattened meal mapping: %#v", got)
	}
	if got.Image == nil || got.Image.URL != "https://images.test/catalog-meals/x.jpg" {
		t.Fatalf("unexpected image: %#v", got.Image)
	}
}

func TestSearchMealsRejectsExcessiveLimit(t *testing.T) {
	service := NewService(&testRepository{}, testResolver{})
	_, err := service.SearchMeals(context.Background(), interfaces.CatalogQuery{Limit: 101})
	if err != interfaces.ErrInvalidCatalogQuery {
		t.Fatalf("expected invalid query, got %v", err)
	}
}

func TestCreateGeneratedMealCreatesAndMapsCatalogMeal(t *testing.T) {
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
	if err != nil {
		t.Fatal(err)
	}

	if repo.createdInput == nil {
		t.Fatal("expected repository CreateGeneratedMeal to be called")
	}
	if repo.createdInput.Name != "Unknown Meal" || repo.createdInput.CategoryCodes[0] != "rice_dishes" {
		t.Fatalf("unexpected create input: %#v", repo.createdInput)
	}
	if got.SourceCode != "ai_generated" || got.SourceRecordID != "gemini:test-meal" {
		t.Fatalf("unexpected generated source mapping: %#v", got)
	}
	if got.SelectedNutrition.Calories == nil || *got.SelectedNutrition.Calories != 500 {
		t.Fatalf("expected generated calories to be mapped, got %#v", got.SelectedNutrition.Calories)
	}
	if got.SelectedPortion.Description != "1 serving" {
		t.Fatalf("unexpected generated portion: %#v", got.SelectedPortion)
	}
}

func TestCreateGeneratedMealRejectsInvalidInput(t *testing.T) {
	valid := interfaces.GeneratedCatalogMealInput{
		Name:               "Unknown Meal",
		CategoryCodes:      []string{"rice_dishes"},
		ServingDescription: "1 serving",
		Calories:           500,
		ProteinG:           24,
		CarbsG:             62,
		FatG:               18,
	}

	for _, test := range []struct {
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
	} {
		t.Run(test.name, func(t *testing.T) {
			repo := &testRepository{categoriesOK: test.categoriesOK}
			service := NewService(repo, testResolver{})

			_, err := service.CreateGeneratedMeal(context.Background(), test.input)
			if err != interfaces.ErrInvalidCatalogQuery {
				t.Fatalf("expected invalid catalog query, got %v", err)
			}
			if repo.createdInput != nil {
				t.Fatalf("repository should not create invalid generated meal: %#v", repo.createdInput)
			}
		})
	}
}
