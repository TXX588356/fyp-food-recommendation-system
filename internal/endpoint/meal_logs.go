package endpoint

import (
	"context"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

type mealLogHandler struct {
	mealLogService interfaces.MealLogService
}

func RegisterMealLogRoutes(ctx context.Context, e *echo.Echo) {
	a := app.FromContext(ctx)

	if a == nil {
		log.Fatal("app missing from context")
		return
	}

	mealLogService, err := a.GetMealLogService(ctx)
	if err != nil {
		log.Fatal("failed to get meal log service", "error", err)
	}

	h := &mealLogHandler{mealLogService: mealLogService}

	mealLogs := e.Group("/meal-logs", middleware.Auth(a.JWTSecret))
	mealLogs.POST("", h.createMealLog)
	mealLogs.GET("", h.getMealLogsByMonth)
}

func (h *mealLogHandler) createMealLog(c *echo.Context) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	var input interfaces.MealLogInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	result, err := h.mealLogService.Create(c.Request().Context(), userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, result)
}

func (h *mealLogHandler) getMealLogsByMonth(c *echo.Context) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	month := strings.TrimSpace(c.QueryParam("month"))

	result, err := h.mealLogService.GetMonth(c.Request().Context(), userID, month)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

func init() {
	endpoints = append(endpoints, RegisterMealLogRoutes)
}
