package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	GeminiAPIKey   string
	DatabaseURL    string
	JWTSecret      string
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	MinIOPublicURL string
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
		GeminiAPIKey:   configString("GEMINI_API_KEY"),
		DatabaseURL:    configString("DATABASE_URL"),
		JWTSecret:      configString("SECURITY_JWT_SECRET"),
		MinIOEndpoint:  configString("MINIO_ENDPOINT"),
		MinIOAccessKey: configString("MINIO_ACCESS_KEY"),
		MinIOSecretKey: configString("MINIO_SECRET_KEY"),
		MinIOBucket:    configString("MINIO_BUCKET"),
		MinIOUseSSL:    viper.GetBool("MINIO_USE_SSL"),
		MinIOPublicURL: configString("MINIO_PUBLIC_URL"),
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
