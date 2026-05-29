package server

import (
	"context"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/config"
	"fyp/food-rs/internal/database"
	"fyp/food-rs/internal/endpoint"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run(parent context.Context) error {
	// Load config
	cfg := config.Load()

	// Open database
	db, err := database.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		return err
	}

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	e.Use(middleware.RequestLogger())

	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	a := app.New(db, cfg.JWTSecret)
	ctx = app.WithApp(ctx, a) // attach App instance to the context, allowing other codes to retrieve
	endpoint.RegisterEndpoints(ctx, e)

	// Start Echo
	sc := echo.StartConfig{
		Address:         ":8088",
		GracefulTimeout: 5 * time.Second,
	}

	return sc.Start(ctx, e)
}
