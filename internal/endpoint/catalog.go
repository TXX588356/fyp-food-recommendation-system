package endpoint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"fyp/food-rs/app"
	endpointmiddleware "fyp/food-rs/internal/endpoint/middleware"
	"fyp/food-rs/internal/interfaces"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

type catalogHandler struct{ catalogService interfaces.CatalogService }

// RegisterCatalogRoutes registers the authenticated public catalogue API.
func RegisterCatalogRoutes(ctx context.Context, e *echo.Echo) {
	a := app.FromContext(ctx)

	if a == nil {
		log.Fatal("app missing from context")
		return
	}

	catalogService, err := a.GetCatalogService(ctx)
	if err != nil {
		log.Fatal("failed to get catalog service", "error", err)
	}

	registerCatalogRoutes(e, catalogService, a.JWTSecret)
}

func registerCatalogRoutes(e *echo.Echo, catalogService interfaces.CatalogService, jwtSecret string) {
	group := e.Group("/api/catalog")
	if jwtSecret != "" {
		group.Use(endpointmiddleware.Auth(jwtSecret))
	}

	h := &catalogHandler{catalogService: catalogService}

	group.GET("/meals", h.search)
	group.GET("/meals/:id", h.detail)
	group.GET("/categories", h.categories)
}

func (h *catalogHandler) search(c *echo.Context) error {
	q := interfaces.CatalogQuery{Query: strings.TrimSpace(c.QueryParam("q")), Categories: c.QueryParams()["category"], Source: strings.TrimSpace(c.QueryParam("source"))}
	if raw := c.QueryParam("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid limit"})
		}
		q.Limit = value
	}
	if raw := c.QueryParam("cursor"); raw != "" {
		var cur struct {
			Name string    `json:"name"`
			ID   uuid.UUID `json:"id"`
		}
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil || json.Unmarshal(decoded, &cur) != nil || cur.ID == uuid.Nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid cursor"})
		}
		q.AfterName, q.AfterID = cur.Name, &cur.ID
	}
	page, err := h.catalogService.SearchMeals(c.Request().Context(), q)
	if err != nil {
		return writeCatalogServiceError(c, err)
	}
	return c.JSON(http.StatusOK, page)
}

func (h *catalogHandler) detail(c *echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid meal id"})
	}
	meal, err := h.catalogService.GetMeal(c.Request().Context(), id)
	if err != nil {
		return writeCatalogServiceError(c, err)
	}
	return c.JSON(http.StatusOK, meal)
}

func (h *catalogHandler) categories(c *echo.Context) error {
	items, err := h.catalogService.ListCategories(c.Request().Context())
	if err != nil {
		return writeCatalogServiceError(c, err)
	}
	return c.JSON(http.StatusOK, items)
}

func writeCatalogServiceError(c *echo.Context, err error) error {
	switch {
	case errors.Is(err, interfaces.ErrCatalogNotFound):
		return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, interfaces.ErrInvalidCatalogQuery):
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "catalogue request failed"})
	}
}

func init() {
	endpoints = append(endpoints, RegisterCatalogRoutes)
}
