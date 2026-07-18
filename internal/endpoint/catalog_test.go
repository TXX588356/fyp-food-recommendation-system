package endpoint

import (
	"context"
	"net/http"
	"net/http/httptest"

	"fyp/food-rs/internal/interfaces"
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type catalogServiceStub struct{ query interfaces.CatalogQuery }

func (s *catalogServiceStub) SearchMeals(_ context.Context, q interfaces.CatalogQuery) (interfaces.CatalogMealPage, error) {
	s.query = q
	return interfaces.CatalogMealPage{}, nil
}
func (s *catalogServiceStub) GetMeal(context.Context, uuid.UUID) (interfaces.CatalogMeal, error) {
	return interfaces.CatalogMeal{}, nil
}
func (s *catalogServiceStub) ListCategories(context.Context) ([]interfaces.CatalogCategory, error) {
	return nil, nil
}
func (s *catalogServiceStub) CreateGeneratedMeal(context.Context, interfaces.GeneratedCatalogMealInput) (interfaces.CatalogMeal, error) {
	return interfaces.CatalogMeal{}, nil
}

var _ = Describe("Catalog endpoint", func() {
	It("parses repeated categories", func() {
		e := echo.New()
		service := &catalogServiceStub{}
		RegisterCatalogRoutes(e, service, "")
		req := httptest.NewRequest(http.MethodGet, "/catalog/meals?category=rice_dishes&category=poultry", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		Expect(rec.Code).To(Equal(http.StatusOK), rec.Body.String())
		Expect(service.query.Categories).To(HaveLen(2))
	})
})
