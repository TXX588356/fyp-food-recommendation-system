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
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func TestCustomMealEndpoints(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Custom Meal Endpoint Suite")
}

// generateCustomMealTestToken creates a valid JWT for endpoint tests.
func generateCustomMealTestToken(userID uuid.UUID, jwtSecret string) string {
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

// registerCustomMealTestRoutes registers custom-meal routes using a mock service.
func registerCustomMealTestRoutes(e *echo.Echo, service interfaces.CustomMealService, jwtSecret string) {
	handler := &customMealHandler{
		customMealService: service,
	}

	customMeals := e.Group("/custom-meals", middleware.Auth(jwtSecret))

	customMeals.POST("", handler.createCustomMeal)
	customMeals.GET("", handler.listVisibleCustomMeals)
	customMeals.GET("/:id", handler.findVisibleCustomMealByID)
}

// performCustomMealRequest executes an HTTP request against the test router.
func performCustomMealRequest(e *echo.Echo, method, path string, body any, token string) *httptest.ResponseRecorder {
	var requestBody *bytes.Reader

	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		encodedBody, err := json.Marshal(body)
		Expect(err).NotTo(HaveOccurred())

		requestBody = bytes.NewReader(encodedBody)
	}

	request := httptest.NewRequest(method, path, requestBody)
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	if token != "" {
		request.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}

	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)

	return response
}

var _ = Describe("Custom meal endpoints", func() {
	const jwtSecret = "test-secret"

	var (
		e                 *echo.Echo
		customMealService *mocks.CustomMealService
		userID            uuid.UUID
		token             string
	)

	BeforeEach(func() {
		e = echo.New()
		customMealService = mocks.NewCustomMealService(GinkgoT())
		userID = uuid.New()
		token = generateCustomMealTestToken(userID, jwtSecret)

		registerCustomMealTestRoutes(e, customMealService, jwtSecret)
	})

	Describe("POST /custom-meals", func() {
		It("should create a custom meal", func() {
			mealID := uuid.New()

			input := interfaces.CustomMealInput{
				Name:           "Dubai Chocolate",
				Price:          9.50,
				Calories:       400,
				FatG:           50,
				ProteinG:       20,
				CarbsG:         30,
				State:          "Kuala Lumpur",
				District:       "Bangsar",
				RestaurantName: "FamilyMart",
				DietaryRestrictionTags: []string{
					"halal",
				},
				MealCategoryTags: []string{
					"nuts",
					"snacks",
				},
			}

			expectedResponse := &interfaces.CustomMealResponse{
				ID:             mealID.String(),
				Name:           "Dubai Chocolate",
				Price:          9.50,
				Calories:       400,
				FatG:           50,
				ProteinG:       20,
				CarbsG:         30,
				State:          "Kuala Lumpur",
				District:       "Bangsar",
				RestaurantName: "FamilyMart",
				IsOwner:        true,
				IsShared:       false,
			}

			customMealService.EXPECT().
				Create(mock.Anything, userID, input).
				Return(expectedResponse, nil).
				Once()

			response := performCustomMealRequest(e, http.MethodPost, "/custom-meals", input, token)

			Expect(response.Code).To(Equal(http.StatusCreated))

			var result interfaces.CustomMealResponse
			Expect(json.Unmarshal(response.Body.Bytes(), &result)).
				To(Succeed())

			Expect(result.ID).To(Equal(mealID.String()))
			Expect(result.Name).To(Equal("Dubai Chocolate"))
			Expect(result.IsOwner).To(BeTrue())
			Expect(result.IsShared).To(BeFalse())
		})

		It("should reject an invalid request body", func() {
			request := httptest.NewRequest(http.MethodPost, "/custom-meals", bytes.NewBufferString(`{"price": invalid}`))

			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			request.Header.Set(echo.HeaderAuthorization, "Bearer "+token)

			response := httptest.NewRecorder()
			e.ServeHTTP(response, request)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("invalid request body"))
		})

		It("should return bad request when creation fails", func() {
			input := interfaces.CustomMealInput{}

			customMealService.EXPECT().
				Create(mock.Anything, userID, input).
				Return(nil, errors.New("custom meal name is required")).
				Once()

			response := performCustomMealRequest(e, http.MethodPost, "/custom-meals", input, token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("custom meal name is required"))

		})
	})

	Describe("GET /custom-meals", func() {
		It("should return visible custom meals", func() {
			expectedResponse := []*interfaces.CustomMealResponse{
				{
					ID:      uuid.NewString(),
					Name:    "My Chicken Rice",
					IsOwner: true,
				},
				{
					ID:       uuid.NewString(),
					Name:     "Shared Chicken Rice",
					IsShared: true,
				},
			}

			customMealService.EXPECT().ListVisible(mock.Anything, userID, "chicken").
				Return(expectedResponse, nil).
				Once()

			response := performCustomMealRequest(e, http.MethodGet, "/custom-meals?q=%20chicken%20", nil, token)

			Expect(response.Code).To(Equal(http.StatusOK))

			var result []*interfaces.CustomMealResponse
			Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())

			Expect(result[0].IsOwner).To(BeTrue())
			Expect(result[1].IsShared).To(BeTrue())
		})

		It("should return internal server error when listing fails", func() {
			customMealService.EXPECT().
				ListVisible(mock.Anything, userID, "").
				Return(nil, errors.New("database unavailable")).
				Once()

			response := performCustomMealRequest(e, http.MethodGet, "/custom-meals", nil, token)

			Expect(response.Code).To(Equal(http.StatusInternalServerError))
			Expect(response.Body.String()).To(ContainSubstring("database unavailable"))
		})
	})

	Describe("GET /custom-meals/:id", func() {
		It("should return one visible custom meal", func() {
			mealID := uuid.New()

			expectedResponse := &interfaces.CustomMealResponse{
				ID:       mealID.String(),
				Name:     "Shared Chicken Rice",
				IsShared: true,
			}

			customMealService.EXPECT().
				FindVisibleByID(mock.Anything, userID, mealID).
				Return(expectedResponse, nil).
				Once()

			response := performCustomMealRequest(e, http.MethodGet, "/custom-meals/"+mealID.String(), nil, token)

			Expect(response.Code).To(Equal(http.StatusOK))

			var result interfaces.CustomMealResponse

			Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())
			Expect(result.ID).To(Equal(mealID.String()))
			Expect(result.Name).To(Equal("Shared Chicken Rice"))
			Expect(result.IsShared).To(BeTrue())
		})

		It("should reject an invalid custom meal ID", func() {
			response := performCustomMealRequest(e, http.MethodGet, "/custom-meals/not-a-uuid", nil, token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("invalid custom meal id"))
		})

		It("should return not found when the meal is not visible", func() {
			mealID := uuid.New()

			customMealService.EXPECT().
				FindVisibleByID(mock.Anything, userID, mealID).
				Return(nil, errors.New("record not found")).
				Once()

			response := performCustomMealRequest(e, http.MethodGet, "/custom-meals/"+mealID.String(), nil, token)

			Expect(response.Code).To(Equal(http.StatusNotFound))
			Expect(response.Body.String()).To(ContainSubstring("record not found"))
		})
	})

	It("should reject requests without authentication", func() {
		response := performCustomMealRequest(e, http.MethodGet, "/custom-meals", nil, "")

		Expect(response.Code).To(Equal(http.StatusUnauthorized))
		Expect(response.Body.String()).To(ContainSubstring("missing authorization header"))
	})
})
