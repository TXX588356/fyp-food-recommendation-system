package endpoint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"net/http"

	"net/http/httptest"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func registerRecommendationTestRoutes(e *echo.Echo, recommendationService interfaces.RecommendationService, jwtSecret string) {
	handler := &recommendationHandler{
		recommendationService: recommendationService,
	}

	recommendations := e.Group("/recommendations", middleware.Auth(jwtSecret))
	recommendations.POST("", handler.generateRecommendations)
}

func performRecommendationRequest(e *echo.Echo, body any, token string) *httptest.ResponseRecorder {
	var requestBody *bytes.Reader

	switch value := body.(type) {
	case nil:
		requestBody = bytes.NewReader(nil)
	case string:
		requestBody = bytes.NewReader([]byte(value))
	default:
		encodedBody, err := json.Marshal(value)
		Expect(err).NotTo(HaveOccurred())
		requestBody = bytes.NewReader(encodedBody)
	}

	request := httptest.NewRequest(http.MethodPost, "/recommendations", requestBody)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	if token != "" {
		request.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}

	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)

	return response
}

func generateRecommendationTestToken(userID uuid.UUID, jwtSecret string) string {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"email":   "user@example.com",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenText, err := token.SignedString([]byte(jwtSecret))
	Expect(err).NotTo(HaveOccurred())

	return tokenText
}

var _ = Describe("Recommendation endpoint contract", func() {
	Describe("validateGenerateRecommendationRequest", func() {
		It("should accept a valid request", func() {
			err := validateGenerateRecommendationRequest(generateRecommendationRequest{
				MealCategory:      "lunch",
				CurrentMonthSpent: 120,
				PerMealBudget:     12,
			})

			Expect(err).NotTo(HaveOccurred())
		})

		It("should require meal category", func() {
			err := validateGenerateRecommendationRequest(generateRecommendationRequest{})

			Expect(err).To(MatchError("meal category is required"))
		})

		It("should reject unsupported meal category", func() {
			err := validateGenerateRecommendationRequest(generateRecommendationRequest{
				MealCategory: "supper",
			})

			Expect(err).To(MatchError("unsupported meal category: supper"))
		})

		It("should reject negative current month spent", func() {
			err := validateGenerateRecommendationRequest(generateRecommendationRequest{
				MealCategory:      "dinner",
				CurrentMonthSpent: -1,
			})

			Expect(err).To(MatchError("current month spent cannot be negative"))
		})

		It("should reject negative per meal budget", func() {
			err := validateGenerateRecommendationRequest(generateRecommendationRequest{
				MealCategory:  "lunch",
				PerMealBudget: -1,
			})

			Expect(err).To(MatchError("per meal budget cannot be negative"))
		})
	})

	Describe("POST /recommendations", func() {
		const jwtSecret = "test-secret"

		var (
			e                     *echo.Echo
			recommendationService *mocks.RecommendationService
			userID                uuid.UUID
			token                 string
		)

		BeforeEach(func() {
			e = echo.New()
			recommendationService = mocks.NewRecommendationService(GinkgoT())
			userID = uuid.New()
			token = generateRecommendationTestToken(userID, jwtSecret)

			registerRecommendationTestRoutes(e, recommendationService, jwtSecret)
		})

		It("should reject unauthenticated requests", func() {
			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory: "lunch",
			}, "")

			Expect(response.Code).To(Equal(http.StatusUnauthorized))
		})

		It("should reject invalid JSON body", func() {
			response := performRecommendationRequest(e, "{bad-json", token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("invalid request body"))
		})

		It("should reject invalid meal category", func() {
			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory: "supper",
			}, token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("unsupported meal category: supper"))
		})

		It("should reject negative budget values", func() {
			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory:  "lunch",
				PerMealBudget: -1,
			}, token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("per meal budget cannot be negative"))
		})

		It("should generate recommendations from request context", func() {
			expectedInput := interfaces.RecommendationRequestInput{
				MealCategory:      "lunch",
				CurrentMonthSpent: 150,
				PerMealBudget:     12,
			}

			candidates := []interfaces.MatchedMealCandidate{
				{
					GeneratedMeal: interfaces.GeneratedMeal{
						Name: "Nasi Lemak",
					},
					Food: interfaces.FoodSearchResult{
						ID:       "nasi lemak",
						Name:     "Nasi Lemak",
						Tags:     []string{"Rice"},
						Calories: 494,
						FatG:     14,
						ProteinG: 13,
						CarbsG:   80,
					},
					MatchedQuery: "Nasi Lemak",
				},
			}
			filteredOut := []interfaces.FilteredMealCandidate{
				{
					Candidate: interfaces.MatchedMealCandidate{
						GeneratedMeal: interfaces.GeneratedMeal{
							Name: "Pork Noodles",
						},
						Food: interfaces.FoodSearchResult{
							ID:   "pork-food",
							Name: "Pork Noodles",
							Tags: []string{"pork"},
						},
						MatchedQuery: "Pork Noodles",
					},
					Reason: "contains pork, not suitable for halal restriction",
				},
			}

			recommendationService.EXPECT().GenerateRecommendationResult(mock.Anything, userID, expectedInput).
				Return(interfaces.RecommendationResult{
					Candidates:       candidates,
					FilteredOut:      filteredOut,
					FilteringApplied: true,
				}, nil).
				Once()

			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory:      "lunch",
				CurrentMonthSpent: 150,
				PerMealBudget:     12,
			}, token)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring(`"candidates"`))
			Expect(response.Body.String()).To(ContainSubstring(`"filtered_out"`))
			Expect(response.Body.String()).To(ContainSubstring(`"filtering_applied":true`))
			Expect(response.Body.String()).To(ContainSubstring(`"reason":"contains pork, not suitable for halal restriction"`))
			Expect(response.Body.String()).To(ContainSubstring(`"Nasi Lemak"`))
			Expect(response.Body.String()).To(ContainSubstring(`"matched_query":"Nasi Lemak"`))
		})

		It("should return an error when recommendation generation fails", func() {
			recommendationService.EXPECT().GenerateRecommendationResult(mock.Anything, userID, mock.Anything).
				Return(interfaces.RecommendationResult{}, errors.New("Gemini unavailable")).
				Once()

			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory: "lunch",
			}, token)

			Expect(response.Code).To(Equal(http.StatusInternalServerError))
			Expect(response.Body.String()).To(ContainSubstring("failed to generate recommendations"))
		})

		It("should return empty candidates when no generated meals can be matched", func() {
			recommendationService.EXPECT().GenerateRecommendationResult(mock.Anything, userID, mock.Anything).
				Return(interfaces.RecommendationResult{
					Candidates:       []interfaces.MatchedMealCandidate{},
					FilteredOut:      []interfaces.FilteredMealCandidate{},
					FilteringApplied: true,
				}, nil).
				Once()

			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory: "dinner",
			}, token)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring(`"candidates":[]`))
		})
	})
})
