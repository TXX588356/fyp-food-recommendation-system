// Package app: central dependency container
package app

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	authservice "fyp/food-rs/internal/service/auth"
	customMealService "fyp/food-rs/internal/service/custommeal"
	"fyp/food-rs/internal/service/llm"
	"fyp/food-rs/internal/service/mealdataset"
	"fyp/food-rs/internal/service/mealsearch"
	preferenceService "fyp/food-rs/internal/service/preference"
	recommendationService "fyp/food-rs/internal/service/recommendation"

	"google.golang.org/genai"
	"gorm.io/gorm"
)

type contextKey string

const appContextKey contextKey = "food-recommendation-system:app"

type App struct {
	PostgresDB            *gorm.DB
	authService           interfaces.AuthService
	JWTSecret             string
	GeminiAPIKey          string
	preferenceService     interfaces.PreferenceService
	customMealService     interfaces.CustomMealService
	recommendationService interfaces.RecommendationService
}

var newMealGenerator = func(ctx context.Context, apiKey string) (interfaces.MealGenerator, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return llm.NewClient(client), nil
}

var newFoodSearcher = func(datasetPath string) (interfaces.FoodSearcher, error) {
	return mealdataset.NewPrebuiltSearcher(datasetPath)
}

func New(db *gorm.DB, jwtSecret string, geminiAPIKey string) *App {
	return &App{
		PostgresDB:   db,
		JWTSecret:    jwtSecret,
		GeminiAPIKey: geminiAPIKey,
	}
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

// GetRecommendationService builds the recommendation service from Gemini and the prebuilt meal dataset.
func (a *App) GetRecommendationService(ctx context.Context) (interfaces.RecommendationService, error) {
	if a.recommendationService != nil {
		return a.recommendationService, nil
	}

	mealGenerator, err := newMealGenerator(ctx, a.GeminiAPIKey)
	if err != nil {
		return nil, err
	}

	customMealService, err := a.GetCustomMealService(ctx)
	if err != nil {
		return nil, err
	}

	const prebuiltMealDatasetPath = "internal/data/prebuilt_meals.json"
	prebuiltSearcher, err := newFoodSearcher(prebuiltMealDatasetPath)
	if err != nil {
		return nil, err
	}

	foodSearcher := mealsearch.NewCombinedSearcher(customMealService, prebuiltSearcher)

	a.recommendationService = recommendationService.NewService(mealGenerator, foodSearcher)

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
