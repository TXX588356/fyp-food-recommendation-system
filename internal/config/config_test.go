package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {
	BeforeEach(func() {
		clearConfigEnv()
	})

	It("should load runtime config from config/config.yaml", func() {
		root := useTempWorkingDir()
		writeConfig(root, `
server:
  port: "8080"
database:
  url: postgres://yaml
security:
  jwt_secret: test-secret
gemini:
  api_key: test-gemini-key
client:
  dir: web
object_storage:
  endpoint: storage.example.com
  access_key: access
  secret_key: secret
  bucket: meals
  use_ssl: true
  public_url: https://cdn.example.com
serpapi:
  api_key: serp
	`)

		cfg := Load()

		Expect(cfg.Server.Port).To(Equal("8080"))
		Expect(cfg.Database.URL).To(Equal("postgres://yaml"))
		Expect(cfg.Security.JWTSecret).To(Equal("test-secret"))
		Expect(cfg.Gemini.APIKey).To(Equal("test-gemini-key"))
		Expect(cfg.Client.Dir).To(Equal("web"))
		Expect(cfg.ObjectStorage.Endpoint).To(Equal("storage.example.com"))
		Expect(cfg.ObjectStorage.AccessKey).To(Equal("access"))
		Expect(cfg.ObjectStorage.SecretKey).To(Equal("secret"))
		Expect(cfg.ObjectStorage.Bucket).To(Equal("meals"))
		Expect(cfg.ObjectStorage.UseSSL).To(BeTrue())
		Expect(cfg.ObjectStorage.PublicURL).To(Equal("https://cdn.example.com"))
		Expect(cfg.SerpAPI.APIKey).To(Equal("serp"))
	})

	It("should let environment variables override yaml values", func() {
		root := useTempWorkingDir()
		writeConfig(root, `
database:
  url: postgres://yaml
security:
  jwt_secret: yaml-secret
`)
		Expect(os.Setenv("DATABASE_URL", "postgres://env")).To(Succeed())
		Expect(os.Setenv("SECURITY_JWT_SECRET", "env-secret")).To(Succeed())

		cfg := Load()

		Expect(cfg.Database.URL).To(Equal("postgres://env"))
		Expect(cfg.Security.JWTSecret).To(Equal("env-secret"))
	})

	It("should load .env values over yaml values", func() {
		root := useTempWorkingDir()
		writeConfig(root, `
gemini:
  api_key: yaml-gemini-key
`)
		Expect(os.WriteFile(filepath.Join(root, ".env"), []byte("GEMINI_API_KEY=env-file-gemini-key\n"), 0o600)).To(Succeed())

		cfg := Load()

		Expect(cfg.Gemini.APIKey).To(Equal("env-file-gemini-key"))
	})

	It("should find .env by walking up from a nested directory", func() {
		root := useTempWorkingDir()
		writeConfig(root, `
gemini:
  api_key: yaml-gemini-key
`)
		Expect(os.WriteFile(filepath.Join(root, ".env"), []byte("GEMINI_API_KEY=nested-env-file-key\n"), 0o600)).To(Succeed())
		nested := filepath.Join(root, "app", "cmd", "server")
		Expect(os.MkdirAll(nested, 0o755)).To(Succeed())
		Expect(os.Chdir(nested)).To(Succeed())

		cfg := Load()

		Expect(cfg.Gemini.APIKey).To(Equal("nested-env-file-key"))
	})

	It("should build database url from database parts when url is omitted", func() {
		root := useTempWorkingDir()
		writeConfig(root, `
database:
  host: localhost
  port: "5433"
  name: fyp_food_recommendation
  user: postgresdb
  password: postgresdb
`)

		cfg := Load()

		Expect(cfg.Database.URL).To(Equal("postgres://postgresdb:postgresdb@localhost:5433/fyp_food_recommendation"))
	})
})

func useTempWorkingDir() string {
	originalDir, err := os.Getwd()
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(func() {
		Expect(os.Chdir(originalDir)).To(Succeed())
	})

	root := GinkgoT().TempDir()
	Expect(os.Chdir(root)).To(Succeed())
	return root
}

func writeConfig(root string, content string) {
	configDir := filepath.Join(root, "config")
	Expect(os.MkdirAll(configDir, 0o755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(strings.TrimSpace(content)+"\n"), 0o600)).To(Succeed())
}

func clearConfigEnv() {
	for _, key := range []string{
		"SERVER_PORT",
		"CLIENT_DIR",
		"DATABASE_URL",
		"DATABASE_HOST",
		"DATABASE_PORT",
		"DATABASE_NAME",
		"DATABASE_USER",
		"DATABASE_PASSWORD",
		"SECURITY_JWT_SECRET",
		"GEMINI_API_KEY",
		"OBJECT_STORAGE_ENDPOINT",
		"OBJECT_STORAGE_ACCESS_KEY",
		"OBJECT_STORAGE_SECRET_KEY",
		"OBJECT_STORAGE_BUCKET",
		"OBJECT_STORAGE_USE_SSL",
		"OBJECT_STORAGE_PUBLIC_URL",
		"SERPAPI_API_KEY",
	} {
		Expect(os.Unsetenv(key)).To(Succeed())
	}
}
