package endpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type stubMealLogService struct {
	updateUserID uuid.UUID
	updateLogID  uuid.UUID
	updateInput  interfaces.MealLogUpdateInput
	updateResult *interfaces.MealLogResponse
	updateErr    error

	deleteUserID uuid.UUID
	deleteLogID  uuid.UUID
	deleteErr    error
}

func (s *stubMealLogService) Create(context.Context, uuid.UUID, interfaces.MealLogInput) (*interfaces.MealLogResponse, error) {
	return nil, nil
}

func (s *stubMealLogService) GetMonth(context.Context, uuid.UUID, string) (*interfaces.MealLogMonthResponse, error) {
	return nil, nil
}

func (s *stubMealLogService) Update(_ context.Context, userID uuid.UUID, logID uuid.UUID, input interfaces.MealLogUpdateInput) (*interfaces.MealLogResponse, error) {
	s.updateUserID = userID
	s.updateLogID = logID
	s.updateInput = input

	return s.updateResult, s.updateErr
}

func (s *stubMealLogService) Delete(_ context.Context, userID uuid.UUID, logID uuid.UUID) error {
	s.deleteUserID = userID
	s.deleteLogID = logID

	return s.deleteErr
}

func registerMealLogTestRoutes(e *echo.Echo, mealLogService interfaces.MealLogService, jwtSecret string) {
	handler := &mealLogHandler{mealLogService: mealLogService}

	mealLogs := e.Group("/meal-logs", middleware.Auth(jwtSecret))
	mealLogs.PATCH("/:id", handler.updateMealLog)
	mealLogs.DELETE("/:id", handler.deleteMealLog)
}

func performMealLogRequest(e *echo.Echo, method string, path string, body any, token string) *httptest.ResponseRecorder {
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

var _ = Describe("Meal log endpoints", func() {
	const jwtSecret = "test-secret"

	var (
		e              *echo.Echo
		mealLogService *stubMealLogService
		userID         uuid.UUID
		token          string
	)

	BeforeEach(func() {
		e = echo.New()
		mealLogService = &stubMealLogService{}
		userID = uuid.New()
		token = generateCustomMealTestToken(userID, jwtSecret)

		registerMealLogTestRoutes(e, mealLogService, jwtSecret)
	})

	Describe("PATCH /meal-logs/:id", func() {
		It("should update the authenticated user's meal log", func() {
			logID := uuid.New()
			eatenAt := time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC)
			mealLogService.updateResult = &interfaces.MealLogResponse{
				ID:       logID.String(),
				MealName: "Chicken Rice",
				MealType: "lunch",
				Price:    9.75,
				EatenAt:  eatenAt,
			}

			response := performMealLogRequest(
				e,
				http.MethodPatch,
				"/meal-logs/"+logID.String(),
				interfaces.MealLogUpdateInput{
					Price:    9.75,
					EatenAt:  eatenAt,
					MealType: "lunch",
				},
				token,
			)

			Expect(response.Code).To(Equal(http.StatusOK))
			Expect(mealLogService.updateUserID).To(Equal(userID))
			Expect(mealLogService.updateLogID).To(Equal(logID))
			Expect(mealLogService.updateInput.Price).To(Equal(9.75))
			Expect(mealLogService.updateInput.EatenAt).To(Equal(eatenAt))
			Expect(mealLogService.updateInput.MealType).To(Equal("lunch"))

			var result interfaces.MealLogResponse
			Expect(json.Unmarshal(response.Body.Bytes(), &result)).To(Succeed())
			Expect(result.ID).To(Equal(logID.String()))
			Expect(result.MealName).To(Equal("Chicken Rice"))
			Expect(result.MealType).To(Equal("lunch"))
			Expect(result.Price).To(Equal(9.75))
		})

		It("should reject an invalid meal log id", func() {
			response := performMealLogRequest(
				e,
				http.MethodPatch,
				"/meal-logs/not-a-uuid",
				interfaces.MealLogUpdateInput{
					Price:    9.75,
					EatenAt:  time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC),
					MealType: "lunch",
				},
				token,
			)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("invalid meal log id"))
			Expect(mealLogService.updateLogID).To(Equal(uuid.Nil))
		})

		It("should return a bad request when the service rejects the update", func() {
			logID := uuid.New()
			mealLogService.updateErr = errors.New("price cannot be negative")

			response := performMealLogRequest(
				e,
				http.MethodPatch,
				"/meal-logs/"+logID.String(),
				interfaces.MealLogUpdateInput{
					Price:    -1,
					EatenAt:  time.Date(2026, 7, 12, 12, 45, 0, 0, time.UTC),
					MealType: "lunch",
				},
				token,
			)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("price cannot be negative"))
			Expect(mealLogService.updateUserID).To(Equal(userID))
			Expect(mealLogService.updateLogID).To(Equal(logID))
		})
	})

	Describe("DELETE /meal-logs/:id", func() {
		It("should delete the authenticated user's meal log", func() {
			logID := uuid.New()

			response := performMealLogRequest(
				e,
				http.MethodDelete,
				"/meal-logs/"+logID.String(),
				nil,
				token,
			)

			Expect(response.Code).To(Equal(http.StatusNoContent))
			Expect(mealLogService.deleteUserID).To(Equal(userID))
			Expect(mealLogService.deleteLogID).To(Equal(logID))
		})

		It("should reject an invalid meal log id", func() {
			response := performMealLogRequest(
				e,
				http.MethodDelete,
				"/meal-logs/not-a-uuid",
				nil,
				token,
			)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("invalid meal log id"))
			Expect(mealLogService.deleteLogID).To(Equal(uuid.Nil))
		})

		It("should return a bad request when the service rejects the delete", func() {
			logID := uuid.New()
			mealLogService.deleteErr = errors.New("meal log not found")

			response := performMealLogRequest(
				e,
				http.MethodDelete,
				"/meal-logs/"+logID.String(),
				nil,
				token,
			)

			Expect(response.Code).To(Equal(http.StatusBadRequest))
			Expect(response.Body.String()).To(ContainSubstring("meal log not found"))
			Expect(mealLogService.deleteUserID).To(Equal(userID))
			Expect(mealLogService.deleteLogID).To(Equal(logID))
		})
	})
})
