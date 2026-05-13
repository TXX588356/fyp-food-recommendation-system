package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"fyp/food-rs/internal/config"
	"fyp/food-rs/internal/foodapi"
	"fyp/food-rs/internal/llm"
	"fyp/food-rs/internal/recommendation"
	"google.golang.org/genai"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()
	if cfg.GeminiAPIKey == "" {
		log.Fatal("GEMINI_API_KEY is not set; make sure it exists in the environment or in .env")
	}
	if cfg.KaloriAPIKey == "" {
		log.Fatal("KAL_API is not set; make sure it exists in the environment or in .env")
	}

	geminiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.GeminiAPIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}

	service := recommendation.NewService(
		llm.NewClient(geminiClient),
		foodapi.NewKaloriClient(cfg.KaloriAPIKey, &http.Client{Timeout: 15 * time.Second}),
	)

	results, err := service.Recommend(ctx)
	if err != nil {
		log.Fatal(err)
	}

	encoder := json.NewEncoder(os.Stdout)
	for _, item := range results {
		if err := encoder.Encode(item); err != nil {
			log.Fatal(err)
		}
	}
}
