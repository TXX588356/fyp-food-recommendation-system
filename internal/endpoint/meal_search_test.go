package endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"fyp/food-rs/types/model"
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

type fakeCatalogService struct {
	meals []interfaces.CatalogMeal
	err   error
	query string
}

func (s *fakeCatalogService) SearchMeals(_ context.Context, query interfaces.CatalogQuery) (interfaces.CatalogMealPage, error) {
	s.query = query.Query
	return interfaces.CatalogMealPage{Items: s.meals}, s.err
}
func (*fakeCatalogService) GetMeal(context.Context, uuid.UUID) (interfaces.CatalogMeal, error) {
	return interfaces.CatalogMeal{}, interfaces.ErrCatalogNotFound
}
func (*fakeCatalogService) ListCategories(context.Context) ([]interfaces.CatalogCategory, error) {
	return nil, nil
}
func (*fakeCatalogService) CreateGeneratedMeal(context.Context, interfaces.GeneratedCatalogMealInput) (interfaces.CatalogMeal, error) {
	return interfaces.CatalogMeal{}, nil
}
func (*fakeCatalogService) ListImages(context.Context, interfaces.CatalogImageQuery) ([]model.PrebuiltMealImage, string, error) {
	return nil, "", nil
}
func (*fakeCatalogService) GetImage(context.Context, uuid.UUID) (*model.PrebuiltMealImage, error) {
	return nil, interfaces.ErrCatalogNotFound
}
func (*fakeCatalogService) ApplyImageAction(context.Context, uuid.UUID, interfaces.CatalogImageAction) (*model.PrebuiltMealImage, error) {
	return nil, interfaces.ErrCatalogNotFound
}

func registerMealSearchTestRoutes(e *echo.Echo, customMealService interfaces.CustomMealService, catalogService interfaces.CatalogService, jwtSecret string) {
	handler := &mealSearchHandler{
		customMealService: customMealService,
		catalogService:    catalogService,
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
		catalogService    *fakeCatalogService
		userID            uuid.UUID
		token             string
	)

	BeforeEach(func() {
		e = echo.New()
		customMealService = mocks.NewCustomMealService(GinkgoT())
		catalogService = &fakeCatalogService{}
		userID = uuid.New()
		token = generateMealSearchTestToken(userID, jwtSecret)

		registerMealSearchTestRoutes(e, customMealService, catalogService, jwtSecret)
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
		calories, protein, carbs, fat := 607.0, 26.0, 75.0, 23.0
		catalogService.meals = []interfaces.CatalogMeal{
			{
				ID:         uuid.New(),
				Name:       "Hainanese Chicken Rice",
				Categories: []string{"rice_dishes"},
				SelectedNutrition: interfaces.CatalogNutrition{
					Calories: &calories,
					ProteinG: &protein,
					CarbsG:   &carbs,
					FatG:     &fat,
				},
				Image: &interfaces.CatalogImage{URL: "https://example.com/chicken-rice.jpg"},
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
