package meallogreport

import (
	"testing"
	"time"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestMealLogReportService runs the meal-log report service test suite.
func TestMealLogReportService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Meal Log Report Service Suite")
}

var _ = Describe("Meal log report service", func() {
	var loc *time.Location

	BeforeEach(func() {
		var err error
		loc, err = time.LoadLocation("Asia/Singapore")
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("buildCategoryBreakdown", func() {
		It("counts every category tag on each meal", func() {
			logs := []model.MealLog{
				{
					MealName:     "Nasi Lemak",
					MealCategory: model.StringArray{"rice_dishes", "fried_foods", "poultry"},
				},
				{
					MealName:     "Fried Rice",
					MealCategory: model.StringArray{"rice_dishes", "fried_foods"},
				},
			}

			result := buildCategoryBreakdown(logs)

			byCategory := map[string]interfaces.CategoryMetric{}
			for _, item := range result {
				byCategory[item.Category] = item
			}

			// Each tag gets one count. Percentages are based on total meals.
			Expect(byCategory).To(HaveKey("rice_dishes"))
			Expect(byCategory["rice_dishes"].Count).To(Equal(2))
			Expect(byCategory["rice_dishes"].Percentage).To(Equal(100.0))

			Expect(byCategory).To(HaveKey("fried_foods"))
			Expect(byCategory["fried_foods"].Count).To(Equal(2))
			Expect(byCategory["fried_foods"].Percentage).To(Equal(100.0))

			Expect(byCategory).To(HaveKey("poultry"))
			Expect(byCategory["poultry"].Count).To(Equal(1))
			Expect(byCategory["poultry"].Percentage).To(Equal(50.0))
		})
	})

	Describe("buildTimePatterns", func() {
		It("calculates usual eating time for breakfast, lunch, dinner, and other windows", func() {
			logs := []model.MealLog{
				{MealType: "breakfast", EatenAt: time.Date(2026, 7, 1, 8, 0, 0, 0, loc)},
				{MealType: "breakfast", EatenAt: time.Date(2026, 7, 2, 10, 0, 0, 0, loc)},
				{MealType: "lunch", EatenAt: time.Date(2026, 7, 1, 12, 0, 0, 0, loc)},
				{MealType: "dinner", EatenAt: time.Date(2026, 7, 1, 19, 0, 0, 0, loc)},
				{MealType: "other", EatenAt: time.Date(2026, 7, 1, 23, 0, 0, 0, loc)},
			}

			result := buildTimePatterns(logs, loc)

			byWindow := map[string]interfaces.TimeWindowMetric{}
			for _, item := range result {
				byWindow[item.Window] = item
			}

			Expect(byWindow["breakfast"].Count).To(Equal(2))
			Expect(byWindow["breakfast"].UsualTime).To(Equal("09:00"))
			Expect(byWindow["breakfast"].UsualTimeMinutes).To(Equal(540))

			Expect(byWindow["lunch"].Count).To(Equal(1))
			Expect(byWindow["lunch"].UsualTime).To(Equal("12:00"))
			Expect(byWindow["lunch"].UsualTimeMinutes).To(Equal(720))

			Expect(byWindow["dinner"].Count).To(Equal(1))
			Expect(byWindow["dinner"].UsualTime).To(Equal("19:00"))
			Expect(byWindow["dinner"].UsualTimeMinutes).To(Equal(1140))

			Expect(byWindow["other"].Count).To(Equal(1))
			Expect(byWindow["other"].UsualTime).To(Equal("23:00"))
			Expect(byWindow["other"].UsualTimeMinutes).To(Equal(1380))
		})
	})

	Describe("buildDailyTrends", func() {
		It("groups spend, calories, and meal counts by local day", func() {
			logs := []model.MealLog{
				{
					Price:    10.50,
					Calories: 500,
					EatenAt:  time.Date(2026, 7, 1, 8, 0, 0, 0, loc),
				},
				{
					Price:    12.25,
					Calories: 650,
					EatenAt:  time.Date(2026, 7, 1, 19, 0, 0, 0, loc),
				},
				{
					Price:    8,
					Calories: 450,
					EatenAt:  time.Date(2026, 7, 2, 12, 0, 0, 0, loc),
				},
			}

			result := buildDailyTrends(logs, loc)

			Expect(result).To(HaveLen(2))

			Expect(result[0].Date).To(Equal("2026-07-01"))
			Expect(result[0].DayLabel).To(Equal("Jul 1"))
			Expect(result[0].MealCount).To(Equal(2))
			Expect(result[0].TotalSpent).To(Equal(22.75))
			Expect(result[0].TotalCalories).To(Equal(1150.0))

			Expect(result[1].Date).To(Equal("2026-07-02"))
			Expect(result[1].DayLabel).To(Equal("Jul 2"))
			Expect(result[1].MealCount).To(Equal(1))
			Expect(result[1].TotalSpent).To(Equal(8.0))
			Expect(result[1].TotalCalories).To(Equal(450.0))
		})
	})

	Describe("buildMacroSummary", func() {
		It("calculates total, average, and calorie-share macro values", func() {
			logs := []model.MealLog{
				{ProteinG: 30, CarbsG: 70, FatG: 20},
				{ProteinG: 20, CarbsG: 50, FatG: 10},
			}

			result := buildMacroSummary(logs)

			Expect(result.TotalProteinG).To(Equal(50.0))
			Expect(result.TotalCarbsG).To(Equal(120.0))
			Expect(result.TotalFatG).To(Equal(30.0))
			Expect(result.AverageProteinG).To(Equal(25.0))
			Expect(result.AverageCarbsG).To(Equal(60.0))
			Expect(result.AverageFatG).To(Equal(15.0))

			// Protein/carbs use 4 kcal per gram; fat uses 9 kcal per gram.
			Expect(result.ProteinCalorieShare).To(Equal(21.05))
			Expect(result.CarbsCalorieShare).To(Equal(50.53))
			Expect(result.FatCalorieShare).To(Equal(28.42))
		})
	})

	Describe("buildSummary", func() {
		It("calculates current-month budget remaining and projected spend", func() {
			svc := &service{
				now: func() time.Time {
					return time.Date(2026, 7, 10, 12, 0, 0, 0, loc)
				},
			}

			start := time.Date(2026, 7, 1, 0, 0, 0, 0, loc)
			end := start.AddDate(0, 1, 0)

			logs := []model.MealLog{
				{
					ID:       uuid.New(),
					Price:    10,
					Calories: 500,
					EatenAt:  time.Date(2026, 7, 1, 8, 0, 0, 0, loc),
				},
				{
					ID:       uuid.New(),
					Price:    20,
					Calories: 700,
					EatenAt:  time.Date(2026, 7, 2, 12, 0, 0, 0, loc),
				},
			}

			prefs := &interfaces.PreferenceResponse{
				MonthlyMealBudget: 100,
			}

			summary := svc.buildSummary(logs, start, end, prefs)

			Expect(summary.TotalMeals).To(Equal(2))
			Expect(summary.ActiveLoggingDays).To(Equal(2))
			Expect(summary.TotalSpent).To(Equal(30.0))
			Expect(summary.AveragePricePerMeal).To(Equal(15.0))
			Expect(summary.AverageDailySpend).To(Equal(15.0))
			Expect(summary.TotalCalories).To(Equal(1200.0))
			Expect(summary.AverageCaloriesPerMeal).To(Equal(600.0))

			Expect(summary.RemainingUsableBudget).NotTo(BeNil())
			Expect(*summary.RemainingUsableBudget).To(Equal(70.0))

			// July has 31 days. RM30 spent by day 10 projects to RM93.
			Expect(summary.ProjectedMonthSpend).NotTo(BeNil())
			Expect(*summary.ProjectedMonthSpend).To(Equal(93.0))
		})

		It("hides budget values for past months", func() {
			svc := &service{
				now: func() time.Time {
					return time.Date(2026, 8, 10, 12, 0, 0, 0, loc)
				},
			}

			start := time.Date(2026, 7, 1, 0, 0, 0, 0, loc)
			end := start.AddDate(0, 1, 0)

			logs := []model.MealLog{
				{
					Price:   30,
					EatenAt: time.Date(2026, 7, 2, 12, 0, 0, 0, loc),
				},
			}

			prefs := &interfaces.PreferenceResponse{
				MonthlyMealBudget: 100,
			}

			summary := svc.buildSummary(logs, start, end, prefs)

			// Requirement: past month should hide remaining budget and projection.
			Expect(summary.RemainingUsableBudget).To(BeNil())
			Expect(summary.ProjectedMonthSpend).To(BeNil())
		})
	})

})
