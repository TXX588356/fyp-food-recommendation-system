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

func registerRecommendationTestRoutes(e *echo.Echo, preferenceService interfaces.PreferenceService, recommendationService interfaces.RecommendationService, jwtSecret string) {
	handler := &recommendationHandler{
		preferenceService:     preferenceService,
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

	Describe("buildMealPromptFromPreferences", func() {
		It("should build meal prompt from saved preferences and request context", func() {
			preferences := interfaces.PreferenceResponse{
				MainGoal:            "eat_healthier",
				MonthlyMealBudget:   600,
				HealthConcerns:      []string{"diabetes"},
				DietaryRestrictions: []string{"halal"},
				PreferredMealTags:   []string{"rice", "healthy"},
			}

			request := generateRecommendationRequest{
				MealCategory:      "lunch",
				CurrentMonthSpent: 150,
				PerMealBudget:     12,
			}

			input := buildMealPromptFromPreferences(preferences, request)

			Expect(input.Goal).To(Equal("eat_healthier"))
			Expect(input.DietaryRestrictions).To(Equal([]string{"halal"}))
			Expect(input.HealthConcerns).To(Equal([]string{"diabetes"}))
			Expect(input.PreferredMealTags).To(Equal([]string{"rice", "healthy"}))
			Expect(input.MealCategory).To(Equal("lunch"))
			Expect(input.MonthlyMealBudget).To(Equal(float64(600)))
			Expect(input.CurrentMonthSpent).To(Equal(float64(150)))
			Expect(input.RemainingBudget).To(Equal(float64(450)))
			Expect(input.PerMealBudget).To(Equal(float64(12)))
		})

		It("should default per meal budget when request value is zero", func() {
			preferences := interfaces.PreferenceResponse{
				MainGoal:          "quick_recommendation",
				MonthlyMealBudget: 600,
			}

			request := generateRecommendationRequest{
				MealCategory:      "lunch",
				CurrentMonthSpent: 150,
				PerMealBudget:     0,
			}

			input := buildMealPromptFromPreferences(preferences, request)

			Expect(input.RemainingBudget).To(Equal(float64(450)))
			Expect(input.PerMealBudget).To(BeNumerically(">", 0))
		})

		It("should calculate dynamic per meal budget from remaining budget and remaining days", func() {
			now := time.Date(2026, time.June, 25, 12, 0, 0, 0, time.Local)

			got := calculateDynamicPerMealBudget(600, 150, now)

			Expect(got).To(Equal(float64(37.5)))
		})

		It("should clamp remaining budget to zero", func() {
			preferences := interfaces.PreferenceResponse{
				MainGoal:          "quick_recommendation",
				MonthlyMealBudget: 300,
			}

			request := generateRecommendationRequest{
				MealCategory:      "snack",
				CurrentMonthSpent: 500,
			}

			input := buildMealPromptFromPreferences(preferences, request)

			Expect(input.RemainingBudget).To(Equal(float64(0)))
		})
	})

	Describe("POST /recommendations", func() {
		const jwtSecret = "test-secret"

		var (
			e                     *echo.Echo
			preferenceService     *mocks.PreferenceService
			recommendationService *mocks.RecommendationService
			userID                uuid.UUID
			token                 string
		)

		BeforeEach(func() {
			e = echo.New()
			preferenceService = mocks.NewPreferenceService(GinkgoT())
			recommendationService = mocks.NewRecommendationService(GinkgoT())
			userID = uuid.New()
			token = generateRecommendationTestToken(userID, jwtSecret)

			registerRecommendationTestRoutes(e, preferenceService, recommendationService, jwtSecret)
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

		It("should generate recommendations from saved preferences", func() {
			preferences := &interfaces.PreferenceResponse{
				MainGoal:            "eat_healthier",
				MonthlyMealBudget:   600,
				HealthConcerns:      []string{"diabetes"},
				DietaryRestrictions: []string{"halal"},
				PreferredMealTags:   []string{"rice"},
			}

			expectedInput := interfaces.MealPromptInput{
				Goal:                "eat_healthier",
				DietaryRestrictions: []string{"halal"},
				HealthConcerns:      []string{"diabetes"},
				PreferredMealTags:   []string{"rice"},
				MealCategory:        "lunch",
				MonthlyMealBudget:   600,
				CurrentMonthSpent:   150,
				RemainingBudget:     450,
				PerMealBudget:       12,
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

			preferenceService.EXPECT().GetByUserID(mock.Anything, userID).
				Return(preferences, nil).
				Once()

			recommendationService.EXPECT().GenerateCandidates(mock.Anything, userID, expectedInput).
				Return(candidates, nil).
				Once()

			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory:      "lunch",
				CurrentMonthSpent: 150,
				PerMealBudget:     12,
			}, token)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring(`"candidates"`))
			Expect(response.Body.String()).To(ContainSubstring(`"Nasi Lemak"`))
			Expect(response.Body.String()).To(ContainSubstring(`"matched_query":"Nasi Lemak"`))
		})

		It("should return an error when saved preferences cannot be loaded", func() {
			preferenceService.EXPECT().GetByUserID(mock.Anything, userID).
				Return(nil, errors.New("preferences missing")).
				Once()

			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory: "lunch",
			}, token)

			Expect(response.Code).To(Equal(http.StatusInternalServerError))
			Expect(response.Body.String()).To(ContainSubstring("failed to load preferences"))
		})

		It("should return an error when recommendation generation fails", func() {
			preferences := &interfaces.PreferenceResponse{
				MainGoal:          "quick_recommendation",
				MonthlyMealBudget: 600,
			}

			preferenceService.EXPECT().GetByUserID(mock.Anything, userID).
				Return(preferences, nil).
				Once()

			recommendationService.EXPECT().GenerateCandidates(mock.Anything, userID, mock.Anything).
				Return(nil, errors.New("Gemini unavailable")).
				Once()

			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory: "lunch",
			}, token)

			Expect(response.Code).To(Equal(http.StatusInternalServerError))
			Expect(response.Body.String()).To(ContainSubstring("failed to generate recommendations"))
		})

		It("should return empty candidates when no generated meals can be matched", func() {
			preferences := &interfaces.PreferenceResponse{
				MainGoal:          "quick_recommendation",
				MonthlyMealBudget: 600,
			}

			preferenceService.EXPECT().GetByUserID(mock.Anything, userID).
				Return(preferences, nil).
				Once()

			recommendationService.EXPECT().GenerateCandidates(mock.Anything, userID, mock.Anything).
				Return([]interfaces.MatchedMealCandidate{}, nil).
				Once()

			response := performRecommendationRequest(e, generateRecommendationRequest{
				MealCategory: "dinner",
			}, token)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring(`"candidates":[]`))
		})
	})
})
