package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	cfg *viper.Viper

	Server struct {
		Port string `mapstructure:"port"`
	} `mapstructure:"server"`

	Database struct {
		URL      string `mapstructure:"url"`
		Host     string `mapstructure:"host"`
		Port     string `mapstructure:"port"`
		Name     string `mapstructure:"name"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`

	Security struct {
		JWTSecret string `mapstructure:"jwt_secret"`
	} `mapstructure:"security"`

	AI struct {
		GeminiAPIKey string `mapstructure:"gemini_api_key"`
	} `mapstructure:"ai"`

	Client struct {
		Dir string `mapstructure:"dir"`
	} `mapstructure:"client"`

	ObjectStorage struct {
		Endpoint  string `mapstructure:"endpoint"`
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
		Bucket    string `mapstructure:"bucket"`
		UseSSL    bool   `mapstructure:"use_ssl"`
		PublicURL string `mapstructure:"public_url"`
	} `mapstructure:"object_storage"`

	SerpAPI struct {
		APIKey string `mapstructure:"api_key"`
	} `mapstructure:"serpapi"`
}

func Load() *Config {
	cfg := viper.New()
	cfg.SetDefault("CLIENT_DIR", "static")
	cfg.SetDefault("SERVER_PORT", "8080")

	if envPath, ok := findDotEnv(); ok {
		cfg.SetConfigFile(envPath)
	} else {
		cfg.SetConfigFile(".env")
	}
	cfg.SetConfigType("env")
	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	cfg.AutomaticEnv()

	_ = cfg.ReadInConfig()

	instance := &Config{
		cfg: cfg,
	}
	instance.Server.Port = configString(cfg, "SERVER_PORT")
	instance.Database.URL = configString(cfg, "DATABASE_URL")
	if instance.Database.URL == "" {
		host := configString(cfg, "DATABASE_HOST")
		port := configString(cfg, "DATABASE_PORT")
		name := configString(cfg, "DATABASE_NAME")
		user := configString(cfg, "DATABASE_USER")
		password := configString(cfg, "DATABASE_PASSWORD")

		instance.Database.URL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s",
			url.QueryEscape(user),
			url.QueryEscape(password),
			host,
			port,
			name,
		)
	}
	instance.Security.JWTSecret = configString(cfg, "SECURITY_JWT_SECRET")
	instance.AI.GeminiAPIKey = configString(cfg, "GEMINI_API_KEY")
	instance.Client.Dir = configString(cfg, "CLIENT_DIR")
	instance.ObjectStorage.Endpoint = configStringWithFallback(cfg, "OBJECT_STORAGE_ENDPOINT", "MINIO_ENDPOINT")
	instance.ObjectStorage.AccessKey = configStringWithFallback(cfg, "OBJECT_STORAGE_ACCESS_KEY", "MINIO_ACCESS_KEY")
	instance.ObjectStorage.SecretKey = configStringWithFallback(cfg, "OBJECT_STORAGE_SECRET_KEY", "MINIO_SECRET_KEY")
	instance.ObjectStorage.Bucket = configStringWithFallback(cfg, "OBJECT_STORAGE_BUCKET", "MINIO_BUCKET")
	instance.ObjectStorage.UseSSL = configBoolWithFallback(cfg, "OBJECT_STORAGE_USE_SSL", "MINIO_USE_SSL")
	instance.ObjectStorage.PublicURL = configStringWithFallback(cfg, "OBJECT_STORAGE_PUBLIC_URL", "MINIO_PUBLIC_URL")
	instance.SerpAPI.APIKey = configString(cfg, "SERPAPI_API_KEY")

	return instance
}

func (c *Config) Validate() error {
	return nil
}

func configString(cfg *viper.Viper, key string) string {
	return strings.TrimSpace(cfg.GetString(key))
}

func configStringWithFallback(cfg *viper.Viper, key, fallbackKey string) string {
	if cfg.IsSet(key) {
		return configString(cfg, key)
	}

	return configString(cfg, fallbackKey)
}

func configBoolWithFallback(cfg *viper.Viper, key, fallbackKey string) bool {
	if cfg.IsSet(key) {
		return cfg.GetBool(key)
	}

	return cfg.GetBool(fallbackKey)
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
