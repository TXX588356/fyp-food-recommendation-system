//go:build external

package llm_test

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

const llmTestModel = "gemini-3.5-flash-lite"

func TestLLMCompliance(t *testing.T) {
	silenceTestLogs()
	RegisterFailHandler(Fail)
	RunSpecs(t, "LLM Recommendation Compliance Suite")
}

func silenceTestLogs() {
	log.SetOutput(io.Discard)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func llmTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 45*time.Second)
}

func requireEnv(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		Skip(name + " is required for LLM compliance tests")
	}

	return value
}

func newRealGeminiClient(ctx context.Context) llm.Client {
	return llm.NewClient(newRealGeminiAPIClient(ctx))
}

func newRealGeminiAPIClient(ctx context.Context) *genai.Client {
	apiKey := requireEnv("GEMINI_API_KEY")

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	Expect(err).NotTo(HaveOccurred())

	return client
}
