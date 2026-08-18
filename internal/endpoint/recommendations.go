package endpoint

import (
	"context"
	"errors"
	"fmt"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"log"
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

type recommendationHandler struct {
	recommendationService interfaces.RecommendationService
}

type generateRecommendationRequest struct {
	MealCategory      string  `json:"mealCategory"`
	CurrentMonthSpent float64 `json:"currentMonthSpent"`
	PerMealBudget     float64 `json:"perMealBudget"`
	Location          string  `json:"location"`
}

var allowedRecommendationMealCategories = map[string]bool{
	"breakfast": true,
	"lunch":     true,
	"dinner":    true,
	"snack":     true,
}

func RegisterRecommendationRoutes(ctx context.Context, e *echo.Echo) {
	a := app.FromContext(ctx)
	if a == nil {
		log.Fatal("app missing from context")
		return
	}

	recommendationService, err := a.GetRecommendationService(ctx)
	if err != nil {
		log.Fatal("failed to get recommendation service", "error", err)
		return
	}

	h := &recommendationHandler{
		recommendationService: recommendationService,
	}

	recommendations := e.Group("/recommendations", middleware.Auth(a.JWTSecret))
	recommendations.POST("", h.generateRecommendations)
	recommendations.POST("/meal-detail", h.getMealDetail)
}

func (h *recommendationHandler) generateRecommendations(c *echo.Context) error {
	var request generateRecommendationRequest

	if err := c.Bind(&request); err != nil {
		slog.Warn("recommendation request rejected: invalid body", "error", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if err := validateGenerateRecommendationRequest(request); err != nil {
		slog.Warn("recommendation request rejected: validation failed",
			"meal_category", request.MealCategory,
			"current_month_spent", request.CurrentMonthSpent,
			"per_meal_budget", request.PerMealBudget,
			"error", err,
		)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		slog.Warn("recommendation request rejected: unauthorized", "error", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	slog.Info("recommendation request started",
		"user_id", userID,
		"meal_category", strings.TrimSpace(request.MealCategory),
		"location", strings.TrimSpace(request.Location),
		"current_month_spent", request.CurrentMonthSpent,
		"per_meal_budget", request.PerMealBudget,
	)

	input := interfaces.RecommendationRequestInput{
		MealCategory:      strings.TrimSpace(request.MealCategory),
		CurrentMonthSpent: request.CurrentMonthSpent,
		PerMealBudget:     request.PerMealBudget,
		Location:          strings.TrimSpace(request.Location),
	}

	result, err := h.recommendationService.GenerateRecommendationResult(c.Request().Context(), userID, input)
	if err != nil {
		slog.Error("recommendation request failed: generate candidates",
			"user_id", userID,
			"meal_category", input.MealCategory,
			"error", err,
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to generate recommendations",
		})
	}

	slog.Info("recommendation request completed",
		"user_id", userID,
		"meal_category", input.MealCategory,
		"candidate_count", len(result.Candidates),
		"filtered_out_count", len(result.FilteredOut),
	)

	return c.JSON(http.StatusOK, result)
}

func validateGenerateRecommendationRequest(input generateRecommendationRequest) error {
	mealCategory := strings.TrimSpace(input.MealCategory)
	if mealCategory == "" {
		return errors.New("meal category is required")
	}

	if !allowedRecommendationMealCategories[mealCategory] {
		return fmt.Errorf("unsupported meal category: %s", mealCategory)
	}

	if input.CurrentMonthSpent < 0 {
		return errors.New("current month spent cannot be negative")
	}

	if input.PerMealBudget < 0 {
		return errors.New("per meal budget cannot be negative")
	}

	return nil
}

func init() {
	endpoints = append(endpoints, RegisterRecommendationRoutes)
}
