package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type CustomMealInput struct {
	Name                   string   `json:"name"`
	Price                  float64  `json:"price"`
	Calories               float64  `json:"calories"`
	FatG                   float64  `json:"fatG"`
	ProteinG               float64  `json:"proteinG"`
	CarbsG                 float64  `json:"carbsG"`
	State                  string   `json:"state"`
	District               string   `json:"district"`
	RestaurantName         string   `json:"restaurantName"`
	DietaryRestrictionTags []string `json:"dietaryRestrictionTags"`
	MealCategoryTags       []string `json:"mealCategoryTags"`
}

type CustomMealResponse struct {
	ID                     string   `json:"id"`
	Name                   string   `json:"name"`
	Price                  float64  `json:"price"`
	Calories               float64  `json:"calories"`
	FatG                   float64  `json:"fatG"`
	ProteinG               float64  `json:"proteinG"`
	CarbsG                 float64  `json:"carbsG"`
	State                  string   `json:"state"`
	District               string   `json:"district"`
	RestaurantName         string   `json:"restaurantName"`
	DietaryRestrictionTags []string `json:"dietaryRestrictionTags"`
	MealCategoryTags       []string `json:"mealCategoryTags"`
	// Only owner can edit/delete, non-owner can only view/log meal
	IsOwner bool `json:"isOwner"`
	// IsShared is true when the current user is viewing another user's custom meal
	// that passed sharing-consent visibility checks. It is false for the owner's
	// own meals.
	IsShared bool `json:"isShared"`
}

type CustomMealService interface {
	Create(ctx context.Context, userID uuid.UUID, input CustomMealInput) (*CustomMealResponse, error)
	ListVisible(ctx context.Context, userID uuid.UUID, query string) ([]*CustomMealResponse, error)
	FindVisibleByID(ctx context.Context, userID uuid.UUID, customMealID uuid.UUID) (*CustomMealResponse, error)
}
