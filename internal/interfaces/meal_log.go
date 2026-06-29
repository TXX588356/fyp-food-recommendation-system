package interfaces

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type MealLogSource string

const (
	MealLogSourceCustom   MealLogSource = "custom"
	MealLogSourcePrebuilt MealLogSource = "prebuilt"
)

type MealLogInput struct {
	Source  MealLogSource `json:"source"`
	MealID  string        `json:"mealId"`
	Price   float64       `json:"price"`
	EatenAt time.Time     `json:"eatenAt"`
}

type MealLogResponse struct {
	ID               string    `json:"id"`
	CustomMealItemID *string   `json:"customMealItemId,omitempty"`
	PrebuiltMealID   *string   `json:"prebuiltMealId,omitempty"`
	EatenAt          time.Time `json:"eatenAt"`
	MealName         string    `json:"mealName"`
	Calories         float64   `json:"calories"`
	Price            float64   `json:"price"`
	MealCategory     []string  `json:"mealCategory"`
}

type MealLogMonthSummary struct {
	Month               string   `json:"month"`
	TotalMealsEaten     int      `json:"totalMealsEaten"`
	TotalSpent          float64  `json:"totalSpent"`
	TotalCalories       float64  `json:"totalCalories"`
	BudgetRemaining     *float64 `json:"budgetRemaining,omitempty"`
	ShowBudgetRemaining bool     `json:"showBudgetRemaining"`
}

type MealLogMonthResponse struct {
	Summary MealLogMonthSummary `json:"summary"`
	Items   []MealLogResponse   `json:"items"`
}

type MealLogService interface {
	Create(ctx context.Context, userID uuid.UUID, input MealLogInput) (*MealLogResponse, error)
	GetMonth(ctx context.Context, userID uuid.UUID, month string) (*MealLogResponse, error)
}
