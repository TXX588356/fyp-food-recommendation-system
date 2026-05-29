package endpoint

import (
	"context"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
)

type preferenceHandler struct {
	preferenceService interfaces.PreferenceService
}

func RegisterPreferenceRoutes(ctx context.Context, e *echo.Echo) {
	a := app.FromContext(ctx)
	if a == nil {
		log.Fatal("app missing from context")
		return
	}
	preferenceService, err := a.GetPreferenceService(ctx)
	if err != nil {
		log.Fatal("failed to get preference service", "error", err)
	}

	h := &preferenceHandler{preferenceService: preferenceService}

	preference := e.Group("/preferences", middleware.Auth(a.JWTSecret))
	preference.GET("", h.getPreferences)
	preference.POST("", h.createPreferences) // Onboarding
	preference.PUT("", h.updatePreferences)
	preference.PUT("/data-sharing", h.updateDataSharing)
}

func (h *preferenceHandler) getPreferences(c *echo.Context) error {
	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	result, err := h.preferenceService.GetByUserID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, result)

}

func (h *preferenceHandler) createPreferences(c *echo.Context) error {
	var input interfaces.PreferenceInput

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

	result, err := h.preferenceService.CompleteOnboarding(c.Request().Context(), userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, result)
}

func (h *preferenceHandler) updatePreferences(c *echo.Context) error {
	var input interfaces.PreferenceInput

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

	result, err := h.preferenceService.Update(c.Request().Context(), userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

func (h *preferenceHandler) updateDataSharing(c *echo.Context) error {
	var input struct {
		DataSharingConsent *bool `json:"dataSharingConsent"`
	}

	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}

	if input.DataSharingConsent == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "dataSharingConsent is required",
		})
	}

	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	if err = h.preferenceService.UpdateDataSharingConsent(c.Request().Context(), userID, *input.DataSharingConsent); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "consent updated",
	})

}

func init() {
	endpoints = append(endpoints, RegisterPreferenceRoutes)
}
