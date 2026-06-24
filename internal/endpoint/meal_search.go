package endpoint

import (
	"context"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/service/mealdataset"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type prebuiltMealSearcher interface {
	Search(ctx context.Context, query string) ([]mealdataset.PrebuiltMeal, error)
}

type mealSearchHandler struct {
	customMealService interfaces.CustomMealService
	prebuiltSearcher  prebuiltMealSearcher
}

type mealSearchResult struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Source   string   `json:"source"`
	Tags     []string `json:"tags"`
	Calories float64  `json:"calories"`
	FatG     float64  `json:"fat_g"`
	ProteinG float64  `json:"protein_g"`
	CarbsG   float64  `json:"carbs_g"`
	Price    float64  `json:"price,omitempty"`
	ImageURL string   `json:"image_url,omitempty"`
}

func RegisterMealSearchRoutes(ctx context.Context, e *echo.Echo) {
	a := app.FromContext(ctx)

	if a == nil {
		log.Fatal("app missing from context")
		return
	}

	customMealService, err := a.GetCustomMealService(ctx)
	if err != nil {
		log.Fatal("failed to get custom meal service", "error", err)
	}

	prebuiltSearcher, err := mealdataset.NewPrebuiltSearcher("internal/data/prebuilt_meals.json")
	if err != nil {
		log.Fatal("failed to load prebuilt meal searcher", "error", err)
	}

	h := &mealSearchHandler{
		customMealService: customMealService,
		prebuiltSearcher:  prebuiltSearcher,
	}

	meals := e.Group("/meals", middleware.Auth(a.JWTSecret))
	meals.GET("/search", h.searchMeals)
}

func (h *mealSearchHandler) searchMeals(c *echo.Context) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	query := strings.TrimSpace(c.QueryParam("q"))
	if query == "" {
		return c.JSON(http.StatusOK, []mealSearchResult{})
	}

	customMeals, err := h.customMealService.ListVisible(c.Request().Context(), userID, query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}
	if len(customMeals) == 0 {
		customMeals, err = h.fuzzyCustomMealSearch(c.Request().Context(), userID, query)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	}

	prebuiltMeals, err := h.prebuiltSearcher.Search(c.Request().Context(), query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	results := make([]mealSearchResult, 0, len(customMeals)+len(prebuiltMeals))
	for _, meal := range customMeals {
		results = append(results, customMealSearchResult(meal))
	}

	for _, meal := range prebuiltMeals {
		results = append(results, prebuiltMealSearchResult(meal))
	}

	return c.JSON(http.StatusOK, results)
}

func (h *mealSearchHandler) fuzzyCustomMealSearch(ctx context.Context, userID uuid.UUID, query string) ([]*interfaces.CustomMealResponse, error) {
	customMeals, err := h.customMealService.ListVisible(ctx, userID, "")
	if err != nil {
		return nil, err
	}

	matches := make([]*interfaces.CustomMealResponse, 0)
	for _, meal := range customMeals {
		if mealdataset.FuzzyMealNameMatch(meal.Name, query) {
			matches = append(matches, meal)
		}
	}

	return matches, nil
}

func customMealSearchResult(meal *interfaces.CustomMealResponse) mealSearchResult {
	return mealSearchResult{
		ID:       meal.ID,
		Name:     meal.Name,
		Source:   "custom",
		Tags:     append([]string{}, meal.DietaryRestrictionTags...),
		Calories: meal.Calories,
		FatG:     meal.FatG,
		ProteinG: meal.ProteinG,
		CarbsG:   meal.CarbsG,
		Price:    meal.Price,
		ImageURL: meal.ImageURL,
	}
}

func prebuiltMealSearchResult(meal mealdataset.PrebuiltMeal) mealSearchResult {
	return mealSearchResult{
		ID:       mealdataset.PrebuiltMealID(meal),
		Name:     meal.Name,
		Source:   "prebuilt",
		Tags:     []string(meal.Category),
		Calories: meal.Calories,
		FatG:     meal.Fat,
		ProteinG: meal.Protein,
		CarbsG:   meal.Carbs,
		ImageURL: meal.ImageURL,
	}
}

func init() {
	endpoints = append(endpoints, RegisterMealSearchRoutes)
}
