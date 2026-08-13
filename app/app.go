// Package app: central dependency container
package app

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	authservice "fyp/food-rs/internal/service/auth"
	catalogService "fyp/food-rs/internal/service/catalog"
	customMealService "fyp/food-rs/internal/service/custommeal"
	"fyp/food-rs/internal/service/llm"
	"fyp/food-rs/internal/service/meallog"
	mealLogReportService "fyp/food-rs/internal/service/meallogreport"
	"fyp/food-rs/internal/service/mealsearch"
	preferenceService "fyp/food-rs/internal/service/preference"
	recommendationService "fyp/food-rs/internal/service/recommendation"
	"fyp/food-rs/internal/service/restaurant"
	"fyp/food-rs/internal/storage"
	"strings"

	"google.golang.org/genai"
	"gorm.io/gorm"
)

type contextKey string

const appContextKey contextKey = "food-recommendation-system:app"

type App struct {
	PostgresDB             *gorm.DB
	ImageStorage           interfaces.ImageStorage
	authService            interfaces.AuthService
	mealLogService         interfaces.MealLogService
	JWTSecret              string
	GeminiAPIKey           string
	preferenceService      interfaces.PreferenceService
	customMealService      interfaces.CustomMealService
	restaurantSearcher     interfaces.RestaurantSearcher
	recommendationService  interfaces.RecommendationService
	aiClient               AIClient
	catalogService         *catalogService.Service
	catalogFoodSearcher    interfaces.FoodSearcher
	mealLogReportService   interfaces.MealLogReportService
	objectStoragePublicURL string
	objectStorageBucket    string
	SerpAPIKey             string
}

type AIClient interface {
	interfaces.MealGenerator
	interfaces.CustomMealAutocompleter
	interfaces.MealMatchAdjudicator
	interfaces.MealDetailExplainer
}

var newMealGenerator = func(ctx context.Context, apiKey string) (AIClient, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return llm.NewClient(client), nil
}

func New(db *gorm.DB, jwtSecret, geminiAPIKey string, imageStorage interfaces.ImageStorage, objectStoragePublicURL, objectStorageBucket, serpAPIKey string) *App {
	return &App{
		PostgresDB:             db,
		JWTSecret:              jwtSecret,
		GeminiAPIKey:           geminiAPIKey,
		ImageStorage:           imageStorage,
		objectStoragePublicURL: objectStoragePublicURL,
		objectStorageBucket:    objectStorageBucket,
		SerpAPIKey:             serpAPIKey,
	}
}

func (a *App) GetCatalogService(_ context.Context) (*catalogService.Service, error) {
	if a.catalogService != nil {
		return a.catalogService, nil
	}

	repository := postgres.NewCatalogPostgresRepository(a.PostgresDB)
	resolver := storage.NewObjectURLResolver(a.objectStoragePublicURL, a.objectStorageBucket)
	a.catalogService = catalogService.NewService(repository, resolver)

	return a.catalogService, nil
}

func (a *App) GetMealLogService(ctx context.Context) (interfaces.MealLogService, error) {
	if a.mealLogService != nil {
		return a.mealLogService, nil
	}

	mealLogRepo := postgres.NewMealLogPostgresRepository(a.PostgresDB)

	customMealService, err := a.GetCustomMealService(ctx)
	if err != nil {
		return nil, err
	}

	catalogService, err := a.GetCatalogService(ctx)
	if err != nil {
		return nil, err
	}

	preferenceService, err := a.GetPreferenceService(ctx)
	if err != nil {
		return nil, err
	}

	a.mealLogService = meallog.NewService(mealLogRepo, customMealService, catalogService, preferenceService)

	return a.mealLogService, nil
}

func (a *App) GetCatalogFoodSearcher(ctx context.Context) (interfaces.FoodSearcher, error) {
	if a.catalogFoodSearcher != nil {
		return a.catalogFoodSearcher, nil
	}

	service, err := a.GetCatalogService(ctx)
	if err != nil {
		return nil, err
	}
	a.catalogFoodSearcher = catalogService.NewFoodSearcher(service)

	return a.catalogFoodSearcher, nil
}

// GetAuthService builds auth service from Postgres user repo
func (a *App) GetAuthService(ctx context.Context) (interfaces.AuthService, error) {
	if a.authService != nil {
		return a.authService, nil
	}

	userRepo := postgres.NewUserPostgresRepository(a.PostgresDB)
	refreshTokenRepo := postgres.NewRefreshTokenPostgresRepository(a.PostgresDB)
	a.authService = authservice.NewService(userRepo, refreshTokenRepo, a.JWTSecret)

	return a.authService, nil
}

// GetPreferenceService builds preference serice from Postgres preference repo
func (a *App) GetPreferenceService(ctx context.Context) (interfaces.PreferenceService, error) {
	if a.preferenceService != nil {
		return a.preferenceService, nil
	}

	preferenceRepo := postgres.NewPreferencePostgresRepository(a.PostgresDB)
	userRepo := postgres.NewUserPostgresRepository(a.PostgresDB)
	a.preferenceService = preferenceService.NewService(preferenceRepo, userRepo)

	return a.preferenceService, nil
}

// GetCustomMealService builds custom meal service from Postgres custom meal repo
func (a *App) GetCustomMealService(ctx context.Context) (interfaces.CustomMealService, error) {
	if a.customMealService != nil {
		return a.customMealService, nil
	}

	customMealRepo := postgres.NewCustomMealPostgresRepository(a.PostgresDB)
	a.customMealService = customMealService.NewService(customMealRepo)
	return a.customMealService, nil
}

func (a *App) GetAIClient(ctx context.Context) (AIClient, error) {
	if a.aiClient != nil {
		return a.aiClient, nil
	}

	aiClient, err := newMealGenerator(ctx, a.GeminiAPIKey)
	if err != nil {
		return nil, err
	}

	a.aiClient = aiClient
	return a.aiClient, nil
}

func (a *App) GetCustomMealAutocompleter(ctx context.Context) (interfaces.CustomMealAutocompleter, error) {
	return a.GetAIClient(ctx)
}

func (a *App) GetRestaurantSearcher(ctx context.Context) (interfaces.RestaurantSearcher, error) {
	if a.restaurantSearcher != nil {
		return a.restaurantSearcher, nil
	}

	if strings.TrimSpace(a.SerpAPIKey) == "" {
		a.restaurantSearcher = restaurant.NoopSearcher{}
		return a.restaurantSearcher, nil
	}

	a.restaurantSearcher = restaurant.NewSerpAPIClient(a.SerpAPIKey)
	return a.restaurantSearcher, nil
}

func (a *App) GetMealLogReportService(ctx context.Context) (interfaces.MealLogReportService, error) {
	if a.mealLogReportService != nil {
		return a.mealLogReportService, nil
	}

	mealLogRepo := postgres.NewMealLogPostgresRepository(a.PostgresDB)

	preferenceService, err := a.GetPreferenceService(ctx)
	if err != nil {
		return nil, err
	}

	a.mealLogReportService = mealLogReportService.NewService(mealLogRepo, preferenceService)

	return a.mealLogReportService, nil
}

// GetRecommendationService builds the recommendation service from Gemini and the prebuilt meal dataset.
func (a *App) GetRecommendationService(ctx context.Context) (interfaces.RecommendationService, error) {
	if a.recommendationService != nil {
		return a.recommendationService, nil
	}

	mealLogRepo := postgres.NewMealLogPostgresRepository(a.PostgresDB)
	customMealRepo := postgres.NewCustomMealPostgresRepository(a.PostgresDB)

	mealGenerator, err := a.GetAIClient(ctx)
	if err != nil {
		return nil, err
	}

	customMealService, err := a.GetCustomMealService(ctx)
	if err != nil {
		return nil, err
	}

	preferenceService, err := a.GetPreferenceService(ctx)
	if err != nil {
		return nil, err
	}

	customMealAutocompleter, err := a.GetCustomMealAutocompleter(ctx)
	if err != nil {
		return nil, err
	}

	catalogSvc, err := a.GetCatalogService(ctx)
	if err != nil {
		return nil, err
	}

	restaurantSearcher, err := a.GetRestaurantSearcher(ctx)
	if err != nil {
		return nil, err
	}

	catalogCandidateSearcher := catalogService.NewCandidateSearcher(catalogSvc)
	candidateSearcher := mealsearch.NewCandidateSearcher(customMealService, catalogCandidateSearcher)

	a.recommendationService = recommendationService.NewService(
		mealGenerator,
		candidateSearcher,
		mealGenerator,
		catalogSvc,
		customMealAutocompleter,
		mealLogRepo,
		customMealRepo,
		preferenceService,
		mealGenerator,
		restaurantSearcher,
	)

	return a.recommendationService, nil
}

func FromContext(ctx context.Context) *App {
	a := ctx.Value(appContextKey)
	if a != nil {
		return a.(*App)
	}

	return nil
}

// WithApp stores the application dependency container inside a context
func WithApp(ctx context.Context, a *App) context.Context {
	return context.WithValue(ctx, appContextKey, a)
}
