package endpoint

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	echomiddleware "github.com/labstack/echo/v5/middleware"
)

const (
	maxMealImageSize         int64 = 5 << 20 // 5 MiB
	maxCustomMealRequestSize int64 = 6 << 20 // 6 MiB
)

type customMealHandler struct {
	customMealService interfaces.CustomMealService
	imageStorage      interfaces.ImageStorage
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

	h := &customMealHandler{customMealService: customMealService, imageStorage: a.ImageStorage}

	customMeal := e.Group("/custom-meals", middleware.Auth(a.JWTSecret))
	customMeal.POST("", h.createCustomMeal, echomiddleware.BodyLimit(maxCustomMealRequestSize))
	customMeal.GET("", h.listVisibleCustomMeals)
	customMeal.GET("/:id", h.findVisibleCustomMealByID)
	customMeal.PUT("/:id", h.updateCustomMeal)
	customMeal.DELETE("/:id", h.deleteCustomMeal)
}

// createCustomMeal validates the request body and creates a custom meal owned
// by the authenticated user.
func (h *customMealHandler) createCustomMeal(c *echo.Context) error {
	var input interfaces.CustomMealInput
	var uploadedObjectName string

	contentType := c.Request().Header.Get(echo.HeaderContentType)

	if strings.HasPrefix(contentType, echo.MIMEMultipartForm) {
		payload := c.FormValue("payload")
		if payload == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "missing custom meal payload",
			})
		}

		if err := json.Unmarshal([]byte(payload), &input); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid custom meal payload",
			})
		}

		input.ImageURL = ""

		imageHeader, err := c.FormFile("image")
		if err != nil && errors.Is(err, http.ErrMissingFile) {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid image upload",
			})
		}

		if imageHeader != nil {
			if imageHeader.Size > maxMealImageSize {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "image must not excced 5 MB",
				})
			}

			imageFile, err := imageHeader.Open()
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "could not open uploaded image",
				})
			}
			defer imageFile.Close()

			detectedType, err := detectMealImageContentType(imageFile)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": err.Error(),
				})
			}

			objectName, publicURL, err := h.imageStorage.UploadMealImage(c.Request().Context(), imageFile, imageHeader.Size, detectedType)
			if err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "could not upload meal image",
				})
			}

			uploadedObjectName = objectName
			input.ImageURL = publicURL
		}

	} else {
		if err := c.Bind(&input); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		input.ImageURL = ""
	}

	userID, err := middleware.UserIDFromContext(c)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": err.Error(),
		})
	}

	result, err := h.customMealService.Create(c.Request().Context(), userID, input)
	if err != nil {
		if uploadedObjectName != "" {
			_ = h.imageStorage.DeleteObject(c.Request().Context(), uploadedObjectName)
		}
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

// updateCustomMeal updates a meal if the meal is owned by the authenticated user.
func (h *customMealHandler) updateCustomMeal(c *echo.Context) error {
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

	customMealID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid custom meal id",
		})
	}

	result, err := h.customMealService.Update(c.Request().Context(), userID, customMealID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, result)
}

// deleteCustomMeal deletes a meal if the meal is owned by the authenticated user.
func (h *customMealHandler) deleteCustomMeal(c *echo.Context) error {
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

	meal, err := h.customMealService.FindVisibleByID(c.Request().Context(), userID, customMealID)
	if err != nil || meal == nil || !meal.IsOwner {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "custom meal not found",
		})
	}

	if err := h.customMealService.Delete(c.Request().Context(), userID, customMealID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": err.Error(),
		})
	}

	if meal.ImageURL != "" && h.imageStorage != nil {
		if err := h.imageStorage.DeleteMealImage(c.Request().Context(), meal.ImageURL); err != nil {
			log.Printf("custom meal deleted but image cleanup failed: %v", err)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "custom meal deleted",
	})
}

func detectMealImageContentType(file multipart.File) (string, error) {
	buffer := make([]byte, 512)

	count, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buffer[:count])

	switch contentType {
	case "image/jpeg", "image/png":
		return contentType, nil
	default:
		return "", fmt.Errorf("only JPEG and PNG images are allowed")
	}
}

// init adds custom-meal route registration to the application's endpoint list
func init() {
	endpoints = append(endpoints, RegisterCustomMealRoutes)
}
