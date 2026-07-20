package config

import (
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
		URL string `mapstructure:"url"`
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

	MinIO struct {
		Endpoint  string `mapstructure:"endpoint"`
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
		Bucket    string `mapstructure:"bucket"`
		UseSSL    bool   `mapstructure:"use_ssl"`
		PublicURL string `mapstructure:"public_url"`
	} `mapstructure:"minio"`

	SerpAPI struct {
		APIKey string `mapstructure:"api_key"`
	} `mapstructure:"serpapi"`
}

func Load() *Config {
	cfg := viper.New()
	cfg.SetDefault("CLIENT_DIR", "static")
	cfg.SetDefault("SERVER_PORT", "8080")
	cfg.SetDefault("MINIO_USE_SSL", false)

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
	instance.Security.JWTSecret = configString(cfg, "SECURITY_JWT_SECRET")
	instance.AI.GeminiAPIKey = configString(cfg, "GEMINI_API_KEY")
	instance.Client.Dir = configString(cfg, "CLIENT_DIR")
	instance.MinIO.Endpoint = configString(cfg, "MINIO_ENDPOINT")
	instance.MinIO.AccessKey = configString(cfg, "MINIO_ACCESS_KEY")
	instance.MinIO.SecretKey = configString(cfg, "MINIO_SECRET_KEY")
	instance.MinIO.Bucket = configString(cfg, "MINIO_BUCKET")
	instance.MinIO.UseSSL = cfg.GetBool("MINIO_USE_SSL")
	instance.MinIO.PublicURL = configString(cfg, "MINIO_PUBLIC_URL")
	instance.SerpAPI.APIKey = configString(cfg, "SERPAPI_API_KEY")

	return instance
}

func (c *Config) Validate() error {
	return nil
}

func configString(cfg *viper.Viper, key string) string {
	return strings.TrimSpace(cfg.GetString(key))
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
