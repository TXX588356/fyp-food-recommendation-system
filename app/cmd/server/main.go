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
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))
	e.Use(middleware.RequestLogger())

	ctx, cancel := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	a := app.New(db)
	ctx = app.WithApp(ctx, a) // attach App instance to the context, allowing other codes to retrieve
	endpoint.RegisterEndpoints(ctx, e)

	// Start Echo
	sc := echo.StartConfig{
		Address:         ":8088",
		GracefulTimeout: 5 * time.Second,
	}

	// if err := sc.Start(ctx, e); err != nil {
	// 	e.Logger.Error("failed to start server", "error", err)
	// }

	// server := &http.Server{
	// 	Addr:    ":8088",
	// 	Handler: endpoint.NewRouter(),
	// }

	// log.Println("server running on port 8088")
	// if err := server.ListenAndServe(); err != nil {
	// 	log.Fatal(err)
	// }

	// ctx := context.Background()
	// if cfg.GeminiAPIKey == "" {
	// 	log.Fatal("GEMINI_API_KEY is not set; make sure it exists in the environment or in .env")
	// }
	// if cfg.KaloriAPIKey == "" {
	// 	log.Fatal("KAL_API is not set; make sure it exists in the environment or in .env")
	// }

	// geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
	// 	APIKey:  cfg.GeminiAPIKey,
	// 	Backend: genai.BackendGeminiAPI,
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// service := recommendation.NewService(
	// 	llm.NewClient(geminiClient),
	// 	foodapi.NewKaloriClient(cfg.KaloriAPIKey, &http.Client{Timeout: 15 * time.Second}),
	// )

	// results, err := service.Recommend(ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// encoder := json.NewEncoder(os.Stdout)
	// for _, item := range results {
	// 	if err := encoder.Encode(item); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
	return sc.Start(ctx, e)
}
