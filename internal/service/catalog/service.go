package catalog

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"github.com/google/uuid"
)

type Service struct {
	repository  interfaces.CatalogRepository
	urlResolver interfaces.ObjectURLResolver
}

func NewService(repository interfaces.CatalogRepository, resolver interfaces.ObjectURLResolver) *Service {
	return &Service{repository: repository, urlResolver: resolver}
}

type cursor struct {
	Name string    `json:"name"`
	ID   uuid.UUID `json:"id"`
}

func (s *Service) SearchMeals(ctx context.Context, query interfaces.CatalogQuery) (interfaces.CatalogMealPage, error) {
	if query.Limit < 0 || query.Limit > 100 {
		return interfaces.CatalogMealPage{}, interfaces.ErrInvalidCatalogQuery
	}
	if query.Limit == 0 {
		query.Limit = 20
	}
	if len(query.Categories) > 0 {
		valid, err := s.repository.CategoryCodesExist(ctx, query.Categories)
		if err != nil {
			return interfaces.CatalogMealPage{}, err
		}
		if !valid {
			return interfaces.CatalogMealPage{}, interfaces.ErrInvalidCatalogQuery
		}
	}
	meals, err := s.repository.SearchMeals(ctx, query)
	if err != nil {
		return interfaces.CatalogMealPage{}, err
	}
	page := interfaces.CatalogMealPage{Items: make([]interfaces.CatalogMeal, 0, len(meals))}
	for _, meal := range meals {
		mapped, err := s.mapMeal(meal)
		if err == nil {
			page.Items = append(page.Items, mapped)
		}
	}
	if len(page.Items) > query.Limit {
		page.Items = page.Items[:query.Limit]
		last := meals[query.Limit-1]
		raw, _ := json.Marshal(cursor{Name: last.NormalizedName, ID: last.ID})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return page, nil
}

func (s *Service) GetMeal(ctx context.Context, id uuid.UUID) (interfaces.CatalogMeal, error) {
	meal, err := s.repository.FindMeal(ctx, id)
	if err != nil {
		return interfaces.CatalogMeal{}, err
	}
	return s.mapMeal(*meal)
}

func (s *Service) ListCategories(ctx context.Context) ([]interfaces.CatalogCategory, error) {
	return s.repository.ListCategories(ctx)
}

func (s *Service) mapMeal(meal model.PrebuiltMeal) (interfaces.CatalogMeal, error) {
	result := interfaces.CatalogMeal{
		ID:                meal.ID,
		Name:              meal.Name,
		SourceCode:        meal.SourceCode,
		SourceRecordID:    meal.SourceRecordID,
		Categories:        []string(meal.CategoryCodes),
		SelectedPortion:   interfaces.CatalogPortion{Amount: 1, Description: meal.ServingDescription, GramWeight: meal.ServingGramWeight},
		SelectedNutrition: nutrition(meal),
	}
	if len(meal.Images) > 0 && meal.Images[0].MinioObjectKey != nil {
		image := meal.Images[0]
		result.Image = &interfaces.CatalogImage{ID: image.ID, URL: s.urlResolver.Resolve(*image.MinioObjectKey), Attribution: value(image.AttributionText), SourceURL: value(image.CommonsPageURL), License: value(image.LicenseName), LicenseURL: value(image.LicenseURL)}
	}
	return result, nil
}

func nutrition(meal model.PrebuiltMeal) interfaces.CatalogNutrition {
	return interfaces.CatalogNutrition{Calories: meal.Calories, ProteinG: meal.ProteinG, CarbsG: meal.CarbsG, FatG: meal.FatG, FiberG: meal.FiberG, SugarG: meal.SugarG, SodiumMg: meal.SodiumMg, CholesterolMg: meal.CholesterolMg}
}
func value(input *string) string {
	if input == nil {
		return ""
	}
	return *input
}
