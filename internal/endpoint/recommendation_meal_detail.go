package endpoint

import (
	"errors"
	"fmt"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
)

type mealDetailRequest struct {
	MealCategory string                          `json:"mealCategory"`
	Candidate    interfaces.MatchedMealCandidate `json:"candidate"`
	Location     string                          `json:"location"`
}

type mealDetailResponse struct {
	Meal                      mealDetailMealResponse     `json:"meal"`
	RecommendationExplanation string                     `json:"recommendationExplanation"`
	Location                  mealDetailLocationResponse `json:"location"`
	Restaurants               []restaurantResponse       `json:"restaurants"`
	RestaurantLookupStatus    string                     `json:"restaurantLookupStatus"`
}

type mealDetailLocationResponse struct {
	Query string `json:"query"`
	Basis string `json:"basis"`
}

type restaurantResponse struct {
	Name         string  `json:"name"`
	Address      string  `json:"address"`
	Rating       float64 `json:"rating"`
	ReviewCount  int     `json:"reviewCount"`
	Price        string  `json:"price"`
	OpenNow      *bool   `json:"openNow,omitempty"`
	ThumbnailURL string  `json:"thumbnailUrl,omitempty"`
	SourceURL    string  `json:"sourceUrl,omitempty"`
	Source       string  `json:"source,omitempty"`
}

type mealDetailMealResponse struct {
	ID                  string                `json:"id"`
	Name                string                `json:"name"`
	MealCategory        string                `json:"mealCategory"`
	ImageURL            string                `json:"imageUrl,omitempty"`
	ServingDescription  string                `json:"servingDescription,omitempty"`
	EstimatedPriceRange interfaces.PriceRange `json:"estimatedPriceRange"`
	Nutrition           mealDetailNutrition   `json:"nutrition"`
	Signals             mealDetailSignals     `json:"signals"`
}

type mealDetailNutrition struct {
	Calories float64 `json:"calories"`
	FatG     float64 `json:"fatG"`
	ProteinG float64 `json:"proteinG"`
	CarbsG   float64 `json:"carbsG"`
}

type mealDetailSignals struct {
	SodiumLevel string            `json:"sodiumLevel"`
	SugarLevel  string            `json:"sugarLevel"`
	PurineRisk  string            `json:"purineRisk"`
	HealthFlags map[string]string `json:"healthFlags"`
}

// getMealDetail returns display-ready meal details plus a Gemini explanation
// for why the selected recommendation fits the authenticated user.
func (h *recommendationHandler) getMealDetail(c *echo.Context) error {
	var request mealDetailRequest

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if err := validateMealDetailRequest(request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	result, err := h.recommendationService.BuildMealDetail(c.Request().Context(), userID, interfaces.MealDetailInput{
		MealCategory: strings.TrimSpace(request.MealCategory),
		Candidate:    request.Candidate,
		Location:     strings.TrimSpace(request.Location),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to build meal detail",
		})
	}

	return c.JSON(http.StatusOK, buildMealDetailResponse(result))
}

func validateMealDetailRequest(input mealDetailRequest) error {
	mealCategory := strings.TrimSpace(input.MealCategory)

	if mealCategory == "" {
		return errors.New("meal category is required")
	}

	if !allowedRecommendationMealCategories[mealCategory] {
		return fmt.Errorf("unsupported meal category: %s", mealCategory)
	}

	food := input.Candidate.Food

	if strings.TrimSpace(food.ID) == "" {
		return errors.New("food id is required")
	}

	if strings.TrimSpace(food.Name) == "" {
		return errors.New("food name is required")
	}

	if food.Calories < 0 {
		return errors.New("calories cannot be negative")
	}

	if food.FatG < 0 {
		return errors.New("fat cannot be negative")
	}

	if food.ProteinG < 0 {
		return errors.New("protein cannot be negative")
	}

	if food.CarbsG < 0 {
		return errors.New("carbs cannot be negative")
	}

	generated := input.Candidate.GeneratedMeal
	priceRange := input.Candidate.GeneratedMeal.EstimatedPriceRange

	if priceRange.Min < 0 {
		return errors.New("minimum price cannot be negative")
	}

	if priceRange.Max < 0 {
		return errors.New("maximum price cannot be negative")
	}

	if priceRange.Min > priceRange.Max {
		return errors.New("minimum price cannot exceed maximum price")
	}

	if err := validateOptionalRiskLevel("sodium level", generated.SodiumLevel); err != nil {
		return err
	}

	if err := validateOptionalRiskLevel("sugar level", generated.SugarLevel); err != nil {
		return err
	}

	if err := validateOptionalRiskLevel("purine risk", generated.PurineRisk); err != nil {
		return err
	}

	if err := validateMealDetailHealthFlags(generated.HealthFlags); err != nil {
		return err
	}

	return nil
}

var allowedMealDetailRiskLevels = map[string]bool{
	"LOW":    true,
	"MEDIUM": true,
	"HIGH":   true,
}

var allowedMealDetailHealthFlags = map[string]bool{
	"SAFE":    true,
	"CAUTION": true,
	"AVOID":   true,
}

func validateOptionalRiskLevel(fieldName, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	if !allowedMealDetailRiskLevels[value] {
		return fmt.Errorf("unsupported %s: %s", fieldName, value)

	}
	return nil
}

func validateMealDetailHealthFlags(healthFlags map[string]string) error {
	for condition, flag := range healthFlags {
		condition = strings.TrimSpace(condition)
		flag = strings.TrimSpace(flag)

		if condition == "" {
			return errors.New("health flag condition cannot be blank")
		}

		if flag == "" {
			return fmt.Errorf("health flag for %s cannot be blank", condition)
		}

		if !allowedMealDetailHealthFlags[flag] {
			return fmt.Errorf("unsupported health flag for %s: %s", condition, flag)
		}
	}

	return nil
}

// buildMealDetailResponse maps service meal detail output into the HTTP response shape.
func buildMealDetailResponse(result interfaces.MealDetailResult) mealDetailResponse {
	return mealDetailResponse{
		Meal: mealDetailMealResponse{
			ID:                  result.Meal.ID,
			Name:                result.Meal.Name,
			MealCategory:        result.Meal.MealCategory,
			ImageURL:            result.Meal.ImageURL,
			ServingDescription:  result.Meal.ServingDescription,
			EstimatedPriceRange: result.Meal.EstimatedPriceRange,
			Nutrition: mealDetailNutrition{
				Calories: result.Meal.Nutrition.Calories,
				FatG:     result.Meal.Nutrition.FatG,
				ProteinG: result.Meal.Nutrition.ProteinG,
				CarbsG:   result.Meal.Nutrition.CarbsG,
			},
			Signals: mealDetailSignals{
				SodiumLevel: result.Meal.Signals.SodiumLevel,
				SugarLevel:  result.Meal.Signals.SugarLevel,
				PurineRisk:  result.Meal.Signals.PurineRisk,
				HealthFlags: result.Meal.Signals.HealthFlags,
			},
		},
		RecommendationExplanation: result.RecommendationExplanation,
		Location: mealDetailLocationResponse{
			Query: result.Location.Query,
			Basis: result.Location.Basis,
		},
		Restaurants:            buildRestaurantResponses(result.Restaurants),
		RestaurantLookupStatus: result.RestaurantLookupStatus,
	}
}

func buildRestaurantResponses(restaurants []interfaces.RestaurantResult) []restaurantResponse {
	response := make([]restaurantResponse, 0, len(restaurants))
	for _, restaurant := range restaurants {
		response = append(response, restaurantResponse{
			Name:         restaurant.Name,
			Address:      restaurant.Address,
			Rating:       restaurant.Rating,
			ReviewCount:  restaurant.ReviewCount,
			Price:        restaurant.Price,
			OpenNow:      restaurant.OpenNow,
			ThumbnailURL: restaurant.ThumbnailURL,
			SourceURL:    restaurant.SourceURL,
			Source:       restaurant.Source,
		})
	}

	return response
}

// copyTrimmedStringSlice returns a clean copy of a string slice without blank entries.
func copyTrimmedStringSlice(values []string) []string {
	cleaned := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cleaned = append(cleaned, value)
	}

	return cleaned
}

// copyTrimmedMealDetailHealthFlags returns a clean copy of health flags for prompt input.
func copyTrimmedMealDetailHealthFlags(values map[string]string) map[string]string {
	cleaned := map[string]string{}

	for condition, flag := range values {
		condition = strings.TrimSpace(condition)
		flag = strings.TrimSpace(flag)

		if condition == "" || flag == "" {
			continue
		}

		cleaned[condition] = flag
	}

	return cleaned
}

// selectMealDetailPreferenceLocation chooses the saved location used for meal detail context.
// Weekdays prefer work/school, while weekends prefer home. If the preferred
// location is blank, it falls back to the other saved location.
func selectMealDetailPreferenceLocation(preferences interfaces.PreferenceResponse, now time.Time) (string, string) {
	homeLocation := strings.TrimSpace(preferences.HomeLocation)
	workSchoolLocation := strings.TrimSpace(preferences.WorkSchoolLocation)

	isWeekend := now.Weekday() == time.Saturday || now.Weekday() == time.Sunday
	if isWeekend {
		if homeLocation != "" {
			return homeLocation, "home"
		}

		if workSchoolLocation != "" {
			return workSchoolLocation, "fallback_work_school"
		}

		return "", "unavailable"
	}

	if workSchoolLocation != "" {
		return workSchoolLocation, "work_school"
	}

	if homeLocation != "" {
		return homeLocation, "fallback_home"
	}

	return "", "unavailable"
}
