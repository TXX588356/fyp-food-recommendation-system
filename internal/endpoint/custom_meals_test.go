package endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	echomiddleware "github.com/labstack/echo/v5/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

type recordingImageStorage struct {
	deletedImageURL string
	deleteImageErr  error
}

type stubCustomMealAutocompleter struct {
	response interfaces.CustomMealAutocompleteResponse
	err      error
	input    interfaces.CustomMealAutocompleteInput
}

func (s *stubCustomMealAutocompleter) AutocompleteCustomMeal(_ context.Context, input interfaces.CustomMealAutocompleteInput) (interfaces.CustomMealAutocompleteResponse, error) {
	s.input = input
	return s.response, s.err
}

func (s *recordingImageStorage) UploadMealImage(context.Context, io.Reader, int64, string) (string, string, error) {
	return "", "", nil
}

func (s *recordingImageStorage) DeleteObject(context.Context, string) error {
	return nil
}

func (s *recordingImageStorage) DeleteMealImage(_ context.Context, imageURL string) error {
	s.deletedImageURL = imageURL
	return s.deleteImageErr
}

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
func registerCustomMealTestRoutes(e *echo.Echo, customMealService interfaces.CustomMealService, preferenceService interfaces.PreferenceService, imageStorage interfaces.ImageStorage, customMealAutocompleter interfaces.CustomMealAutocompleter, jwtSecret string) {
	handler := &customMealHandler{
		customMealService:       customMealService,
		preferenceService:       preferenceService,
		imageStorage:            imageStorage,
		customMealAutocompleter: customMealAutocompleter,
	}

	customMeals := e.Group("/custom-meals", middleware.Auth(jwtSecret))

	customMeals.POST("", handler.createCustomMeal, echomiddleware.BodyLimit(maxCustomMealRequestSize))
	customMeals.GET("", handler.listVisibleCustomMeals)
	customMeals.GET("/:id", handler.findVisibleCustomMealByID)
	customMeals.PUT("/:id", handler.updateCustomMeal)
	customMeals.DELETE("/:id", handler.deleteCustomMeal)
	customMeals.POST("/autocomplete", handler.autocompleteCustomMeal)
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
		preferenceService *mocks.PreferenceService
		autocompleter     *stubCustomMealAutocompleter

		imageStorage *recordingImageStorage
		userID       uuid.UUID
		token        string
	)

	BeforeEach(func() {
		e = echo.New()
		customMealService = mocks.NewCustomMealService(GinkgoT())
		preferenceService = mocks.NewPreferenceService(GinkgoT())
		autocompleter = &stubCustomMealAutocompleter{}
		imageStorage = &recordingImageStorage{}
		userID = uuid.New()
		token = generateCustomMealTestToken(userID, jwtSecret)

		registerCustomMealTestRoutes(e, customMealService, preferenceService, imageStorage, autocompleter, jwtSecret)
	})

	Describe("POST /custom-meals/autocomplete", func() {
		It("should autocomplete meal details from a meaningful meal name", func() {
			autocompleter.response = interfaces.CustomMealAutocompleteResponse{
				Calories:               150,
				FatG:                   3,
				ProteinG:               5,
				CarbsG:                 22,
				DietaryRestrictionTags: []string{"vegetarian"},
				MealCategoryTags:       []string{"korean", "vegetables"},
			}

			response := performCustomMealRequest(
				e,
				http.MethodPost,
				"/custom-meals/autocomplete",
				interfaces.CustomMealAutocompleteInput{Name: "Kimchi"},
				token,
			)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(autocompleter.input.Name).To(Equal("Kimchi"))

			var result interfaces.CustomMealAutocompleteResponse
			Expect(json.Unmarshal(response.Body.Bytes(), &result)).
				To(Succeed())

			Expect(result.Calories).To(Equal(150.0))
			Expect(result.MealCategoryTags).To(Equal([]string{"korean", "vegetables"}))
		})

		It("should return unable-to-generate message when AI cannot generate details", func() {
			autocompleter.err = errors.New("unable to generate meal details")

			response := performCustomMealRequest(
				e,
				http.MethodPost,
				"/custom-meals/autocomplete",
				interfaces.CustomMealAutocompleteInput{Name: "Kimchi"},
				token,
			)

			Expect(response.Code).To(Equal(http.StatusInternalServerError))
			Expect(response.Body.String()).To(ContainSubstring("unable to generate meal details"))
		})
	})

	Describe("POST /custom-meals", func() {
		It("should reject a request body larger than the configured limit", func() {
			response := performCustomMealRequest(
				e,
				http.MethodPost,
				"/custom-meals",
				strings.Repeat("x", 6<<20),
				token,
			)

			Expect(response.Code).To(Equal(http.StatusRequestEntityTooLarge))
		})

		It("should create a custom meal", func() {
			consent := true
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

			preferenceService.EXPECT().
				GetByUserID(mock.Anything, userID).
				Return(&interfaces.PreferenceResponse{
					DataSharingConsent: &consent,
				}, nil).
				Once()

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
			consent := true
			request := httptest.NewRequest(http.MethodPost, "/custom-meals", bytes.NewBufferString(`{"price": invalid}`))

			request.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			request.Header.Set(echo.HeaderAuthorization, "Bearer "+token)

			preferenceService.EXPECT().
				GetByUserID(mock.Anything, userID).
				Return(&interfaces.PreferenceResponse{
					DataSharingConsent: &consent,
				}, nil).
				Once()

			response := httptest.NewRecorder()
			e.ServeHTTP(response, request)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("invalid request body"))
		})

		It("should return bad request when creation fails", func() {
			consent := true
			input := interfaces.CustomMealInput{}

			preferenceService.EXPECT().
				GetByUserID(mock.Anything, userID).
				Return(&interfaces.PreferenceResponse{
					DataSharingConsent: &consent,
				}, nil).
				Once()

			customMealService.EXPECT().
				Create(mock.Anything, userID, input).
				Return(nil, errors.New("custom meal name is required")).
				Once()

			response := performCustomMealRequest(e, http.MethodPost, "/custom-meals", input, token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("custom meal name is required"))

		})

		It("should reject creation when consent is unanswered", func() {
			input := interfaces.CustomMealInput{
				Name:           "Private Noodles",
				Price:          8.50,
				Calories:       500,
				FatG:           10,
				ProteinG:       20,
				CarbsG:         70,
				State:          "Selangor",
				District:       "Petaling",
				RestaurantName: "Test Restaurant",
				MealCategoryTags: []string{
					"noodles",
				},
			}

			preferenceService.EXPECT().
				GetByUserID(mock.Anything, userID).
				Return(&interfaces.PreferenceResponse{
					DataSharingConsent: nil,
				}, nil).
				Once()

			response := performCustomMealRequest(e, http.MethodPost, "/custom-meals", input, token)
			Expect(response.Code).To(Equal(http.StatusPreconditionRequired))
			Expect(response.Body.String()).To(ContainSubstring("data sharing consent must be answered"))
		})

		It("should allow creation when consent is false", func() {
			consent := false

			input := interfaces.CustomMealInput{
				Name:           "Private Noodles",
				Price:          8.50,
				Calories:       500,
				FatG:           10,
				ProteinG:       20,
				CarbsG:         70,
				State:          "Selangor",
				District:       "Petaling",
				RestaurantName: "Test Restaurant",
				MealCategoryTags: []string{
					"noodles",
				},
			}

			preferenceService.EXPECT().
				GetByUserID(mock.Anything, userID).
				Return(&interfaces.PreferenceResponse{
					DataSharingConsent: &consent,
				}, nil).
				Once()

			customMealService.EXPECT().
				Create(mock.Anything, userID, input).
				Return(&interfaces.CustomMealResponse{
					ID:      uuid.NewString(),
					Name:    input.Name,
					IsOwner: true,
				}, nil).
				Once()

			response := performCustomMealRequest(
				e,
				http.MethodPost,
				"/custom-meals",
				input,
				token,
			)

			Expect(response.Code).To(Equal(http.StatusCreated))
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

	Describe("PUT /custom-meals/:id", func() {
		It("should update one owned custom meal", func() {
			mealID := uuid.New()
			input := interfaces.CustomMealInput{
				Name:           "Dubai Chocolate with Strawberry",
				Price:          10.50,
				Calories:       420,
				FatG:           52,
				ProteinG:       21,
				CarbsG:         31,
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
				Name:           "Dubai Chocolate with Strawberry",
				Price:          10.50,
				Calories:       420,
				FatG:           52,
				ProteinG:       21,
				CarbsG:         31,
				State:          "Kuala Lumpur",
				District:       "Bangsar",
				RestaurantName: "FamilyMart",
				IsOwner:        true,
			}

			customMealService.EXPECT().
				Update(mock.Anything, userID, mealID, input).
				Return(expectedResponse, nil).
				Once()

			response := performCustomMealRequest(e, http.MethodPut, "/custom-meals/"+mealID.String(), input, token)

			Expect(response.Code).To(Equal(http.StatusOK))

			var result interfaces.CustomMealResponse
			Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())
			Expect(result.ID).To(Equal(mealID.String()))
			Expect(result.Name).To(Equal("Dubai Chocolate with Strawberry"))
			Expect(result.Price).To(Equal(10.50))
		})

		It("should reject an invalid custom meal ID", func() {
			response := performCustomMealRequest(e, http.MethodPut, "/custom-meals/not-a-uuid", interfaces.CustomMealInput{}, token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("invalid custom meal id"))
		})

		It("should return bad request when updating fails", func() {
			mealID := uuid.New()
			input := interfaces.CustomMealInput{}

			customMealService.EXPECT().
				Update(mock.Anything, userID, mealID, input).
				Return(nil, errors.New("custom meal name is required")).
				Once()

			response := performCustomMealRequest(e, http.MethodPut, "/custom-meals/"+mealID.String(), input, token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("custom meal name is required"))
		})
	})

	Describe("DELETE /custom-meals/:id", func() {
		It("should delete one owned custom meal and its image", func() {
			mealID := uuid.New()
			imageURL := "http://localhost:9000/images/custom-meals/test.jpg"

			customMealService.EXPECT().
				FindVisibleByID(mock.Anything, userID, mealID).
				Return(&interfaces.CustomMealResponse{
					ID:       mealID.String(),
					ImageURL: imageURL,
					IsOwner:  true,
				}, nil).
				Once()

			customMealService.EXPECT().
				Delete(mock.Anything, userID, mealID).
				Return(nil).
				Once()

			response := performCustomMealRequest(e, http.MethodDelete, "/custom-meals/"+mealID.String(), nil, token)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(response.Body.String()).To(ContainSubstring("custom meal deleted"))
			Expect(imageStorage.deletedImageURL).To(Equal(imageURL))
		})

		It("should skip image cleanup when the meal has no image", func() {
			mealID := uuid.New()

			customMealService.EXPECT().
				FindVisibleByID(mock.Anything, userID, mealID).
				Return(&interfaces.CustomMealResponse{
					ID:      mealID.String(),
					IsOwner: true,
				}, nil).
				Once()
			customMealService.EXPECT().
				Delete(mock.Anything, userID, mealID).
				Return(nil).
				Once()

			response := performCustomMealRequest(e, http.MethodDelete, "/custom-meals/"+mealID.String(), nil, token)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(imageStorage.deletedImageURL).To(BeEmpty())
		})

		It("should keep deletion successful when image cleanup fails", func() {
			mealID := uuid.New()
			imageURL := "http://localhost:9000/images/custom-meals/test.jpg"
			imageStorage.deleteImageErr = errors.New("MinIO unavailable")

			customMealService.EXPECT().
				FindVisibleByID(mock.Anything, userID, mealID).
				Return(&interfaces.CustomMealResponse{
					ID:       mealID.String(),
					ImageURL: imageURL,
					IsOwner:  true,
				}, nil).
				Once()
			customMealService.EXPECT().
				Delete(mock.Anything, userID, mealID).
				Return(nil).
				Once()

			response := performCustomMealRequest(e, http.MethodDelete, "/custom-meals/"+mealID.String(), nil, token)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(imageStorage.deletedImageURL).To(Equal(imageURL))
		})

		It("should reject an invalid custom meal ID", func() {
			response := performCustomMealRequest(e, http.MethodDelete, "/custom-meals/not-a-uuid", nil, token)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("invalid custom meal id"))
		})

		It("should return not found when deleting fails", func() {
			mealID := uuid.New()

			customMealService.EXPECT().
				FindVisibleByID(mock.Anything, userID, mealID).
				Return(&interfaces.CustomMealResponse{
					ID:      mealID.String(),
					IsOwner: true,
				}, nil).
				Once()

			customMealService.EXPECT().
				Delete(mock.Anything, userID, mealID).
				Return(errors.New("record not found")).
				Once()

			response := performCustomMealRequest(e, http.MethodDelete, "/custom-meals/"+mealID.String(), nil, token)

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
