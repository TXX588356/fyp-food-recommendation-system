package foodapi

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFoodAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Food API Suite")
}

var _ = Describe("Kalori food searches", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("HTTP client", func() {
		var (
			server       *httptest.Server
			client       interfaces.FoodSearcher
			responseBody string
			statusCode   int
		)

		BeforeEach(func() {
			responseBody = `{
				"success": true,
				"data": [],
				"count": 0
			}`
			statusCode = http.StatusOK

			server = httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					Expect(r.URL.Query().Get("q")).To(Equal("Nasi Lemak"))
					Expect(r.Header.Get("X-API-KEY")).To(Equal("test-key"))

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(statusCode)

					_, err := w.Write([]byte(responseBody))
					Expect(err).NotTo((HaveOccurred()))
				},
			))

			client = NewKaloriClientWithBaseURL("test-key", server.URL, server.Client())
		})

		AfterEach(func() {
			server.Close()
		})

		It("should map the first API result into the shared food-search DTO", func() {
			responseBody = `{
			"success": true,
			"data": [
			{
				"id": "food-1",
				"name": "Nasi Lemak",	
				"tags": ["rice"],
				"calories": 655,
				"fat_g": 24,
				"protein_g": 16,
				"carbs_g": 84
			},
			{
				"id": "food-2",
				"name": "Other"
			}
			],
				"count": 2
			}`

			food, found, err := client.SearchFood(ctx, "Nasi Lemak")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(food.ID).To(Equal("food-1"))
			Expect(food.Name).To(Equal("Nasi Lemak"))
			Expect(food.Tags).To(Equal([]string{"rice"}))
			Expect(food.Calories).To(Equal(float64(655)))
		})

		It("should return false when the API has no matching food", func() {
			_, found, err := client.SearchFood(ctx, "Nasi Lemak")

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})

		It("should return an error for invalid JSON", func() {
			responseBody = `not-json`

			_, _, err := client.SearchFood(ctx, "Nasi Lemak")
			Expect(err).To(HaveOccurred())
		})

		It("should return an error for a non-success status", func() {
			statusCode = http.StatusServiceUnavailable
			responseBody = `unavailable`

			_, _, err := client.SearchFood(ctx, "Nasi Lemak")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("mock-client", func() {
		var client interfaces.FoodSearcher

		BeforeEach(func() {
			client = NewMockKaloriClient()
		})

		It("should return a know meal", func() {
			food, found, err := client.SearchFood(ctx, "Nasi Lemak")

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(food.ID).To(Equal("mock-nasi-lemak"))
		})

		It("should ignore casing and surrounding whitespace", func() {
			food, found, err := client.SearchFood(ctx, "  WANTAN MEE  ")

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			Expect(food.Name).To(Equal("Wantan Mee"))
		})

		It("should return found false for an unknown meal", func() {
			_, found, err := client.SearchFood(ctx, "Unknown Meal")

			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})
})
