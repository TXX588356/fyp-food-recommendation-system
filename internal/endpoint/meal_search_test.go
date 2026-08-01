package endpoint

import (
	"bytes"
	"context"
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

type fakeCatalogService struct {
	meals    []interfaces.CatalogMeal
	err      error
	query    string
	detail   interfaces.CatalogMeal
	getErr   error
	detailID uuid.UUID
}

func (s *fakeCatalogService) SearchMeals(_ context.Context, query interfaces.CatalogQuery) (interfaces.CatalogMealPage, error) {
	s.query = query.Query
	return interfaces.CatalogMealPage{Items: s.meals}, s.err
}
func (s *fakeCatalogService) GetMeal(_ context.Context, id uuid.UUID) (interfaces.CatalogMeal, error) {
	s.detailID = id
	return s.detail, s.getErr
}
func (*fakeCatalogService) ListCategories(context.Context) ([]interfaces.CatalogCategory, error) {
	return nil, nil
}
func (*fakeCatalogService) CreateGeneratedMeal(context.Context, interfaces.GeneratedCatalogMealInput) (interfaces.CatalogMeal, error) {
	return interfaces.CatalogMeal{}, nil
}

func registerMealSearchTestRoutes(e *echo.Echo, customMealService interfaces.CustomMealService, catalogService interfaces.CatalogService, jwtSecret string) {
	handler := &mealSearchHandler{
		customMealService: customMealService,
		catalogService:    catalogService,
	}

	meals := e.Group("/meals", middleware.Auth(jwtSecret))
	meals.GET("/search", handler.searchMeals)
	meals.GET("/:source/:mealID", handler.getMealDetail)
}

func registerMealDetailTestRoutes(
	e *echo.Echo,
	customMealService interfaces.CustomMealService,
	catalogService interfaces.CatalogService,
	preferenceService interfaces.PreferenceService,
	restaurantSearcher interfaces.RestaurantSearcher,
	jwtSecret string,
) {
	handler := &mealSearchHandler{
		customMealService:  customMealService,
		catalogService:     catalogService,
		preferenceService:  preferenceService,
		restaurantSearcher: restaurantSearcher,
	}

	meals := e.Group("/meals", middleware.Auth(jwtSecret))
	meals.GET("/:source/:mealID", handler.getMealDetail)
}

type fakeRestaurantSearcher struct {
	input  interfaces.RestaurantSearchInput
	result interfaces.RestaurantSearchResult
	err    error
}

func (s *fakeRestaurantSearcher) SearchRestaurants(_ context.Context, input interfaces.RestaurantSearchInput) (interfaces.RestaurantSearchResult, error) {
	s.input = input
	return s.result, s.err
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

	It("should return manual prebuilt meal detail with nutrition and restaurants", func() {
		preferenceService := mocks.NewPreferenceService(GinkgoT())
		restaurantSearcher := &fakeRestaurantSearcher{
			result: interfaces.RestaurantSearchResult{
				Status: interfaces.RestaurantLookupOK,
				Restaurants: []interfaces.RestaurantResult{
					{
						Name:        "Nasi House",
						Address:     "Kuala Lumpur",
						Rating:      4.3,
						ReviewCount: 12,
						Price:       "$$",
					},
				},
			},
		}
		mealID := uuid.New()
		calories, protein, carbs, fat := 500.0, 24.0, 66.0, 16.0
		catalogService.detail = interfaces.CatalogMeal{
			ID:         mealID,
			Name:       "Nasi Lemak",
			Categories: []string{"rice_dishes"},
			SelectedPortion: interfaces.CatalogPortion{
				Description: "1 plate",
			},
			SelectedNutrition: interfaces.CatalogNutrition{
				Calories: &calories,
				ProteinG: &protein,
				CarbsG:   &carbs,
				FatG:     &fat,
			},
		}

		preferenceService.EXPECT().
			GetByUserID(mock.Anything, userID).
			Return(&interfaces.PreferenceResponse{
				HomeLocation:       "Petaling Jaya",
				WorkSchoolLocation: "Kuala Lumpur",
			}, nil).
			Once()

		e = echo.New()
		registerMealDetailTestRoutes(e, customMealService, catalogService, preferenceService, restaurantSearcher, jwtSecret)

		response := performMealSearchRequest(e, "/meals/prebuilt/"+mealID.String(), token)

		Expect(response.Code).To(Equal(http.StatusOK))

		var result map[string]any
		Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())

		Expect(result).To(HaveKey("meal"))
		Expect(result).To(HaveKey("restaurants"))
		Expect(result).NotTo(HaveKey("recommendationExplanation"))
		Expect(result).NotTo(HaveKey("score"))

		meal := result["meal"].(map[string]any)
		Expect(meal["name"]).To(Equal("Nasi Lemak"))
		Expect(meal["source"]).To(Equal("prebuilt"))
		Expect(meal["servingDescription"]).To(Equal("1 plate"))

		nutrition := meal["nutrition"].(map[string]any)
		Expect(nutrition["calories"]).To(BeNumerically("==", 500))
		Expect(nutrition["proteinG"]).To(BeNumerically("==", 24))

		Expect(restaurantSearcher.input.MealName).To(Equal("Nasi Lemak"))
		Expect(restaurantSearcher.input.Location).NotTo(BeEmpty())
	})

	It("should use selected location query for manual meal detail restaurants", func() {
		preferenceService := mocks.NewPreferenceService(GinkgoT())
		restaurantSearcher := &fakeRestaurantSearcher{
			result: interfaces.RestaurantSearchResult{
				Status: interfaces.RestaurantLookupOK,
				Restaurants: []interfaces.RestaurantResult{
					{
						Name:    "Bangsar Nasi House",
						Address: "Bangsar",
					},
				},
			},
		}
		mealID := uuid.New()
		catalogService.detail = interfaces.CatalogMeal{
			ID:   mealID,
			Name: "Nasi Lemak",
		}

		e = echo.New()
		registerMealDetailTestRoutes(e, customMealService, catalogService, preferenceService, restaurantSearcher, jwtSecret)

		response := performMealSearchRequest(e, "/meals/prebuilt/"+mealID.String()+"?location=Bangsar%2C%20Kuala%20Lumpur", token)

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(restaurantSearcher.input.Location).To(Equal("Bangsar, Kuala Lumpur"))

		var result map[string]any
		Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())

		location := result["location"].(map[string]any)
		Expect(location["query"]).To(Equal("Bangsar, Kuala Lumpur"))
		Expect(location["basis"]).To(Equal("selected"))
	})

	It("should hide a manual custom meal restaurant when the selected state does not match", func() {
		preferenceService := mocks.NewPreferenceService(GinkgoT())
		restaurantSearcher := &fakeRestaurantSearcher{
			result: interfaces.RestaurantSearchResult{
				Status:      interfaces.RestaurantLookupOK,
				Restaurants: []interfaces.RestaurantResult{},
			},
		}
		mealID := uuid.New()

		customMealService.EXPECT().
			FindVisibleByID(mock.Anything, userID, mealID).
			Return(&interfaces.CustomMealResponse{
				ID:             mealID.String(),
				Name:           "Sabah Custom Meal",
				State:          "Sabah",
				District:       "Kota Kinabalu",
				RestaurantName: "Sabah Food Place",
				Calories:       500,
			}, nil).
			Once()

		e = echo.New()
		registerMealDetailTestRoutes(e, customMealService, catalogService, preferenceService, restaurantSearcher, jwtSecret)

		response := performMealSearchRequest(e, "/meals/custom/"+mealID.String()+"?location=Kajang%2C%20Selangor", token)

		Expect(response.Code).To(Equal(http.StatusOK))

		var result map[string]any
		Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())

		restaurants := result["restaurants"].([]any)
		Expect(restaurants).To(BeEmpty())
	})
})
