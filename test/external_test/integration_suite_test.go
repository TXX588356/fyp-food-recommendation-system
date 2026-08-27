//go:build external

package external_test

import (
	"context"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"fyp/food-rs/internal/service/llm"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/genai"
)

func TestExternalIntegration(t *testing.T) {
	silenceTestLogs()
	RegisterFailHandler(Fail)
	RunSpecs(t, "External Integration Suite")
}

func silenceTestLogs() {
	log.SetOutput(io.Discard)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func externalTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func requireEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		Skip(name + " is required for external integration tests")
	}

	return value
}

func newRealGeminiClient(ctx context.Context) llm.Client {
	apiKey := requireEnv("GEMINI_API_KEY")

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	Expect(err).NotTo(HaveOccurred())

	return llm.NewClient(client)
}
