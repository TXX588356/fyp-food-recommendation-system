package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestFindDotEnvWalksUpFromNestedDirectory(t *testing.T) {
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	root := t.TempDir()
	nested := filepath.Join(root, "backend", "cmd", "server")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}
	envPath := filepath.Join(root, ".env")
	if err := os.WriteFile(envPath, []byte("GEMINI_API_KEY=test\n"), 0o600); err != nil {
		t.Fatalf("failed to create .env: %v", err)
	}
	if err := os.Chdir(nested); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	got, ok := findDotEnv()
	if !ok {
		t.Fatal("expected .env to be found")
	}
	if got != envPath {
		t.Fatalf("expected %q, got %q", envPath, got)
	}
}

func TestLoadReadsRuntimeConfig(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatalf("failed to restore working directory: %v", err)
		}
	})

	root := t.TempDir()
	envPath := filepath.Join(root, ".env")
	env := []byte("GEMINI_API_KEY= test-gemini-key \nDATABASE_URL= postgres://test \nSECURITY_JWT_SECRET= test-secret \n")
	if err := os.WriteFile(envPath, env, 0o600); err != nil {
		t.Fatalf("failed to create .env: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to change directory: %v", err)
	}

	cfg := Load()

	if cfg.GeminiAPIKey != "test-gemini-key" {
		t.Fatalf("expected GeminiAPIKey to be read and trimmed")
	}
	if cfg.DatabaseURL != "postgres://test" {
		t.Fatalf("expected DatabaseURL to be read and trimmed")
	}
	if cfg.JWTSecret != "test-secret" {
		t.Fatalf("expected JWTSecret to be read and trimmed")
	}
}
