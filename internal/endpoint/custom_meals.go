package endpoint

import (
	"context"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type customMealHandler struct {
	customMealService interfaces.CustomMealService
}

func RegisterCustomMealRoutes(ctx context.Context, e *echo.Echo) {
	a := app.FromContext(ctx)

	if a == nil {
		log.Fatal("app missing from context")
		return
	}

	customMealService, err := a.GetCustomMealService(ctx)
	if err != nil {
		log.Fatal("failed to get custom meal service", "error", err)
	}

	h := &customMealHandler{customMealService: customMealService}

	customMeal := e.Group("/custom-meals", middleware.Auth(a.JWTSecret))
	customMeal.POST("", h.createCustomMeal)
	customMeal.GET("", h.listVisibleCustomMeals)
	customMeal.GET("/:id", h.findVisibleCustomMealByID)
}

// createCustomMeal validates the request body and creates a custom meal owned
// by the authenticated user.
func (h *customMealHandler) createCustomMeal(c *echo.Context) error {
	var input interfaces.CustomMealInput

	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	result, err := h.customMealService.Create(c.Request().Context(), userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, result)
}

// listVisibleCustomMeals returns the authenticated user's own custom meals and
// custom meals shared by other users. The optional q param filters by name.
func (h *customMealHandler) listVisibleCustomMeals(c *echo.Context) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	query := strings.TrimSpace(c.QueryParam("q"))

	result, err := h.customMealService.ListVisible(c.Request().Context(), userID, query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// findVisibleCustomMealByID returns one custom meal when it is owned by the
// authenticated user or shared by other user.
func (h *customMealHandler) findVisibleCustomMealByID(c *echo.Context) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	customMealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid custom meal id",
		})
	}

	result, err := h.customMealService.FindVisibleByID(c.Request().Context(), userID, customMealID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// init adds custom-meal route registration to the application's endpoint list
func init() {
	endpoints = append(endpoints, RegisterCustomMealRoutes)
}
