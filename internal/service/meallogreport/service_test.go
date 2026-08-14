package meallogreport

import (
	"context"
	"errors"
	"testing"
	"time"

	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
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

type testMealLogReportRepository struct {
	logs      []model.MealLog
	listErr   error
	listStart time.Time
	listEnd   time.Time
}

func (r *testMealLogReportRepository) Create(context.Context, *model.MealLog) (*model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogReportRepository) ListByUserAndMonth(context.Context, uuid.UUID, time.Time, time.Time) ([]model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogReportRepository) ListByUserAndRange(_ context.Context, _ uuid.UUID, start time.Time, end time.Time) ([]model.MealLog, error) {
	r.listStart = start
	r.listEnd = end
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.logs, nil
}

func (r *testMealLogReportRepository) FindByIDAndUser(context.Context, uuid.UUID, uuid.UUID) (*model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogReportRepository) Update(context.Context, *model.MealLog) (*model.MealLog, error) {
	return nil, nil
}

func (r *testMealLogReportRepository) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
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

	Describe("parseReportPeriodInLocation", func() {
		It("falls back to Singapore time when the provided location is nil for monthly reports", func() {
			start, end, label, err := parseReportPeriodInLocation("month", "2026-07", "", "", "", nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(start.Format(time.RFC3339)).To(Equal("2026-07-01T00:00:00+08:00"))
			Expect(end.Format(time.RFC3339)).To(Equal("2026-08-01T00:00:00+08:00"))
			Expect(label).To(Equal("July 2026"))
		})

		It("falls back to Singapore time when the provided location is nil for weekly reports", func() {
			start, end, label, err := parseReportPeriodInLocation("week", "", "", "2026-07-27", "2026-08-02", nil)

			Expect(err).NotTo(HaveOccurred())
			Expect(start.Format(time.RFC3339)).To(Equal("2026-07-27T00:00:00+08:00"))
			Expect(end.Format(time.RFC3339)).To(Equal("2026-08-03T00:00:00+08:00"))
			Expect(label).To(Equal("Jul 27 - Aug 2, 2026"))
		})
	})

	Describe("Generate", func() {
		It("returns an empty monthly report when no logs exist", func() {
			ctx := context.Background()
			userID := uuid.New()
			repo := &testMealLogReportRepository{}
			prefs := mocks.NewPreferenceService(GinkgoT())
			svc := NewService(repo, prefs)

			result, err := svc.Generate(ctx, userID, "month", "2026-07", "", "", "")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Period.Kind).To(Equal("month"))
			Expect(result.Period.Label).To(Equal("July 2026"))
			Expect(result.Summary.TotalMeals).To(Equal(0))
			Expect(result.CategoryBreakdown).To(BeEmpty())
			Expect(result.Insights).To(BeEmpty())
			Expect(repo.listStart.Format(time.RFC3339)).To(Equal("2026-07-01T00:00:00+08:00"))
			Expect(repo.listEnd.Format(time.RFC3339)).To(Equal("2026-08-01T00:00:00+08:00"))
		})

		It("builds a weekly report and continues when preferences are unavailable", func() {
			ctx := context.Background()
			userID := uuid.New()
			repo := &testMealLogReportRepository{
				logs: []model.MealLog{
					{
						ID:           uuid.New(),
						UserID:       userID,
						MealName:     "Chicken Rice",
						MealCategory: model.StringArray{"rice_dishes", "poultry"},
						Price:        8.50,
						Calories:     650,
						ProteinG:     32,
						CarbsG:       78,
						FatG:         21,
						EatenAt:      time.Date(2026, 7, 27, 12, 0, 0, 0, loc),
						MealType:     "lunch",
					},
					{
						ID:           uuid.New(),
						UserID:       userID,
						MealName:     "Chicken Rice",
						MealCategory: model.StringArray{"rice_dishes"},
						Price:        9.50,
						Calories:     700,
						ProteinG:     28,
						CarbsG:       82,
						FatG:         24,
						EatenAt:      time.Date(2026, 7, 28, 12, 30, 0, 0, loc),
						MealType:     "lunch",
					},
				},
			}
			prefs := mocks.NewPreferenceService(GinkgoT())
			svc := NewService(repo, prefs)

			prefs.EXPECT().
				GetByUserID(ctx, userID).
				Return(nil, errors.New("preferences unavailable")).
				Once()

			result, err := svc.Generate(ctx, userID, "week", "", "", "2026-07-27", "2026-08-02")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Period.Kind).To(Equal("week"))
			Expect(result.Summary.TotalMeals).To(Equal(2))
			Expect(result.Summary.TotalSpent).To(Equal(18.0))
			Expect(result.MacroSummary.TotalProteinG).To(Equal(60.0))
			Expect(result.TopMeals).To(HaveLen(1))
			Expect(result.TopMeals[0].MealName).To(Equal("Chicken Rice"))
			Expect(result.LowDataWarning).NotTo(BeNil())
			Expect(result.Summary.MonthlyMealBudget).To(BeNil())
		})

		It("returns repository errors", func() {
			ctx := context.Background()
			userID := uuid.New()
			repoErr := errors.New("list failed")
			repo := &testMealLogReportRepository{listErr: repoErr}
			prefs := mocks.NewPreferenceService(GinkgoT())
			svc := NewService(repo, prefs)

			result, err := svc.Generate(ctx, userID, "month", "2026-07", "", "", "")

			Expect(err).To(MatchError(repoErr))
			Expect(result).To(BeNil())
		})

		It("rejects unsupported report period", func() {
			ctx := context.Background()
			userID := uuid.New()
			repo := &testMealLogReportRepository{}
			prefs := mocks.NewPreferenceService(GinkgoT())
			svc := NewService(repo, prefs)

			result, err := svc.Generate(ctx, userID, "year", "2026-07", "", "", "")

			Expect(err).To(MatchError("period must be month or week"))
			Expect(result).To(BeNil())
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

			summary := svc.buildSummary(logs, start, end, prefs, "month")

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
			Expect(summary.MonthlyMealBudget).NotTo(BeNil())
			Expect(*summary.MonthlyMealBudget).To(Equal(100.0))
			Expect(summary.BudgetSpendStatus).NotTo(BeNil())
			Expect(*summary.BudgetSpendStatus).To(Equal("on_track"))
		})

		It("predicts overspending when projected spend exceeds monthly budget", func() {
			svc := &service{
				now: func() time.Time {
					return time.Date(2026, 7, 10, 12, 0, 0, 0, loc)
				},
			}

			start := time.Date(2026, 7, 1, 0, 0, 0, 0, loc)
			end := start.AddDate(0, 1, 0)

			logs := []model.MealLog{
				{
					Price:   50,
					EatenAt: time.Date(2026, 7, 1, 8, 0, 0, 0, loc),
				},
			}

			prefs := &interfaces.PreferenceResponse{
				MonthlyMealBudget: 100,
			}

			summary := svc.buildSummary(logs, start, end, prefs, "month")

			Expect(summary.ProjectedMonthSpend).NotTo(BeNil())
			Expect(*summary.ProjectedMonthSpend).To(Equal(155.0))
			Expect(summary.BudgetSpendStatus).NotTo(BeNil())
			Expect(*summary.BudgetSpendStatus).To(Equal("overspending"))
		})

		It("calculates weekly budget outlook from a prorated monthly budget", func() {
			svc := &service{
				now: func() time.Time {
					return time.Date(2026, 7, 28, 12, 0, 0, 0, loc)
				},
			}

			start, end, err := parseWeekInLocation("", "2026-07-27", "2026-08-02", loc)
			Expect(err).NotTo(HaveOccurred())

			logs := []model.MealLog{
				{
					Price:   20,
					EatenAt: time.Date(2026, 7, 27, 8, 0, 0, 0, loc),
				},
			}

			prefs := &interfaces.PreferenceResponse{
				MonthlyMealBudget: 310,
			}

			summary := svc.buildSummary(logs, start, end, prefs, "week")

			Expect(start).To(Equal(time.Date(2026, 7, 27, 0, 0, 0, 0, loc)))
			Expect(end).To(Equal(time.Date(2026, 8, 3, 0, 0, 0, 0, loc)))
			Expect(summary.BudgetLimit).NotTo(BeNil())
			Expect(*summary.BudgetLimit).To(Equal(70.0))
			Expect(summary.ProjectedPeriodSpend).NotTo(BeNil())
			Expect(*summary.ProjectedPeriodSpend).To(Equal(70.0))
			Expect(summary.BudgetLabel).NotTo(BeNil())
			Expect(*summary.BudgetLabel).To(Equal("weekly budget"))
			Expect(summary.BudgetSpendStatus).NotTo(BeNil())
			Expect(*summary.BudgetSpendStatus).To(Equal("on_track"))
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

			summary := svc.buildSummary(logs, start, end, prefs, "month")

			// Requirement: past month should hide remaining budget and projection.
			Expect(summary.MonthlyMealBudget).To(BeNil())
			Expect(summary.RemainingUsableBudget).To(BeNil())
			Expect(summary.ProjectedMonthSpend).To(BeNil())
			Expect(summary.BudgetSpendStatus).To(BeNil())
		})
	})

})
