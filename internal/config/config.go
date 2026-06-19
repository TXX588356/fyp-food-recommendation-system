package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	GeminiAPIKey string
	DatabaseURL  string
	JWTSecret    string
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
		GeminiAPIKey: configString("GEMINI_API_KEY"),
		DatabaseURL:  configString("DATABASE_URL"),
		JWTSecret:    configString("SECURITY_JWT_SECRET"),
	}
}

func configString(key string) string {
	return strings.TrimSpace(viper.GetString(key))
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
