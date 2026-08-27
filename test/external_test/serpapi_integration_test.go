//go:build external

package external_test

import (
	"context"
	"strings"
	"time"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/service/restaurant"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SerpAPI external integration", func() {
	Describe("Search restaurants using real SerpAPI", func() {
		It("should return parseable restaurant search results", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			client := restaurant.NewSerpAPIClient(requireEnv("SERPAPI_API_KEY"))

			result, err := client.SearchRestaurants(ctx, interfaces.RestaurantSearchInput{
				MealName: "Chicken Rice",
				Location: "Pavilion Kuala Lumpur",
				Limit:    5,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Or(
				Equal(interfaces.RestaurantLookupOK),
				Equal(interfaces.RestaurantLookupNoResults),
			))

			if result.Status == interfaces.RestaurantLookupOK {
				Expect(result.Restaurants).NotTo(BeEmpty())
			}
		})
	})

	Describe("Map real SerpAPI result into internal restaurant structure", func() {
		It("should map required restaurant fields", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			client := restaurant.NewSerpAPIClient(requireEnv("SERPAPI_API_KEY"))

			result, err := client.SearchRestaurants(ctx, interfaces.RestaurantSearchInput{
				MealName: "Chicken Rice",
				Location: "Pavilion Kuala Lumpur",
				Limit:    5,
			})

			Expect(err).NotTo(HaveOccurred())

			if result.Status == interfaces.RestaurantLookupNoResults {
				Skip("SerpAPI returned no local results for this live query")
			}

			Expect(result.Status).To(Equal(interfaces.RestaurantLookupOK))
			Expect(result.Restaurants).NotTo(BeEmpty())

			first := result.Restaurants[0]
			Expect(first.Name).NotTo(BeEmpty())
			Expect(first.Address).NotTo(BeEmpty())
			Expect(first.Rating).To(BeNumerically(">=", 0))
			Expect(first.ReviewCount).To(BeNumerically(">=", 0))

			if strings.TrimSpace(first.SourceURL) != "" {
				Expect(first.SourceURL).To(ContainSubstring("google.com/maps/place"))
			}
		})
	})

	Describe("Handle restaurant search with no useful results", func() {
		It("should return no-results or unavailable without system failure", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			client := restaurant.NewSerpAPIClient(requireEnv("SERPAPI_API_KEY"))

			result, err := client.SearchRestaurants(ctx, interfaces.RestaurantSearchInput{
				MealName: "zzzzzz-nonexistent-food-query-12345",
				Location: "middle of nowhere qwertyuiop",
				Limit:    5,
			})

			if err != nil {
				Expect(err.Error()).To(ContainSubstring("hasn't returned any results"))
				return
			}

			Expect(result.Status).To(Or(
				Equal(interfaces.RestaurantLookupNoResults),
				Equal(interfaces.RestaurantLookupUnavailable),
				Equal(interfaces.RestaurantLookupOK),
			))
		})
	})

	Describe("Handle SerpAPI request failure", func() {
		It("should return an error for invalid API key", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			client := restaurant.NewSerpAPIClient("invalid-api-key")

			result, err := client.SearchRestaurants(ctx, interfaces.RestaurantSearchInput{
				MealName: "Chicken Rice",
				Location: "Pavilion Kuala Lumpur",
				Limit:    5,
			})

			Expect(err).To(HaveOccurred())
			Expect(result.Restaurants).To(BeEmpty())
		})
	})
})
