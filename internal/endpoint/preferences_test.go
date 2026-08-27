package endpoint

import (
	"errors"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

func registerPreferenceTestRoutes(e *echo.Echo, preferenceService interfaces.PreferenceService, jwtSecret string) {
	handler := &preferenceHandler{preferenceService: preferenceService}

	preferences := e.Group("/preferences", middleware.Auth(jwtSecret))
	preferences.GET("", handler.getPreferences)
	preferences.POST("", handler.createPreferences)
	preferences.PUT("", handler.updatePreferences)
	preferences.PUT("/data-sharing", handler.updateDataSharing)
}

var _ = Describe("Preference endpoints", func() {
	const jwtSecret = "test-secret"

	var (
		e                 *echo.Echo
		preferenceService *mocks.PreferenceService
		userID            uuid.UUID
		token             string
	)

	BeforeEach(func() {
		e = echo.New()
		preferenceService = mocks.NewPreferenceService(GinkgoT())
		userID = uuid.New()
		token = generateCustomMealTestToken(userID, jwtSecret)

		registerPreferenceTestRoutes(e, preferenceService, jwtSecret)
	})

	It("should reject unauthenticated preference requests", func() {
		response := performCustomMealRequest(e, http.MethodGet, "/preferences", nil, "")

		Expect(response.Code).To(Equal(http.StatusUnauthorized))
	})

	It("should get saved preferences for the authenticated user", func() {
		consent := true
		expected := &interfaces.PreferenceResponse{
			MainGoal:           "health",
			MonthlyMealBudget:  500,
			DataSharingConsent: &consent,
			HomeLocation:       "Bukit Jalil",
		}

		preferenceService.EXPECT().GetByUserID(mock.Anything, userID).
			Return(expected, nil).
			Once()

		response := performCustomMealRequest(e, http.MethodGet, "/preferences", nil, token)

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Body.String()).To(ContainSubstring(`"mainGoal":"health"`))
		Expect(response.Body.String()).To(ContainSubstring(`"homeLocation":"Bukit Jalil"`))
	})

	It("should create preferences during onboarding", func() {
		consent := true
		input := interfaces.PreferenceInput{
			MainGoal:           "budget",
			MonthlyMealBudget:  400,
			DataSharingConsent: &consent,
			HomeLocation:       "Cheras",
		}
		expected := &interfaces.PreferenceResponse{
			MainGoal:           "budget",
			MonthlyMealBudget:  400,
			DataSharingConsent: &consent,
			HomeLocation:       "Cheras",
		}

		preferenceService.EXPECT().CompleteOnboarding(mock.Anything, userID, input).
			Return(expected, nil).
			Once()

		response := performCustomMealRequest(e, http.MethodPost, "/preferences", input, token)

		Expect(response.Code).To(Equal(http.StatusCreated))
		Expect(response.Body.String()).To(ContainSubstring(`"mainGoal":"budget"`))
	})

	It("should update preferences for the authenticated user", func() {
		consent := false
		input := interfaces.PreferenceInput{
			MainGoal:            "muscle_gain",
			MonthlyMealBudget:   650,
			DataSharingConsent:  &consent,
			WorkSchoolLocation:  "KL Sentral",
			HealthConcerns:      []string{"hypertension"},
			DietaryRestrictions: []string{"halal"},
			PreferredMealTags:   []string{"high-protein"},
		}
		expected := &interfaces.PreferenceResponse{
			MainGoal:           "muscle_gain",
			MonthlyMealBudget:  650,
			DataSharingConsent: &consent,
		}

		preferenceService.EXPECT().Update(mock.Anything, userID, input).
			Return(expected, nil).
			Once()

		response := performCustomMealRequest(e, http.MethodPut, "/preferences", input, token)

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Body.String()).To(ContainSubstring(`"mainGoal":"muscle_gain"`))
	})

	It("should require explicit data sharing consent when updating consent only", func() {
		response := performCustomMealRequest(e, http.MethodPut, "/preferences/data-sharing", map[string]any{}, token)

		Expect(response.Code).To(Equal(http.StatusBadRequest))
		Expect(response.Body.String()).To(ContainSubstring("dataSharingConsent is required"))
	})

	It("should update data sharing consent for the authenticated user", func() {
		preferenceService.EXPECT().UpdateDataSharingConsent(mock.Anything, userID, false).
			Return(nil).
			Once()

		response := performCustomMealRequest(e, http.MethodPut, "/preferences/data-sharing", map[string]any{
			"dataSharingConsent": false,
		}, token)

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Body.String()).To(ContainSubstring("consent updated"))
	})

	It("should return service validation errors when consent cannot be updated", func() {
		preferenceService.EXPECT().UpdateDataSharingConsent(mock.Anything, userID, true).
			Return(errors.New("preference not found")).
			Once()

		response := performCustomMealRequest(e, http.MethodPut, "/preferences/data-sharing", map[string]any{
			"dataSharingConsent": true,
		}, token)

		Expect(response.Code).To(Equal(http.StatusBadRequest))
		Expect(response.Body.String()).To(ContainSubstring("preference not found"))
	})
})
