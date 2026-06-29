package catalog

import (
	"context"
	"testing"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"github.com/google/uuid"
)

type testRepository struct {
	meals []model.PrebuiltMeal
	meal  *model.PrebuiltMeal
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
func (r *testRepository) CategoryCodesExist(context.Context, []string) (bool, error) {
	return true, nil
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

type testResolver struct{}

func (testResolver) Resolve(key string) string { return "https://images.test/" + key }

func float(value float64) *float64 { return &value }
func str(value string) *string     { return &value }

func TestGetMealReturnsStoredServingNutritionAndImageURL(t *testing.T) {
	mealID, imageID := uuid.New(), uuid.New()
	repo := &testRepository{meal: &model.PrebuiltMeal{
		ID: mealID, Name: "Fried Rice", SourceCode: "usda_fndds", SourceRecordID: "270001",
		CategoryCodes: []string{"rice_dishes", "indonesian"}, ServingDescription: "plate", ServingGramWeight: 250,
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
