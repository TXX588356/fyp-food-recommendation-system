package recommendation

import (
	"fyp/food-rs/types/model"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("BuildMealHistoryContext", func() {
	It("detects recent repeats and category counts", func() {
		now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
		logs := []model.MealLog{
			{MealName: "Nasi Lemak", MealCategory: []string{"rice_dishes", "malaysian"}, EatenAt: time.Date(2026, time.July, 14, 8, 0, 0, 0, time.UTC)},
			{MealName: "Nasi Lemak", MealCategory: []string{"rice_dishes", "malaysian"}, EatenAt: time.Date(2026, time.July, 12, 8, 0, 0, 0, time.UTC)},
			{MealName: "Chicken Soup", MealCategory: []string{"soups"}, EatenAt: time.Date(2026, time.July, 5, 22, 0, 0, 0, time.UTC)},
		}

		got := buildMealHistoryContext(logs, now)

		Expect(got.RecentMealNames[0]).To(Equal("Nasi Lemak"))
		Expect(got.RepeatedMealNames).To(Equal([]string{"Nasi Lemak"}))
		Expect(got.RecentCategoryCounts).To(HaveKeyWithValue("rice_dishes", 2))
	})
})
