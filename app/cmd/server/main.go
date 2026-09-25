package server

import (
	"context"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"fyp/food-rs/app"
	"fyp/food-rs/internal/config"
	"fyp/food-rs/internal/database"
	"fyp/food-rs/internal/endpoint"
	"fyp/food-rs/internal/storage"
	"fyp/food-rs/static"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Run(parent context.Context) error {
	// Load config
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return err
	}

	imageStorage, err := storage.NewImageStorage(parent, cfg.ObjectStorage.Endpoint, cfg.ObjectStorage.AccessKey, cfg.ObjectStorage.SecretKey, cfg.ObjectStorage.Bucket, cfg.ObjectStorage.UseSSL, cfg.ObjectStorage.PublicURL)
	if err != nil {
		return err
	}

	// Open database
	db, err := database.NewPostgresDB(cfg.Database.URL)
	if err != nil {
		return err
	}

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	e.Use(middleware.RequestLogger())

	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	a := app.New(db, cfg.Security.JWTSecret, cfg.Gemini.APIKey, imageStorage, cfg.ObjectStorage.PublicURL, cfg.ObjectStorage.Bucket, cfg.SerpAPI.APIKey)
	ctx = app.WithApp(ctx, a) // attach App instance to the context, allowing other codes to retrieve
	endpoint.RegisterEndpoints(ctx, e)

	clientDir := resolveClientDir(cfg.Client.Dir)
	e.GET("/*", func(c *echo.Context) error {
		return serveClientRoute(c, clientDir)
	})

	// Start Echo
	sc := echo.StartConfig{
		Address:         ":" + cfg.Server.Port,
		GracefulTimeout: 5 * time.Second,
	}

	return sc.Start(ctx, e)
}

func serveClientRoute(c *echo.Context, clientDir string) error {
	requestPath := c.Request().URL.Path
	if strings.HasPrefix(requestPath, "/api/") || requestPath == "/api" {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "not found"})
	}

	if requestPath != "/" {
		filePath := filepath.Join(clientDir, requestPath)
		if _, err := os.Stat(filePath); err == nil {
			return c.File(filePath)
		}

		if err := serveEmbeddedClientFile(c, requestPath); err == nil {
			return nil
		}
	}

	indexPath := filepath.Join(clientDir, "index.html")
	if err := c.File(indexPath); err == nil {
		return nil
	}

	if err := serveEmbeddedClientFile(c, "/index.html"); err == nil {
		return nil
	}

	return c.JSON(http.StatusInternalServerError, map[string]string{
		"error": fmt.Sprintf("frontend entrypoint not found at %s or embedded static/index.html", indexPath),
	})
}

func serveEmbeddedClientFile(c *echo.Context, requestPath string) error {
	fileName := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
	if fileName == "." || fileName == "" {
		fileName = "index.html"
	}

	data, err := fs.ReadFile(static.FS, fileName)
	if err != nil {
		return err
	}

	contentType := mime.TypeByExtension(path.Ext(fileName))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	return c.Blob(http.StatusOK, contentType, data)
}

func resolveClientDir(configuredDir string) string {
	configuredDir = strings.TrimSpace(configuredDir)
	if configuredDir == "" {
		configuredDir = "static"
	}

	for _, candidate := range clientDirCandidates(configuredDir) {
		if _, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil {
			return candidate
		}
	}

	return configuredDir
}

func clientDirCandidates(configuredDir string) []string {
	candidates := []string{configuredDir}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, configuredDir))

		for dir := cwd; ; dir = filepath.Dir(dir) {
			candidates = append(candidates, filepath.Join(dir, configuredDir))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}

	if executable, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executable)
		candidates = append(candidates, filepath.Join(executableDir, configuredDir))
	}

	seen := make(map[string]struct{}, len(candidates))
	unique := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		unique = append(unique, cleaned)
	}

	return unique
}
