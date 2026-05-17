package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	GeminiAPIKey string
	KaloriAPIKey string
	DatabaseURL  string
}

func Load() Config {
	if envPath, ok := findDotEnv(); ok {
		viper.SetConfigFile(envPath)
	} else {
		viper.SetConfigFile(".env")
	}
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()

	return Config{
		GeminiAPIKey: strings.TrimSpace(viper.GetString("GEMINI_API_KEY")),
		KaloriAPIKey: strings.TrimSpace(viper.GetString("KAL_API")),
		DatabaseURL:  viper.GetString("DATABASE_URL"),
	}
}

func findDotEnv() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}

	for {
		envPath := filepath.Join(dir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			return envPath, true
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
