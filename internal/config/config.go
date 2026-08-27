package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

type Config struct {
	cfg *viper.Viper `mapstructure:"-"`

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

	Gemini struct {
		APIKey string `mapstructure:"api_key"`
	} `mapstructure:"gemini"`

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
	loadDotEnv()

	cfg := viper.New()

	cfg.SetConfigName("config")
	cfg.SetConfigType("yaml")
	cfg.AddConfigPath("./config")
	cfg.AutomaticEnv()

	cfg.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults(cfg)
	bindEnv(cfg)

	if err := cfg.ReadInConfig(); err != nil {
		fmt.Printf("Config | Read Error: %s\n", err)
	}

	instance := &Config{cfg: cfg}
	if err := cfg.Unmarshal(instance); err != nil {
		panic(err)
	}
	normalize(instance)

	return instance
}

func setDefaults(cfg *viper.Viper) {
	cfg.SetDefault("server.port", "8080")
	cfg.SetDefault("client.dir", "static")
}

func bindEnv(cfg *viper.Viper) {
	_ = cfg.BindEnv("server.port", "SERVER_PORT", "PORT")
	_ = cfg.BindEnv("client.dir", "CLIENT_DIR")
	_ = cfg.BindEnv("database.url", "DATABASE_URL")
	_ = cfg.BindEnv("database.host", "DATABASE_HOST")
	_ = cfg.BindEnv("database.port", "DATABASE_PORT")
	_ = cfg.BindEnv("database.name", "DATABASE_NAME")
	_ = cfg.BindEnv("database.user", "DATABASE_USER")
	_ = cfg.BindEnv("database.password", "DATABASE_PASSWORD")
	_ = cfg.BindEnv("security.jwt_secret", "SECURITY_JWT_SECRET")
	_ = cfg.BindEnv("gemini.api_key", "GEMINI_API_KEY")
	_ = cfg.BindEnv("object_storage.endpoint", "OBJECT_STORAGE_ENDPOINT")
	_ = cfg.BindEnv("object_storage.access_key", "OBJECT_STORAGE_ACCESS_KEY")
	_ = cfg.BindEnv("object_storage.secret_key", "OBJECT_STORAGE_SECRET_KEY")
	_ = cfg.BindEnv("object_storage.bucket", "OBJECT_STORAGE_BUCKET")
	_ = cfg.BindEnv("object_storage.use_ssl", "OBJECT_STORAGE_USE_SSL")
	_ = cfg.BindEnv("object_storage.public_url", "OBJECT_STORAGE_PUBLIC_URL")
	_ = cfg.BindEnv("serpapi.api_key", "SERPAPI_API_KEY")
}

func normalize(cfg *Config) {
	cfg.Server.Port = strings.TrimSpace(cfg.Server.Port)
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		cfg.Server.Port = port
	}
	cfg.Client.Dir = strings.TrimSpace(cfg.Client.Dir)
	cfg.Database.URL = strings.TrimSpace(cfg.Database.URL)
	cfg.Database.Host = strings.TrimSpace(cfg.Database.Host)
	cfg.Database.Port = strings.TrimSpace(cfg.Database.Port)
	cfg.Database.Name = strings.TrimSpace(cfg.Database.Name)
	cfg.Database.User = strings.TrimSpace(cfg.Database.User)
	cfg.Database.Password = strings.TrimSpace(cfg.Database.Password)
	cfg.Security.JWTSecret = strings.TrimSpace(cfg.Security.JWTSecret)
	cfg.Gemini.APIKey = strings.TrimSpace(cfg.Gemini.APIKey)
	cfg.ObjectStorage.Endpoint = strings.TrimSpace(cfg.ObjectStorage.Endpoint)
	cfg.ObjectStorage.AccessKey = strings.TrimSpace(cfg.ObjectStorage.AccessKey)
	cfg.ObjectStorage.SecretKey = strings.TrimSpace(cfg.ObjectStorage.SecretKey)
	cfg.ObjectStorage.Bucket = strings.TrimSpace(cfg.ObjectStorage.Bucket)
	cfg.ObjectStorage.PublicURL = strings.TrimSpace(cfg.ObjectStorage.PublicURL)
	cfg.SerpAPI.APIKey = strings.TrimSpace(cfg.SerpAPI.APIKey)

	if cfg.Database.URL == "" {
		cfg.Database.URL = buildDatabaseURL(cfg)
	}
}

func buildDatabaseURL(cfg *Config) string {
	if cfg.Database.Host == "" || cfg.Database.Port == "" || cfg.Database.Name == "" || cfg.Database.User == "" {
		return ""
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		url.QueryEscape(cfg.Database.User),
		url.QueryEscape(cfg.Database.Password),
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
	)
}

func loadDotEnv() {
	if envPath, ok := findDotEnv(); ok {
		_ = gotenv.Load(envPath)
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

func (c *Config) Validate() error {
	return nil
}
