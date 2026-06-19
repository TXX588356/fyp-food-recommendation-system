package custommeal

import (
	"context"
	"errors"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"strings"

	"github.com/google/uuid"
)

type service struct {
	customMealRepo interfaces.CustomMealRepository
}

// NewService creates a custom meal service with repository dependencies.
func NewService(customMealrepo interfaces.CustomMealRepository) interfaces.CustomMealService {
	return &service{
		customMealRepo: customMealrepo,
	}
}

// Create validates user input, builds a custom meal model, persists it, and
// returns the saved meal as an API response.
func (s *service) Create(ctx context.Context, userID uuid.UUID, input interfaces.CustomMealInput) (*interfaces.CustomMealResponse, error) {
	if err := validateCustomMealInput(input); err != nil {
		return nil, err
	}

	meal := buildCustomMealModel(userID, input)

	savedMeal, err := s.customMealRepo.Create(ctx, meal)
	if err != nil {
		return nil, err
	}

	return buildCustomMealResponse(savedMeal, userID), nil
}

// ListVisible returns custom meals visible to the user, including their own
// meals and shared meals from other users.
func (s *service) ListVisible(ctx context.Context, userID uuid.UUID, query string) ([]*interfaces.CustomMealResponse, error) {
	ownedMeals, err := s.customMealRepo.ListOwnedByUser(ctx, userID, query)
	if err != nil {
		return nil, err
	}

	sharedMeals, err := s.customMealRepo.ListSharedFromOtherUsers(ctx, userID, query)
	if err != nil {
		return nil, err
	}

	responses := make([]*interfaces.CustomMealResponse, 0, len(ownedMeals)+len(sharedMeals))

	for _, meal := range ownedMeals {
		mealCopy := meal
		responses = append(responses, buildCustomMealResponse(&mealCopy, userID))
	}

	for _, meal := range sharedMeals {
		mealCopy := meal
		responses = append(responses, buildCustomMealResponse(&mealCopy, userID))
	}

	return responses, nil
}

// FindVisibleByID returns a single custom meal if the user owns it or the meal
// owner's sharing consent makes it visible.
func (s *service) FindVisibleByID(ctx context.Context, userID uuid.UUID, customMealID uuid.UUID) (*interfaces.CustomMealResponse, error) {
	meal, err := s.customMealRepo.FindVisibleByID(ctx, userID, customMealID)
	if err != nil {
		return nil, err
	}

	return buildCustomMealResponse(meal, userID), nil
}

var allowedDietaryRestrictions = map[string]bool{
	"halal":        true,
	"non_beef":     true,
	"vegetarian":   true,
	"vegan":        true,
	"seafood_free": true,
	"nut_free":     true,
	"low_salt":     true,
	"low_sugar":    true,
	"low_fat":      true,
}

var allowedMealPreferenceTags = map[string]bool{
	"american":       true,
	"basics":         true,
	"chinese":        true,
	"condiments":     true,
	"desserts":       true,
	"drinks":         true,
	"french":         true,
	"fruits":         true,
	"grains":         true,
	"greek":          true,
	"healthy":        true,
	"indian":         true,
	"italian":        true,
	"japanese":       true,
	"korean":         true,
	"kuih":           true,
	"legumes":        true,
	"meat":           true,
	"mexican":        true,
	"middle_eastern": true,
	"noodles":        true,
	"nuts":           true,
	"proteins":       true,
	"rice":           true,
	"roti":           true,
	"seafood":        true,
	"seeds":          true,
	"snacks":         true,
	"soups":          true,
	"spanish":        true,
	"thai":           true,
	"vegetables":     true,
	"vietnamese":     true,
	"western":        true,
}

// Update validates input, rebuilds the custom meal model, and updates it only
// when the meal belongs to the authenticated user.
func (s *service) Update(ctx context.Context, userID uuid.UUID, customMealID uuid.UUID, input interfaces.CustomMealInput) (*interfaces.CustomMealResponse, error) {
	if err := validateCustomMealInput(input); err != nil {
		return nil, err
	}

	meal := buildCustomMealModel(userID, input)
	meal.ID = customMealID

	updatedMeal, err := s.customMealRepo.UpdateOwned(ctx, userID, meal)
	if err != nil {
		return nil, err
	}

	return buildCustomMealResponse(updatedMeal, userID), nil
}

// Delete removes a custom meal only when it belongs to the authenticated user.
func (s *service) Delete(ctx context.Context, userID uuid.UUID, customMealID uuid.UUID) error {
	return s.customMealRepo.DeleteOwned(ctx, userID, customMealID)
}

// validateCustomMealInput enforces required fields, non-negative nutrition
// values, and supported tag values before a custom meal is saved.
func validateCustomMealInput(input interfaces.CustomMealInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return errors.New("custom meal name is required")
	}

	if input.Price < 0 {
		return errors.New("price cannot be negative")
	}

	if input.Calories < 0 {
		return errors.New("calories cannot be negative")
	}

	if input.FatG < 0 {
		return errors.New("fat cannot be negative")
	}

	if input.ProteinG < 0 {
		return errors.New("protein cannot be negative")
	}

	if input.CarbsG < 0 {
		return errors.New("carbs cannot be negative")
	}

	if strings.TrimSpace(input.State) == "" {
		return errors.New("state is required")
	}

	if strings.TrimSpace(input.District) == "" {
		return errors.New("district is required")
	}

	if strings.TrimSpace(input.RestaurantName) == "" {
		return errors.New("restaurant name is required")
	}

	for _, restriction := range input.DietaryRestrictionTags {
		if !allowedDietaryRestrictions[restriction] {
			return errors.New("unsupported dietary restriction tag: " + restriction)
		}
	}

	for _, tag := range input.MealCategoryTags {
		if !allowedMealPreferenceTags[tag] {
			return errors.New("unsupported meal preference tag: " + tag)
		}
	}

	return nil
}

// buildCustomMealModel converts validated custom meal input into the database
// model, including dietary restriction and meal category tag rows.
func buildCustomMealModel(userID uuid.UUID, input interfaces.CustomMealInput) *model.CustomMealItem {
	meal := &model.CustomMealItem{
		Name:           strings.TrimSpace(input.Name),
		Price:          input.Price,
		Calories:       input.Calories,
		FatG:           input.FatG,
		ProteinG:       input.ProteinG,
		CarbsG:         input.CarbsG,
		State:          strings.TrimSpace(input.State),
		District:       strings.TrimSpace(input.District),
		RestaurantName: strings.TrimSpace(input.RestaurantName),
		CreatedBy:      userID,
	}

	for _, restriction := range input.DietaryRestrictionTags {
		meal.DietaryRestrictionTags = append(meal.DietaryRestrictionTags, model.CustomMealDietaryRestrictionTag{
			DietaryRestrictionTag: restriction,
		})
	}

	for _, tag := range input.MealCategoryTags {
		meal.MealCategoryTags = append(meal.MealCategoryTags, model.CustomMealCategoryTag{
			MealCategory: tag,
		})
	}

	return meal
}

// buildCustomMealResponse converts a custom meal model into the API response
// shape and marks whether the current user owns or views it as shared.
func buildCustomMealResponse(meal *model.CustomMealItem, userID uuid.UUID) *interfaces.CustomMealResponse {
	response := &interfaces.CustomMealResponse{
		ID:             meal.ID.String(),
		Name:           meal.Name,
		Price:          meal.Price,
		Calories:       meal.Calories,
		FatG:           meal.FatG,
		ProteinG:       meal.ProteinG,
		CarbsG:         meal.CarbsG,
		State:          meal.State,
		District:       meal.District,
		RestaurantName: meal.RestaurantName,
		IsOwner:        meal.CreatedBy == userID,
		IsShared:       meal.CreatedBy != userID,
	}

	for _, restriction := range meal.DietaryRestrictionTags {
		response.DietaryRestrictionTags = append(response.DietaryRestrictionTags, restriction.DietaryRestrictionTag)
	}

	for _, tag := range meal.MealCategoryTags {
		response.MealCategoryTags = append(response.MealCategoryTags, tag.MealCategory)
	}

	return response
}
