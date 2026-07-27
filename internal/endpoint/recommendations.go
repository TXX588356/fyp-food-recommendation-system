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
	"time"

	"github.com/labstack/echo/v5"
)

type recommendationHandler struct {
	preferenceService     interfaces.PreferenceService
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

	preferenceService, err := a.GetPreferenceService(ctx)
	if err != nil {
		log.Fatal("failed to get preference service", "error", err)
		return
	}

	recommendationService, err := a.GetRecommendationService(ctx)
	if err != nil {
		log.Fatal("failed to get recommendation service", "error", err)
		return
	}

	h := &recommendationHandler{
		preferenceService:     preferenceService,
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

	// Load saved preferences for the user
	preferences, err := h.preferenceService.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		slog.Error("recommendation request failed: load preferences",
			"user_id", userID,
			"error", err,
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to load preferences",
		})
	}

	// Build meal prompt input
	input := buildMealPromptFromPreferences(*preferences, request)
	slog.Info("recommendation prompt input built",
		"user_id", userID,
		"goal", input.Goal,
		"meal_category", input.MealCategory,
		"dietary_restrictions_count", len(input.DietaryRestrictions),
		"health_concerns_count", len(input.HealthConcerns),
		"preferred_tags_count", len(input.PreferredMealTags),
		"monthly_budget", input.MonthlyMealBudget,
		"remaining_budget", input.RemainingBudget,
		"per_meal_budget", input.PerMealBudget,
	)

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

func buildMealPromptFromPreferences(preferences interfaces.PreferenceResponse, request generateRecommendationRequest) interfaces.MealPromptInput {
	remainingBudget := preferences.MonthlyMealBudget - request.CurrentMonthSpent
	if remainingBudget < 0 {
		remainingBudget = 0
	}

	perMealBudget := request.PerMealBudget
	if perMealBudget <= 0 {
		perMealBudget = calculateDynamicPerMealBudget(preferences.MonthlyMealBudget, request.CurrentMonthSpent, time.Now())
	}

	dayOfTheWeek := time.Now().Weekday()

	location := strings.TrimSpace(request.Location)
	if location != "" {
		// User selected a temporary recommendation location.
	} else if dayOfTheWeek == 0 || dayOfTheWeek == 6 {
		location = preferences.HomeLocation
	} else {
		location = preferences.WorkSchoolLocation
	}

	log.Printf(
		"data used to build prompt: \nGoal: %s\nDiet Restriction: %v\nConcern: %v\nPreferredMeal: %v\nMeal Category: %s\nBudget: %.2f \nCurrent Spent: %.2f\nRemaning Budget: %.2f\nPer Meal Budget: %.2f\nMarket Location: %s\n",
		preferences.MainGoal,
		preferences.DietaryRestrictions,
		preferences.HealthConcerns,
		preferences.PreferredMealTags,
		strings.TrimSpace(request.MealCategory),
		preferences.MonthlyMealBudget,
		request.CurrentMonthSpent,
		remainingBudget,
		perMealBudget,
		location,
	)

	return interfaces.MealPromptInput{
		Goal:                preferences.MainGoal,
		DietaryRestrictions: preferences.DietaryRestrictions,
		HealthConcerns:      preferences.HealthConcerns,
		PreferredMealTags:   preferences.PreferredMealTags,
		MealCategory:        strings.TrimSpace(request.MealCategory),
		MonthlyMealBudget:   preferences.MonthlyMealBudget,
		CurrentMonthSpent:   request.CurrentMonthSpent,
		RemainingBudget:     remainingBudget,
		PerMealBudget:       perMealBudget,
		PriceMarketLocation: location,
	}
}

// calculateDynamicPerMealBudget estimates a per-meal budget from the remaining
// monthly budget and days left in the current month.
func calculateDynamicPerMealBudget(
	monthlyBudget float64,
	currentMonthSpent float64,
	now time.Time,
) float64 {
	remainingBudget := monthlyBudget - currentMonthSpent
	if remainingBudget <= 0 {
		return 0
	}

	firstOfNextMonth := time.Date(
		now.Year(),
		now.Month()+1,
		1,
		0,
		0,
		0,
		0,
		now.Location(),
	)

	todayStart := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		0,
		0,
		0,
		0,
		now.Location(),
	)

	daysRemaining := int(firstOfNextMonth.Sub(todayStart).Hours() / 24)
	if daysRemaining < 1 {
		daysRemaining = 1
	}

	const plannedMealsPerDay = 3
	return remainingBudget / float64(daysRemaining*plannedMealsPerDay)
}

func init() {
	endpoints = append(endpoints, RegisterRecommendationRoutes)
}
