package catalog

import (
	"context"

	"fyp/food-rs/internal/interfaces"
	"github.com/google/uuid"
)

type FoodSearcher struct{ service *Service }

func NewFoodSearcher(service *Service) interfaces.FoodSearcher {
	return &FoodSearcher{service: service}
}

func (s *FoodSearcher) SearchFood(ctx context.Context, _ uuid.UUID, query string) (interfaces.FoodSearchResult, bool, error) {
	page, err := s.service.SearchMeals(ctx, interfaces.CatalogQuery{Query: query, Limit: 1})
	if err != nil {
		return interfaces.FoodSearchResult{}, false, err
	}
	if len(page.Items) == 0 {
		return interfaces.FoodSearchResult{}, false, nil
	}
	meal := page.Items[0]
	n := meal.SelectedNutrition
	if n.Calories == nil || n.ProteinG == nil || n.CarbsG == nil || n.FatG == nil {
		return interfaces.FoodSearchResult{}, false, nil
	}
	imageURL := ""
	if meal.Image != nil {
		imageURL = meal.Image.URL
	}
	return interfaces.FoodSearchResult{
		ID:                 meal.ID.String(),
		Name:               meal.Name,
		Tags:               meal.Categories,
		Calories:           *n.Calories,
		ProteinG:           *n.ProteinG,
		CarbsG:             *n.CarbsG,
		FatG:               *n.FatG,
		ServingDescription: meal.SelectedPortion.Description,
		ImageURL:           imageURL,
	}, true, nil
}
