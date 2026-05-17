package endpoint

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"
)

func RegisterHealthRoutes(ctx context.Context, e *echo.Echo) {
	e.GET("/health", healthHandler)
}

func healthHandler(c *echo.Context) error {
	return c.String(http.StatusOK, "connected")
}

func init() {
	endpoints = append(endpoints, RegisterHealthRoutes)
}
