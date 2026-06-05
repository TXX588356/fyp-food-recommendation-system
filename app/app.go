// Package app: central dependency container
package app

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	authservice "fyp/food-rs/internal/service/auth"
	customMealService "fyp/food-rs/internal/service/custommeal"
	preferenceService "fyp/food-rs/internal/service/preference"

	"gorm.io/gorm"
)

type contextKey string

const appContextKey contextKey = "food-recommendation-system:app"

type App struct {
	PostgresDB        *gorm.DB
	authService       interfaces.AuthService
	JWTSecret         string
	preferenceService interfaces.PreferenceService
	customMealService interfaces.CustomMealService
}

func New(db *gorm.DB, jwtSecret string) *App {
	return &App{
		PostgresDB: db,
		JWTSecret:  jwtSecret,
	}
}

// GetAuthService builds auth service from Postgres user repo
func (a *App) GetAuthService(ctx context.Context) (interfaces.AuthService, error) {
	if a.authService != nil {
		return a.authService, nil
	}

	userRepo := postgres.NewUserPostgresRepository(a.PostgresDB)
	a.authService = authservice.NewService(userRepo, a.JWTSecret)

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
