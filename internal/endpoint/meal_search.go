package endpoint

import (
	"context"
	"errors"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/service/mealdataset"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"gorm.io/gorm"
)

type mealSearchHandler struct {
	customMealService  interfaces.CustomMealService
	catalogService     interfaces.CatalogService
	preferenceService  interfaces.PreferenceService
	restaurantSearcher interfaces.RestaurantSearcher
	imageStorage       interfaces.ImageStorage
}

type mealSearchResult struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Source             string   `json:"source"`
	Tags               []string `json:"tags"`
	Calories           float64  `json:"calories"`
	FatG               float64  `json:"fat_g"`
	ProteinG           float64  `json:"protein_g"`
	CarbsG             float64  `json:"carbs_g"`
	Price              float64  `json:"price,omitempty"`
	ServingDescription string   `json:"serving_description,omitempty"`
	ImageURL           string   `json:"image_url,omitempty"`
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

	catalogService, err := a.GetCatalogService(ctx)
	if err != nil {
		log.Fatal("failed to get meal catalogue service", "error", err)
	}

	preferenceService, err := a.GetPreferenceService(ctx)
	if err != nil {
		log.Fatal("failed to get preference service", "error", err)
	}

	restaurantSearcher, err := a.GetRestaurantSearcher(ctx)
	if err != nil {
		log.Fatal("failed to get restaurant searcher", "error", err)
	}

	h := &mealSearchHandler{
		customMealService:  customMealService,
		catalogService:     catalogService,
		preferenceService:  preferenceService,
		restaurantSearcher: restaurantSearcher,
		imageStorage:       a.ImageStorage,
	}

	meals := e.Group("/api/meals", middleware.Auth(a.JWTSecret))
	meals.GET("/search", h.searchMeals)
	meals.GET("/:source/:mealID", h.getMealDetail)
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

	prebuiltPage, err := h.catalogService.SearchMeals(c.Request().Context(), interfaces.CatalogQuery{
		Query: query,
		Limit: 100,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	results := make([]mealSearchResult, 0, len(customMeals)+len(prebuiltPage.Items))
	for _, meal := range customMeals {
		h.signCustomMealImageURL(c.Request().Context(), meal)
		results = append(results, customMealSearchResult(meal))
	}

	for _, meal := range prebuiltPage.Items {
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

func (h *mealSearchHandler) signCustomMealImageURL(ctx context.Context, meal *interfaces.CustomMealResponse) {
	if meal == nil || meal.ImageURL == "" || h.imageStorage == nil {
		return
	}

	signedURL, err := h.imageStorage.PresignMealImage(ctx, meal.ImageURL)
	if err != nil {
		log.Printf("custom meal image presign failed: %v", err)
		return
	}

	meal.ImageURL = signedURL
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

func prebuiltMealSearchResult(meal interfaces.CatalogMeal) mealSearchResult {
	nutrition := meal.SelectedNutrition
	imageURL := ""
	if meal.Image != nil {
		imageURL = meal.Image.URL
	}

	return mealSearchResult{
		ID:                 meal.ID.String(),
		Name:               meal.Name,
		Source:             "prebuilt",
		Tags:               meal.Categories,
		Calories:           float64Value(nutrition.Calories),
		FatG:               float64Value(nutrition.FatG),
		ProteinG:           float64Value(nutrition.ProteinG),
		CarbsG:             float64Value(nutrition.CarbsG),
		ServingDescription: meal.SelectedPortion.Description,
		ImageURL:           imageURL,
	}
}

type manualMealDetailResponse struct {
	Meal                   manualMealDetailMealResponse `json:"meal"`
	Location               mealDetailLocationResponse   `json:"location"`
	Restaurants            []restaurantResponse         `json:"restaurants"`
	RestaurantLookupStatus string                       `json:"restaurantLookupStatus"`
}

type manualMealDetailMealResponse struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Source             string              `json:"source"`
	ImageURL           string              `json:"imageUrl,omitempty"`
	ServingDescription string              `json:"servingDescription,omitempty"`
	Price              float64             `json:"price,omitempty"`
	Tags               []string            `json:"tags"`
	Nutrition          mealDetailNutrition `json:"nutrition"`
}

func (h *mealSearchHandler) getMealDetail(c *echo.Context) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	source := strings.TrimSpace(c.Param("source"))
	mealID, err := uuid.Parse(strings.TrimSpace(c.Param("mealID")))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid meal id",
		})
	}

	var meal manualMealDetailMealResponse
	var customMeal *interfaces.CustomMealResponse

	switch source {
	case "prebuilt":
		catalogMeal, err := h.catalogService.GetMeal(c.Request().Context(), mealID)
		if err != nil {
			if errors.Is(err, interfaces.ErrCatalogNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "meal not found",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to load meal detail",
			})
		}
		meal = manualPrebuiltMealDetail(catalogMeal)
	case "custom":
		customMeal, err = h.customMealService.FindVisibleByID(c.Request().Context(), userID, mealID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": "meal not found",
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "failed to load meal detail",
			})
		}
		h.signCustomMealImageURL(c.Request().Context(), customMeal)
		meal = manualCustomMealDetail(customMeal)
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "unsupported meal source",
		})
	}

	locationOverride := strings.TrimSpace(c.QueryParam("location"))
	location, basis, restaurants, status := h.lookupManualMealRestaurants(c.Request().Context(), userID, meal.Name, locationOverride)

	if customMeal != nil && customMealStateMatchesLocation(customMeal.State, location) {
		if restaurantName := strings.TrimSpace(customMeal.RestaurantName); restaurantName != "" {
			customRestaurant := restaurantResponse{
				Name: restaurantName,
				Address: strings.TrimSpace(strings.Join([]string{
					customMeal.District,
					customMeal.State,
				}, ", ")),
				Source: "user_recommended",
			}

			restaurants = append([]restaurantResponse{customRestaurant}, restaurants...)
		}
	}

	return c.JSON(http.StatusOK, manualMealDetailResponse{
		Meal: meal,
		Location: mealDetailLocationResponse{
			Query: location,
			Basis: basis,
		},
		Restaurants:            restaurants,
		RestaurantLookupStatus: status,
	})
}

func manualPrebuiltMealDetail(meal interfaces.CatalogMeal) manualMealDetailMealResponse {
	nutrition := meal.SelectedNutrition
	imageURL := ""
	if meal.Image != nil {
		imageURL = strings.TrimSpace(meal.Image.URL)
	}

	return manualMealDetailMealResponse{
		ID:                 meal.ID.String(),
		Name:               meal.Name,
		Source:             "prebuilt",
		ImageURL:           imageURL,
		ServingDescription: meal.SelectedPortion.Description,
		Tags:               append([]string{}, meal.Categories...),
		Nutrition: mealDetailNutrition{
			Calories: float64Value(nutrition.Calories),
			FatG:     float64Value(nutrition.FatG),
			ProteinG: float64Value(nutrition.ProteinG),
			CarbsG:   float64Value(nutrition.CarbsG),
		},
	}
}

func manualCustomMealDetail(meal *interfaces.CustomMealResponse) manualMealDetailMealResponse {
	tags := append([]string{}, meal.DietaryRestrictionTags...)
	tags = append(tags, meal.MealCategoryTags...)

	return manualMealDetailMealResponse{
		ID:       meal.ID,
		Name:     meal.Name,
		Source:   "custom",
		ImageURL: meal.ImageURL,
		Price:    meal.Price,
		Tags:     tags,
		Nutrition: mealDetailNutrition{
			Calories: meal.Calories,
			FatG:     meal.FatG,
			ProteinG: meal.ProteinG,
			CarbsG:   meal.CarbsG,
		},
	}
}

func (h *mealSearchHandler) lookupManualMealRestaurants(ctx context.Context, userID uuid.UUID, mealName string, locationOverride string) (string, string, []restaurantResponse, string) {
	if h.restaurantSearcher == nil {
		return "", "unavailable", []restaurantResponse{}, interfaces.RestaurantLookupUnavailable
	}

	location := ""
	basis := "unavailable"
	if selectedLocation := strings.TrimSpace(locationOverride); selectedLocation != "" {
		location = selectedLocation
		basis = "selected"
	} else {
		if h.preferenceService == nil {
			return "", "unavailable", []restaurantResponse{}, interfaces.RestaurantLookupUnavailable
		}

		preferences, err := h.preferenceService.GetByUserID(ctx, userID)
		if err != nil {
			return "", "unavailable", []restaurantResponse{}, interfaces.RestaurantLookupUnavailable
		}

		location, basis = selectMealDetailPreferenceLocation(*preferences, timeNow())
	}

	if strings.TrimSpace(location) == "" || basis == "unavailable" {
		return location, basis, []restaurantResponse{}, interfaces.RestaurantLookupUnavailable
	}

	result, err := h.restaurantSearcher.SearchRestaurants(ctx, interfaces.RestaurantSearchInput{
		MealName: mealName,
		Location: location,
		Limit:    30,
	})
	if err != nil {
		return location, basis, []restaurantResponse{}, interfaces.RestaurantLookupUnavailable
	}

	return location, basis, buildRestaurantResponses(result.Restaurants), result.Status
}

func customMealStateMatchesLocation(mealState string, currentLocation string) bool {
	mealState = normalizeCustomMealState(mealState)
	if mealState == "" {
		return false
	}

	locationState := normalizeCustomMealState(extractStateFromLocation(currentLocation))
	return mealState == locationState
}

func extractStateFromLocation(location string) string {
	location = strings.TrimSpace(location)
	if location == "" {
		return ""
	}

	separatorIndex := strings.LastIndex(location, ",")
	if separatorIndex == -1 {
		return location
	}

	return location[separatorIndex+1:]
}

func normalizeCustomMealState(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func float64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

var timeNow = func() time.Time {
	return time.Now()
}

func init() {
	endpoints = append(endpoints, RegisterMealSearchRoutes)
}
