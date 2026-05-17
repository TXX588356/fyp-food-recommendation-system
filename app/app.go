// Package app: central dependency container
package app

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	authservice "fyp/food-rs/internal/service/auth"

	"gorm.io/gorm"
)

type contextKey string

const appContextKey contextKey = "food-recommendation-system:app"

type App struct {
	PostgresDB  *gorm.DB
	authService interfaces.AuthService
	JWTSecret   string
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

func FromContext(ctx context.Context) *App {
	a := ctx.Value(appContextKey)
	if a != nil {
		return a.(*App)
	}

	return nil
}

func WithApp(ctx context.Context, a *App) context.Context {
	return context.WithValue(ctx, appContextKey, a)
}
