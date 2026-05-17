package endpoint

import (
	"context"

	"github.com/labstack/echo/v5"
)

// func NewRouter() http.Handler {
// 	mux := http.NewServeMux()

// 	RegisterHealthRoutes(mux)
// 	RegisterAuthRoutes(mux)

// 	return mux
// }

type endpointRegister func(ctx context.Context, e *echo.Echo)

var endpoints = make([]endpointRegister, 0)

func RegisterEndpoints(ctx context.Context, e *echo.Echo) {
	for _, r := range endpoints {
		r(ctx, e)
	}
}
