package catalog

import (
	"context"

	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FoodSearcher", func() {
	It("should use default portion and complete macros", func() {
		id := uuid.New()
		repo := &testRepository{meals: []model.PrebuiltMeal{{
			ID: id, Name: "Soup", CategoryCodes: []string{"soups"}, ServingDescription: "bowl",
			Calories: float(240), ProteinG: float(12), CarbsG: float(30), FatG: float(6),
		}}}
		searcher := NewFoodSearcher(NewService(repo, testResolver{}))

		got, found, err := searcher.SearchFood(context.Background(), uuid.New(), "soup")

		Expect(err).NotTo(HaveOccurred())
		Expect(found).To(BeTrue())
		Expect(got.Calories).To(Equal(float64(240)))
		Expect(got.ProteinG).To(Equal(float64(12)))
	})
})
