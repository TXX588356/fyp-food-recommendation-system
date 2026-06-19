package endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"fyp/food-rs/internal/service/mealdataset"
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

type fakePrebuiltMealSearcher struct {
	meals []mealdataset.PrebuiltMeal
	err   error
	query string
}

func (s *fakePrebuiltMealSearcher) Search(_ context.Context, query string) ([]mealdataset.PrebuiltMeal, error) {
	s.query = query
	return s.meals, s.err
}

func registerMealSearchTestRoutes(e *echo.Echo, customMealService interfaces.CustomMealService, prebuiltSearcher prebuiltMealSearcher, jwtSecret string) {
	handler := &mealSearchHandler{
		customMealService: customMealService,
		prebuiltSearcher:  prebuiltSearcher,
	}

	meals := e.Group("/meals", middleware.Auth(jwtSecret))
	meals.GET("/search", handler.searchMeals)
}

func performMealSearchRequest(e *echo.Echo, path string, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, bytes.NewReader(nil))
	request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

	if token != "" {
		request.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}

	response := httptest.NewRecorder()
	e.ServeHTTP(response, request)

	return response
}

func generateMealSearchTestToken(userID uuid.UUID, jwtSecret string) string {
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

var _ = Describe("Meal search endpoints", func() {
	const jwtSecret = "test-secret"

	var (
		e                 *echo.Echo
		customMealService *mocks.CustomMealService
		prebuiltSearcher  *fakePrebuiltMealSearcher
		userID            uuid.UUID
		token             string
	)

	BeforeEach(func() {
		e = echo.New()
		customMealService = mocks.NewCustomMealService(GinkgoT())
		prebuiltSearcher = &fakePrebuiltMealSearcher{}
		userID = uuid.New()
		token = generateMealSearchTestToken(userID, jwtSecret)

		registerMealSearchTestRoutes(e, customMealService, prebuiltSearcher, jwtSecret)
	})

	It("should return matching custom meals and prebuilt meals", func() {
		customMealService.EXPECT().
			ListVisible(mock.Anything, userID, "chicken").
			Return([]*interfaces.CustomMealResponse{
				{
					ID:                     uuid.NewString(),
					Name:                   "My Chicken Rice",
					Price:                  8.50,
					Calories:               620,
					FatG:                   18,
					ProteinG:               31,
					CarbsG:                 74,
					DietaryRestrictionTags: []string{"halal"},
					MealCategoryTags:       []string{"rice"},
				},
			}, nil).
			Once()
		prebuiltSearcher.meals = []mealdataset.PrebuiltMeal{
			{
				ID:       "hainanese-chicken-rice",
				Name:     "Hainanese Chicken Rice",
				Calories: 607,
				Protein:  26,
				Carbs:    75,
				Fat:      23,
				Category: mealdataset.CategoryTags{"Rice"},
				ImageURL: "https://example.com/chicken-rice.jpg",
			},
		}

		response := performMealSearchRequest(e, "/meals/search?q=%20chicken%20", token)

		Expect(response.Code).To(Equal(http.StatusOK))

		var result []mealSearchResult
		Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())

		Expect(result).To(HaveLen(2))
		Expect(result[0].Source).To(Equal("custom"))
		Expect(result[0].Name).To(Equal("My Chicken Rice"))
		Expect(result[1].Source).To(Equal("prebuilt"))
		Expect(result[1].Name).To(Equal("Hainanese Chicken Rice"))
		Expect(result[1].ImageURL).To(Equal("https://example.com/chicken-rice.jpg"))
	})

	It("should return fuzzy custom meal matches when the query has a typo", func() {
		customMealService.EXPECT().
			ListVisible(mock.Anything, userID, "chiken rice").
			Return([]*interfaces.CustomMealResponse{}, nil).
			Once()
		customMealService.EXPECT().
			ListVisible(mock.Anything, userID, "").
			Return([]*interfaces.CustomMealResponse{
				{
					ID:       uuid.NewString(),
					Name:     "My Chicken Rice",
					Price:    8.50,
					Calories: 620,
				},
			}, nil).
			Once()

		response := performMealSearchRequest(e, "/meals/search?q=chiken%20rice", token)

		Expect(response.Code).To(Equal(http.StatusOK))

		var result []mealSearchResult
		Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())

		Expect(result).To(HaveLen(1))
		Expect(result[0].Source).To(Equal("custom"))
		Expect(result[0].Name).To(Equal("My Chicken Rice"))
	})

	It("should return an empty result without searching when query is blank", func() {
		response := performMealSearchRequest(e, "/meals/search?q=%20%20", token)

		Expect(response.Code).To(Equal(http.StatusOK))

		var result []mealSearchResult
		Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())
		Expect(result).To(BeEmpty())
	})

	It("should return internal server error when custom meal search fails", func() {
		customMealService.EXPECT().
			ListVisible(mock.Anything, userID, "chicken").
			Return(nil, errors.New("database unavailable")).
			Once()

		response := performMealSearchRequest(e, "/meals/search?q=chicken", token)

		Expect(response.Code).To(Equal(http.StatusInternalServerError))
		Expect(response.Body.String()).To(ContainSubstring("database unavailable"))
	})
})
