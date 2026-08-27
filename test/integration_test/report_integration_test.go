//go:build integration

package integration_test

import (
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	meallogreportservice "fyp/food-rs/internal/service/meallogreport"
	preferenceservice "fyp/food-rs/internal/service/preference"
	"fyp/food-rs/types/model"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var _ = Describe("Meal log report integration", func() {
	newReportService := func(tx *gorm.DB) interfaces.MealLogReportService {
		mealLogRepo := postgres.NewMealLogPostgresRepository(tx)
		preferenceSvc := preferenceservice.NewService(
			postgres.NewPreferencePostgresRepository(tx),
			postgres.NewUserPostgresRepository(tx),
		)

		return meallogreportservice.NewService(mealLogRepo, preferenceSvc)
	}

	floatPtr := func(value float64) *float64 {
		return &value
	}

	createTestUser := func(tx *gorm.DB) model.User {
		user := model.User{
			Name:         "Report User",
			Email:        fmt.Sprintf("integration-%s@example.com", uuid.NewString()),
			PasswordHash: "hashed-password",
		}

		Expect(tx.Create(&user).Error).NotTo(HaveOccurred())
		Expect(user.ID).NotTo(BeZero())

		return user
	}

	createPrebuiltMeal := func(tx *gorm.DB, name string) model.PrebuiltMeal {
		meal := model.PrebuiltMeal{
			SourceCode:         "integration",
			SourceRecordID:     uuid.NewString(),
			Name:               name,
			CategoryCodes:      model.StringArray{"malaysian", "rice_dishes"},
			ServingDescription: "1 serving",
			Calories:           floatPtr(500),
			ProteinG:           floatPtr(25),
			CarbsG:             floatPtr(60),
			FatG:               floatPtr(15),
		}

		Expect(tx.Create(&meal).Error).NotTo(HaveOccurred())
		Expect(meal.ID).NotTo(BeZero())

		return meal
	}

	createMealLog := func(
		tx *gorm.DB,
		userID uuid.UUID,
		meal model.PrebuiltMeal,
		name string,
		price float64,
		eatenAt time.Time,
		calories float64,
		proteinG float64,
		carbsG float64,
		fatG float64,
	) {
		log := model.MealLog{
			UserID:         userID,
			PrebuiltMealID: &meal.ID,
			MealName:       name,
			Price:          price,
			EatenAt:        eatenAt,
			MealType:       "lunch",
			MealCategory:   meal.CategoryCodes,
			Calories:       calories,
			ProteinG:       proteinG,
			CarbsG:         carbsG,
			FatG:           fatG,
		}

		Expect(tx.Create(&log).Error).NotTo(HaveOccurred())
		Expect(log.ID).NotTo(BeZero())
	}

	completePreferences := func(tx *gorm.DB, userID uuid.UUID, monthlyBudget float64) {
		ctx := GinkgoT().Context()
		consent := true

		preferenceSvc := preferenceservice.NewService(
			postgres.NewPreferencePostgresRepository(tx),
			postgres.NewUserPostgresRepository(tx),
		)

		_, err := preferenceSvc.CompleteOnboarding(ctx, userID, interfaces.PreferenceInput{
			MainGoal:            "eat_healthier",
			MonthlyMealBudget:   monthlyBudget,
			DataSharingConsent:  &consent,
			HomeLocation:        "Bukit Jalil",
			WorkSchoolLocation:  "KL Sentral",
			HealthConcerns:      []string{"high_blood_pressure"},
			DietaryRestrictions: []string{"halal"},
			PreferredMealTags:   []string{"malaysian", "rice_dishes"},
		})
		Expect(err).NotTo(HaveOccurred())
	}

	Describe("Generate weekly summary report", func() {
		It("should include only records within the selected week", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			meal := createPrebuiltMeal(tx, "Weekly Report Meal")
			svc := newReportService(tx)

			loc := time.FixedZone("Asia/Singapore", 8*60*60)

			createMealLog(tx, user.ID, meal, "Monday Meal", 10,
				time.Date(2026, 7, 6, 12, 0, 0, 0, loc), 400, 20, 50, 10)

			createMealLog(tx, user.ID, meal, "Sunday Meal", 15,
				time.Date(2026, 7, 12, 18, 0, 0, 0, loc), 500, 25, 60, 15)

			createMealLog(tx, user.ID, meal, "Outside Week Meal", 20,
				time.Date(2026, 7, 13, 12, 0, 0, 0, loc), 600, 30, 70, 20)

			report, err := svc.Generate(ctx, user.ID, "week", "", "", "2026-07-06", "2026-07-12")

			Expect(err).NotTo(HaveOccurred())
			Expect(report).NotTo(BeNil())
			Expect(report.Period.Kind).To(Equal("week"))
			Expect(report.Summary.TotalMeals).To(Equal(2))
			Expect(report.Summary.TotalSpent).To(Equal(25.0))
			Expect(report.Summary.TotalCalories).To(Equal(900.0))

			names := make([]string, 0, len(report.TopExpensiveMeals))
			for _, item := range report.TopExpensiveMeals {
				names = append(names, item.MealName)
			}

			Expect(names).To(ContainElement("Monday Meal"))
			Expect(names).To(ContainElement("Sunday Meal"))
			Expect(names).NotTo(ContainElement("Outside Week Meal"))
		})
	})

	Describe("Generate monthly summary report", func() {
		It("should include only records within the selected month", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			meal := createPrebuiltMeal(tx, "Monthly Report Meal")
			svc := newReportService(tx)

			loc := time.FixedZone("Asia/Singapore", 8*60*60)

			createMealLog(tx, user.ID, meal, "July Meal 1", 10,
				time.Date(2026, 7, 3, 12, 0, 0, 0, loc), 400, 20, 50, 10)

			createMealLog(tx, user.ID, meal, "July Meal 2", 15,
				time.Date(2026, 7, 20, 12, 0, 0, 0, loc), 500, 25, 60, 15)

			createMealLog(tx, user.ID, meal, "August Meal", 20,
				time.Date(2026, 8, 1, 12, 0, 0, 0, loc), 600, 30, 70, 20)

			report, err := svc.GenerateMonth(ctx, user.ID, "2026-07")

			Expect(err).NotTo(HaveOccurred())
			Expect(report.Period.Kind).To(Equal("month"))
			Expect(report.Period.Label).To(Equal("July 2026"))
			Expect(report.Summary.TotalMeals).To(Equal(2))
			Expect(report.Summary.TotalSpent).To(Equal(25.0))
			Expect(report.Summary.TotalCalories).To(Equal(900.0))
		})
	})

	Describe("Calculate total spending", func() {
		It("should calculate total spending as RM45", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			meal := createPrebuiltMeal(tx, "Spending Report Meal")
			svc := newReportService(tx)

			loc := time.FixedZone("Asia/Singapore", 8*60*60)

			createMealLog(tx, user.ID, meal, "Meal RM10", 10,
				time.Date(2026, 7, 3, 12, 0, 0, 0, loc), 400, 20, 50, 10)

			createMealLog(tx, user.ID, meal, "Meal RM15", 15,
				time.Date(2026, 7, 4, 12, 0, 0, 0, loc), 500, 25, 60, 15)

			createMealLog(tx, user.ID, meal, "Meal RM20", 20,
				time.Date(2026, 7, 5, 12, 0, 0, 0, loc), 600, 30, 70, 20)

			report, err := svc.GenerateMonth(ctx, user.ID, "2026-07")

			Expect(err).NotTo(HaveOccurred())
			Expect(report.Summary.TotalMeals).To(Equal(3))
			Expect(report.Summary.TotalSpent).To(Equal(45.0))
			Expect(report.Summary.AveragePricePerMeal).To(Equal(15.0))
		})
	})

	Describe("Calculate remaining monthly budget", func() {
		It("should calculate remaining budget as RM380", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			meal := createPrebuiltMeal(tx, "Budget Report Meal")
			completePreferences(tx, user.ID, 500)

			svc := newReportService(tx)

			loc := time.FixedZone("Asia/Singapore", 8*60*60)
			now := time.Now().In(loc)
			currentMonth := now.Format("2006-01")

			createMealLog(tx, user.ID, meal, "Budget Meal 1", 40,
				time.Date(now.Year(), now.Month(), 2, 12, 0, 0, 0, loc), 400, 20, 50, 10)

			createMealLog(tx, user.ID, meal, "Budget Meal 2", 80,
				time.Date(now.Year(), now.Month(), 3, 12, 0, 0, 0, loc), 500, 25, 60, 15)

			report, err := svc.GenerateMonth(ctx, user.ID, currentMonth)

			Expect(err).NotTo(HaveOccurred())
			Expect(report.Summary.TotalSpent).To(Equal(120.0))
			Expect(report.Summary.MonthlyMealBudget).NotTo(BeNil())
			Expect(*report.Summary.MonthlyMealBudget).To(Equal(500.0))
			Expect(report.Summary.RemainingUsableBudget).NotTo(BeNil())
			Expect(*report.Summary.RemainingUsableBudget).To(Equal(380.0))
		})
	})

	Describe("Calculate calorie and macronutrient summary", func() {
		It("should sum calories, protein, carbohydrate, and fat values", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			meal := createPrebuiltMeal(tx, "Macro Report Meal")
			svc := newReportService(tx)

			loc := time.FixedZone("Asia/Singapore", 8*60*60)

			createMealLog(tx, user.ID, meal, "Macro Meal 1", 10,
				time.Date(2026, 7, 3, 12, 0, 0, 0, loc), 400, 20, 50, 10)

			createMealLog(tx, user.ID, meal, "Macro Meal 2", 15,
				time.Date(2026, 7, 4, 12, 0, 0, 0, loc), 500, 25, 60, 15)

			createMealLog(tx, user.ID, meal, "Macro Meal 3", 20,
				time.Date(2026, 7, 5, 12, 0, 0, 0, loc), 600, 30, 70, 20)

			report, err := svc.GenerateMonth(ctx, user.ID, "2026-07")

			Expect(err).NotTo(HaveOccurred())
			Expect(report.Summary.TotalCalories).To(Equal(1500.0))
			Expect(report.Summary.AverageCaloriesPerMeal).To(Equal(500.0))

			Expect(report.MacroSummary.TotalProteinG).To(Equal(75.0))
			Expect(report.MacroSummary.TotalCarbsG).To(Equal(180.0))
			Expect(report.MacroSummary.TotalFatG).To(Equal(45.0))

			Expect(report.MacroSummary.AverageProteinG).To(Equal(25.0))
			Expect(report.MacroSummary.AverageCarbsG).To(Equal(60.0))
			Expect(report.MacroSummary.AverageFatG).To(Equal(15.0))
		})
	})

	Describe("Generate report when no meal logs exist", func() {
		It("should return an empty report with zero totals", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			svc := newReportService(tx)

			report, err := svc.GenerateMonth(ctx, user.ID, "2026-07")

			Expect(err).NotTo(HaveOccurred())
			Expect(report).NotTo(BeNil())
			Expect(report.Period.Kind).To(Equal("month"))
			Expect(report.Summary.TotalMeals).To(Equal(0))
			Expect(report.Summary.TotalSpent).To(Equal(0.0))
			Expect(report.Summary.TotalCalories).To(Equal(0.0))
			Expect(report.MacroSummary.TotalProteinG).To(Equal(0.0))
			Expect(report.MacroSummary.TotalCarbsG).To(Equal(0.0))
			Expect(report.MacroSummary.TotalFatG).To(Equal(0.0))
			Expect(report.CategoryBreakdown).To(BeEmpty())
			Expect(report.TimePatterns).To(BeEmpty())
			Expect(report.DailyTrends).To(BeEmpty())
			Expect(report.TopMeals).To(BeEmpty())
			Expect(report.TopExpensiveMeals).To(BeEmpty())
			Expect(report.Insights).To(BeEmpty())
		})
	})
})
