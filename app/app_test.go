package app

import (
	"context"
	"errors"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/service/restaurant"
	"io"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

type stubMealGenerator struct{}

type stubImageStorage struct{}

func (stubImageStorage) UploadMealImage(ctx context.Context, reader io.Reader, size int64, contentType string) (string, string, error) {
	return "", "", nil
}

func (stubImageStorage) PresignMealImage(ctx context.Context, imageURL string) (string, error) {
	return imageURL, nil
}

func (stubImageStorage) DeleteObject(ctx context.Context, objectName string) error {
	return nil
}

func (stubImageStorage) DeleteMealImage(ctx context.Context, imageURL string) error {
	return nil
}

func (stubMealGenerator) GenerateMeals(ctx context.Context, input interfaces.MealPromptInput) (interfaces.GeminiMealsResponse, error) {
	return interfaces.GeminiMealsResponse{}, nil
}

func (stubMealGenerator) AutocompleteCustomMeal(ctx context.Context, input interfaces.CustomMealAutocompleteInput) (interfaces.CustomMealAutocompleteResponse, error) {
	return interfaces.CustomMealAutocompleteResponse{}, nil
}

func (stubMealGenerator) ResolveMatches(ctx context.Context, tasks []interfaces.MealMatchTask) ([]interfaces.MealMatchDecision, error) {
	return nil, nil
}

func (stubMealGenerator) ExplainMealRecommendation(ctx context.Context, input interfaces.MealDetailExplanationInput) (string, error) {
	return "This meal fits the user's current preferences.", nil
}

func TestApp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "App Suite")
}

var _ = Describe("App dependencies", func() {
	var originalMealGeneratorFactory func(context.Context, string) (AIClient, error)

	BeforeEach(func() {
		originalMealGeneratorFactory = newAIClient
	})

	AfterEach(func() {
		newAIClient = originalMealGeneratorFactory
	})

	It("should keep constructor dependencies and configuration", func() {
		db := &gorm.DB{}
		imageStorage := &stubImageStorage{}

		a := New(db, "jwt-secret", "gemini-key", imageStorage, "http://localhost:9000", "images", "serp-key")

		Expect(a.PostgresDB).To(BeIdenticalTo(db))
		Expect(a.JWTSecret).To(Equal("jwt-secret"))
		Expect(a.GeminiAPIKey).To(Equal("gemini-key"))
		Expect(a.ImageStorage).To(BeIdenticalTo(imageStorage))
		Expect(a.SerpAPIKey).To(Equal("serp-key"))
	})

	It("should store and retrieve the app from context", func() {
		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images", "")

		Expect(FromContext(context.Background())).To(BeNil())

		ctx := WithApp(context.Background(), a)

		Expect(FromContext(ctx)).To(BeIdenticalTo(a))
	})

	It("should cache repository-backed services", func() {
		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images", "")
		ctx := context.Background()

		firstCatalog, err := a.GetCatalogService(ctx)
		Expect(err).NotTo(HaveOccurred())
		secondCatalog, err := a.GetCatalogService(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstCatalog).To(BeIdenticalTo(secondCatalog))

		firstAuth, err := a.GetAuthService(ctx)
		Expect(err).NotTo(HaveOccurred())
		secondAuth, err := a.GetAuthService(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstAuth).To(BeIdenticalTo(secondAuth))

		firstPreference, err := a.GetPreferenceService(ctx)
		Expect(err).NotTo(HaveOccurred())
		secondPreference, err := a.GetPreferenceService(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstPreference).To(BeIdenticalTo(secondPreference))

		firstCustomMeal, err := a.GetCustomMealService(ctx)
		Expect(err).NotTo(HaveOccurred())
		secondCustomMeal, err := a.GetCustomMealService(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstCustomMeal).To(BeIdenticalTo(secondCustomMeal))
	})

	It("should cache meal log and report services", func() {
		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images", "")
		ctx := context.Background()

		firstMealLog, err := a.GetMealLogService(ctx)
		Expect(err).NotTo(HaveOccurred())
		secondMealLog, err := a.GetMealLogService(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstMealLog).To(BeIdenticalTo(secondMealLog))

		firstReport, err := a.GetMealLogReportService(ctx)
		Expect(err).NotTo(HaveOccurred())
		secondReport, err := a.GetMealLogReportService(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(firstReport).To(BeIdenticalTo(secondReport))
	})

	It("should return AI client factory errors without caching a failed client", func() {
		expectedErr := errors.New("missing API key")
		factoryCalls := 0

		newAIClient = func(ctx context.Context, apiKey string) (AIClient, error) {
			factoryCalls++
			if factoryCalls == 1 {
				return nil, expectedErr
			}

			return stubMealGenerator{}, nil
		}

		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images", "")

		client, err := a.GetAIClient(context.Background())
		Expect(client).To(BeNil())
		Expect(err).To(MatchError(expectedErr))

		client, err = a.GetAIClient(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(client).NotTo(BeNil())
		Expect(factoryCalls).To(Equal(2))
	})

	It("should use a cached catalog searcher for recommendations", func() {
		var gotGeminiAPIKey string

		newAIClient = func(ctx context.Context, apiKey string) (AIClient, error) {
			gotGeminiAPIKey = apiKey
			return stubMealGenerator{}, nil
		}
		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images", "")

		service, err := a.GetRecommendationService(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(service).NotTo(BeNil())
		Expect(gotGeminiAPIKey).To(Equal("gemini-key"))

		first, err := a.GetCatalogFoodSearcher(context.Background())
		Expect(err).NotTo(HaveOccurred())
		second, err := a.GetCatalogFoodSearcher(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(first).To(BeIdenticalTo(second))
	})

	It("should use a cached AI client for custom meal autocomplete", func() {
		factoryCalls := 0

		newAIClient = func(ctx context.Context, apiKey string) (AIClient, error) {
			factoryCalls++
			return stubMealGenerator{}, nil
		}

		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images", "")

		first, err := a.GetCustomMealAutocompleter(context.Background())
		Expect(err).NotTo(HaveOccurred())

		second, err := a.GetCustomMealAutocompleter(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(first).To(BeIdenticalTo(second))
		Expect(factoryCalls).To(Equal(1))
	})

	It("should use a no-op restaurant searcher when SerpAPI key is missing", func() {
		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images", " ")

		first, err := a.GetRestaurantSearcher(context.Background())
		Expect(err).NotTo(HaveOccurred())
		second, err := a.GetRestaurantSearcher(context.Background())
		Expect(err).NotTo(HaveOccurred())

		Expect(first).To(BeIdenticalTo(second))
		_, ok := first.(restaurant.NoopSearcher)
		Expect(ok).To(BeTrue())

		result, err := first.SearchRestaurants(context.Background(), interfaces.RestaurantSearchInput{
			MealName: "Chicken Rice",
			Location: "Pavilion Kuala Lumpur",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Status).To(Equal(interfaces.RestaurantLookupUnavailable))
		Expect(result.Restaurants).To(BeEmpty())
	})

	It("should use a SerpAPI restaurant searcher when SerpAPI key is configured", func() {
		a := New(nil, "jwt-secret", "gemini-key", nil, "http://localhost:9000", "images", "serp-key")

		searcher, err := a.GetRestaurantSearcher(context.Background())

		Expect(err).NotTo(HaveOccurred())
		_, ok := searcher.(*restaurant.SerpAPIClient)
		Expect(ok).To(BeTrue())
	})
})
