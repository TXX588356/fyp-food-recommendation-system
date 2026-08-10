package server

import (
	"context"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/config"
	"fyp/food-rs/internal/database"
	"fyp/food-rs/internal/endpoint"
	"fyp/food-rs/internal/storage"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run(parent context.Context) error {
	// Load config
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}

	imageStorage, err := storage.NewMinIOImageStorage(parent, cfg.MinIO.Endpoint, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, cfg.MinIO.Bucket, cfg.MinIO.UseSSL, cfg.MinIO.PublicURL)
	if err != nil {
		return err
	}

	// Open database
	db, err := database.NewPostgresDB(cfg.Database.URL)
	if err != nil {
		return err
	}

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	e.Use(middleware.RequestLogger())

	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	a := app.New(db, cfg.Security.JWTSecret, cfg.Gemini.APIKey, imageStorage, cfg.MinIO.PublicURL, cfg.MinIO.Bucket, cfg.SerpAPI.APIKey)
	ctx = app.WithApp(ctx, a) // attach App instance to the context, allowing other codes to retrieve
	endpoint.RegisterEndpoints(ctx, e)

	catalogService, err := a.GetCatalogService(ctx)
	if err != nil {
		return err
	}
	endpoint.RegisterCatalogRoutes(e, catalogService, cfg.Security.JWTSecret)

	// Makes Go backend serve the built React frontend
	// by exposing files inside the backend container's static folder & serve them from
	// website root '/'
	e.HTTPErrorHandler = func(c *echo.Context, err error) {
		if c.Request().Method == http.MethodGet {
			path := c.Request().URL.Path

			if path != "/" {
				filePath := filepath.Join(cfg.Client.Dir, path)
				if _, statErr := os.Stat(filePath); statErr == nil {
					_ = c.File(filePath)
					return
				}
			}

			_ = c.File(filepath.Join(cfg.Client.Dir, "index.html"))
			return
		}

		_ = c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	// Start Echo
	sc := echo.StartConfig{
		Address:         ":" + cfg.Server.Port,
		GracefulTimeout: 5 * time.Second,
	}

	return sc.Start(ctx, e)
}
