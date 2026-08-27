package endpoint

import (
	"fyp/food-rs/internal/interfaces"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Recommendation meal detail endpoint helpers", func() {
	validMealDetailRequest := func() mealDetailRequest {
		price := 9.5

		return mealDetailRequest{
			MealCategory: "lunch",
			Candidate: interfaces.MatchedMealCandidate{
				GeneratedMeal: interfaces.GeneratedMeal{
					Name:                "Chicken Rice",
					EstimatedPriceRange: interfaces.PriceRange{Min: 8, Max: 12},
					SodiumLevel:         "MEDIUM",
					SugarLevel:          "LOW",
					PurineRisk:          "LOW",
					HealthFlags: map[string]string{
						"hypertension": "CAUTION",
					},
				},
				Food: interfaces.FoodSearchResult{
					ID:       "food-1",
					Name:     "Chicken Rice",
					Calories: 620,
					FatG:     18,
					ProteinG: 32,
					CarbsG:   72,
					Price:    &price,
				},
			},
		}
	}

	Describe("validateMealDetailRequest", func() {
		It("should accept a valid meal detail request", func() {
			Expect(validateMealDetailRequest(validMealDetailRequest())).To(Succeed())
		})

		DescribeTable("should reject invalid meal detail requests",
			func(mutator func(*mealDetailRequest), expectedErr string) {
				request := validMealDetailRequest()
				mutator(&request)

				Expect(validateMealDetailRequest(request)).To(MatchError(expectedErr))
			},
			Entry("missing meal category", func(request *mealDetailRequest) {
				request.MealCategory = " "
			}, "meal category is required"),
			Entry("unsupported meal category", func(request *mealDetailRequest) {
				request.MealCategory = "supper"
			}, "unsupported meal category: supper"),
			Entry("missing food id", func(request *mealDetailRequest) {
				request.Candidate.Food.ID = " "
			}, "food id is required"),
			Entry("missing food name", func(request *mealDetailRequest) {
				request.Candidate.Food.Name = " "
			}, "food name is required"),
			Entry("negative calories", func(request *mealDetailRequest) {
				request.Candidate.Food.Calories = -1
			}, "calories cannot be negative"),
			Entry("negative price", func(request *mealDetailRequest) {
				price := -1.0
				request.Candidate.Food.Price = &price
			}, "price cannot be negative"),
			Entry("inverted generated price range", func(request *mealDetailRequest) {
				request.Candidate.GeneratedMeal.EstimatedPriceRange = interfaces.PriceRange{Min: 20, Max: 10}
			}, "minimum price cannot exceed maximum price"),
			Entry("unsupported sodium level", func(request *mealDetailRequest) {
				request.Candidate.GeneratedMeal.SodiumLevel = "EXTREME"
			}, "unsupported sodium level: EXTREME"),
			Entry("blank health flag condition", func(request *mealDetailRequest) {
				request.Candidate.GeneratedMeal.HealthFlags = map[string]string{" ": "SAFE"}
			}, "health flag condition cannot be blank"),
			Entry("unsupported health flag", func(request *mealDetailRequest) {
				request.Candidate.GeneratedMeal.HealthFlags = map[string]string{"diabetes": "UNKNOWN"}
			}, "unsupported health flag for diabetes: UNKNOWN"),
		)
	})

	It("should map meal detail service results into the response shape", func() {
		openNow := true

		response := buildMealDetailResponse(interfaces.MealDetailResult{
			Meal: interfaces.MealDetailMeal{
				ID:                 "food-1",
				Name:               "Chicken Rice",
				MealCategory:       "lunch",
				ImageURL:           "https://example.test/chicken-rice.jpg",
				ServingDescription: "1 plate",
				EstimatedPriceRange: interfaces.PriceRange{
					Min: 8,
					Max: 12,
				},
				Nutrition: interfaces.MealDetailNutrition{
					Calories: 620,
					FatG:     18,
					ProteinG: 32,
					CarbsG:   72,
				},
				Signals: interfaces.MealDetailSignals{
					SodiumLevel: "MEDIUM",
					SugarLevel:  "LOW",
					PurineRisk:  "LOW",
					HealthFlags: map[string]string{"hypertension": "CAUTION"},
				},
			},
			RecommendationExplanation: "Balanced enough for today's lunch.",
			Location: interfaces.MealDetailLocation{
				Query: "KL Sentral",
				Basis: "work_school",
			},
			Restaurants: []interfaces.RestaurantResult{
				{
					Name:         "Chicken Rice Shop",
					Address:      "KL Sentral",
					Rating:       4.4,
					ReviewCount:  120,
					Price:        "$$",
					OpenNow:      &openNow,
					ThumbnailURL: "https://example.test/thumb.jpg",
					SourceURL:    "https://example.test/shop",
					Source:       "serpapi",
				},
			},
			RestaurantLookupStatus: interfaces.RestaurantLookupOK,
		})

		Expect(response.Meal.ID).To(Equal("food-1"))
		Expect(response.Meal.Nutrition.Calories).To(Equal(620.0))
		Expect(response.Meal.Signals.HealthFlags).To(Equal(map[string]string{"hypertension": "CAUTION"}))
		Expect(response.RecommendationExplanation).To(Equal("Balanced enough for today's lunch."))
		Expect(response.Location.Query).To(Equal("KL Sentral"))
		Expect(response.Location.Basis).To(Equal("work_school"))
		Expect(response.RestaurantLookupStatus).To(Equal(interfaces.RestaurantLookupOK))
		Expect(response.Restaurants).To(HaveLen(1))
		Expect(response.Restaurants[0].Name).To(Equal("Chicken Rice Shop"))
		Expect(response.Restaurants[0].OpenNow).NotTo(BeNil())
		Expect(*response.Restaurants[0].OpenNow).To(BeTrue())
	})

	It("should copy string slices with whitespace removed", func() {
		Expect(copyTrimmedStringSlice([]string{" halal ", "", " low sugar", " "})).
			To(Equal([]string{"halal", "low sugar"}))
	})

	It("should copy health flags with whitespace removed", func() {
		Expect(copyTrimmedMealDetailHealthFlags(map[string]string{
			" diabetes ": " SAFE ",
			"":           "CAUTION",
			"gout":       " ",
		})).To(Equal(map[string]string{"diabetes": "SAFE"}))
	})

	Describe("selectMealDetailPreferenceLocation", func() {
		It("should prefer work or school location on weekdays", func() {
			location, basis := selectMealDetailPreferenceLocation(interfaces.PreferenceResponse{
				HomeLocation:       "Home",
				WorkSchoolLocation: "Office",
			}, time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC))

			Expect(location).To(Equal("Office"))
			Expect(basis).To(Equal("work_school"))
		})

		It("should prefer home location on weekends", func() {
			location, basis := selectMealDetailPreferenceLocation(interfaces.PreferenceResponse{
				HomeLocation:       "Home",
				WorkSchoolLocation: "Office",
			}, time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC))

			Expect(location).To(Equal("Home"))
			Expect(basis).To(Equal("home"))
		})

		It("should fall back when the preferred location is blank", func() {
			location, basis := selectMealDetailPreferenceLocation(interfaces.PreferenceResponse{
				HomeLocation: "Home",
			}, time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC))

			Expect(location).To(Equal("Home"))
			Expect(basis).To(Equal("fallback_home"))
		})

		It("should report unavailable when no saved location exists", func() {
			location, basis := selectMealDetailPreferenceLocation(interfaces.PreferenceResponse{}, time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC))

			Expect(location).To(BeEmpty())
			Expect(basis).To(Equal("unavailable"))
		})
	})
})
