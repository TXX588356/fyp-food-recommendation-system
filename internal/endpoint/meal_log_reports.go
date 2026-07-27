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

type mealLogReportHandler struct {
	reportService interfaces.MealLogReportService
}

func RegisterMealLogReportRoutes(ctx context.Context, e *echo.Echo) {
	a := app.FromContext(ctx)

	if a == nil {
		log.Fatal("app missing from context")
		return
	}

	reportService, err := a.GetMealLogReportService(ctx)
	if err != nil {
		log.Fatal("failed to get meal log report service", "error", err)
	}

	h := &mealLogReportHandler{reportService: reportService}

	// Keep report under meal-logs because it analyses meal-log records.
	mealReports := e.Group("/meal-logs/report", middleware.Auth(a.JWTSecret))
	mealReports.GET("", h.getMonthReport)
}

func (h *mealLogReportHandler) getMonthReport(c *echo.Context) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	month := strings.TrimSpace(c.QueryParam("month"))

	report, err := h.reportService.GenerateMonth(c.Request().Context(), userID, month)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, report)
}

func init() {
	endpoints = append(endpoints, RegisterMealLogReportRoutes)
}
