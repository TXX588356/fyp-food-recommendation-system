package endpoint

import (
	"context"
	"fyp/food-rs/app"
	"fyp/food-rs/internal/interfaces"
	"log"
	"net/http"

	"github.com/labstack/echo/v5"
)

type authHandler struct {
	authService interfaces.AuthService
}

func RegisterAuthRoutes(ctx context.Context, e *echo.Echo) {
	a := app.FromContext(ctx) // read the App instance previously stored in the context
	if a == nil {
		log.Fatal("app missing from context")
		return
	}
	authService, err := a.GetAuthService(ctx)
	if err != nil {
		log.Fatal("failed to get auth service", "error", err)
	}

	h := &authHandler{authService: authService}

	auth := e.Group("/auth")
	auth.POST("/register", h.register)
	auth.POST("/login", h.login)

}

func (h *authHandler) register(c *echo.Context) error {
	var input interfaces.RegisterInput

	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
	}
	result, err := h.authService.Register(c.Request().Context(), input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, result)
}

func (h *authHandler) login(c *echo.Context) error {
	return c.String(http.StatusCreated, "created but yet to implement")

}

func init() {
	endpoints = append(endpoints, RegisterAuthRoutes)
}
