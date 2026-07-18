package endpoint

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	endpointmiddleware "fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type catalogHandler struct{ service interfaces.CatalogService }

// RegisterCatalogRoutes registers the authenticated public catalogue API.
func RegisterCatalogRoutes(e *echo.Echo, service interfaces.CatalogService, jwtSecret string) {
	group := e.Group("/catalog")
	if jwtSecret != "" {
		group.Use(endpointmiddleware.Auth(jwtSecret))
	}
	h := &catalogHandler{service: service}
	group.GET("/meals", h.search)
	group.GET("/meals/:id", h.detail)
	group.GET("/categories", h.categories)
}

func (h *catalogHandler) search(c *echo.Context) error {
	q := interfaces.CatalogQuery{Query: strings.TrimSpace(c.QueryParam("q")), Categories: c.QueryParams()["category"], Source: strings.TrimSpace(c.QueryParam("source"))}
	if raw := c.QueryParam("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return catalogError(c, http.StatusBadRequest, "invalid limit", "INVALID_QUERY")
		}
		q.Limit = value
	}
	if raw := c.QueryParam("has_image"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return catalogError(c, http.StatusBadRequest, "invalid has_image", "INVALID_QUERY")
		}
		q.HasImage = &value
	}
	if raw := c.QueryParam("cursor"); raw != "" {
		var cur struct {
			Name string    `json:"name"`
			ID   uuid.UUID `json:"id"`
		}
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || json.Unmarshal(decoded, &cur) != nil || cur.ID == uuid.Nil {
			return catalogError(c, http.StatusBadRequest, "invalid cursor", "INVALID_CURSOR")
		}
		q.AfterName, q.AfterID = cur.Name, &cur.ID
	}
	page, err := h.service.SearchMeals(c.Request().Context(), q)
	if err != nil {
		return writeCatalogServiceError(c, err)
	}
	return c.JSON(http.StatusOK, page)
}

func (h *catalogHandler) detail(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return catalogError(c, http.StatusBadRequest, "invalid meal id", "INVALID_ID")
	}
	meal, err := h.service.GetMeal(c.Request().Context(), id)
	if err != nil {
		return writeCatalogServiceError(c, err)
	}
	return c.JSON(http.StatusOK, meal)
}
func (h *catalogHandler) categories(c *echo.Context) error {
	items, err := h.service.ListCategories(c.Request().Context())
	if err != nil {
		return writeCatalogServiceError(c, err)
	}
	return c.JSON(http.StatusOK, items)
}

func catalogError(c *echo.Context, status int, message, code string) error {
	return c.JSON(status, map[string]string{"error": message, "code": code})
}
func writeCatalogServiceError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, interfaces.ErrCatalogNotFound):
		return catalogError(c, http.StatusNotFound, err.Error(), "CATALOG_NOT_FOUND")
	case errors.Is(err, interfaces.ErrInvalidCatalogQuery):
		return catalogError(c, http.StatusBadRequest, err.Error(), "INVALID_QUERY")
	default:
		return catalogError(c, http.StatusInternalServerError, "catalogue request failed", "INTERNAL_ERROR")
	}
}
