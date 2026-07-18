package config

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("Config", func() {
	BeforeEach(func() {
		viper.Reset()
		DeferCleanup(viper.Reset)
	})

	It("should find .env by walking up from a nested directory", func() {
		originalDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			Expect(os.Chdir(originalDir)).To(Succeed())
		})

		root := GinkgoT().TempDir()
		nested := filepath.Join(root, "backend", "cmd", "server")
		Expect(os.MkdirAll(nested, 0o755)).To(Succeed())
		envPath := filepath.Join(root, ".env")
		Expect(os.WriteFile(envPath, []byte("GEMINI_API_KEY=test\n"), 0o600)).To(Succeed())
		Expect(os.Chdir(nested)).To(Succeed())

		got, ok := findDotEnv()

		Expect(ok).To(BeTrue())
		Expect(got).To(Equal(envPath))
	})

	It("should load runtime config from .env", func() {
		originalDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			Expect(os.Chdir(originalDir)).To(Succeed())
		})

		root := GinkgoT().TempDir()
		envPath := filepath.Join(root, ".env")
		env := []byte("GEMINI_API_KEY= test-gemini-key \nDATABASE_URL= postgres://test \nSECURITY_JWT_SECRET= test-secret \n")
		Expect(os.WriteFile(envPath, env, 0o600)).To(Succeed())
		Expect(os.Chdir(root)).To(Succeed())

		cfg := Load()

		Expect(cfg.GeminiAPIKey).To(Equal("test-gemini-key"))
		Expect(cfg.DatabaseURL).To(Equal("postgres://test"))
		Expect(cfg.JWTSecret).To(Equal("test-secret"))
	})

})
