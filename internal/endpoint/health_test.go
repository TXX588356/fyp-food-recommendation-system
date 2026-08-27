package endpoint

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health endpoints", func() {
	It("should return connected from the health handler", func() {
		e := echo.New()
		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		response := httptest.NewRecorder()
		c := e.NewContext(request, response)

		Expect(healthHandler(c)).To(Succeed())

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Body.String()).To(Equal("connected"))
	})

	It("should register the health route", func() {
		e := echo.New()

		RegisterHealthRoutes(context.Background(), e)

		request := httptest.NewRequest(http.MethodGet, "/health", nil)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusOK))
		Expect(response.Body.String()).To(Equal("connected"))
	})
})

var _ = Describe("Endpoint registry", func() {
	It("should invoke registered endpoint installers", func() {
		originalEndpoints := endpoints
		DeferCleanup(func() {
			endpoints = originalEndpoints
		})

		endpoints = []endpointRegister{
			func(ctx context.Context, e *echo.Echo) {
				e.GET("/registered", func(c *echo.Context) error {
					return c.String(http.StatusAccepted, "registered")
				})
			},
		}
		e := echo.New()

		RegisterEndpoints(context.Background(), e)

		request := httptest.NewRequest(http.MethodGet, "/registered", nil)
		response := httptest.NewRecorder()
		e.ServeHTTP(response, request)

		Expect(response.Code).To(Equal(http.StatusAccepted))
		Expect(response.Body.String()).To(Equal("registered"))
	})
})
