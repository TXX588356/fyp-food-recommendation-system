//go:build integration

package integration_test

import (
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	catalogservice "fyp/food-rs/internal/service/catalog"
	custommealservice "fyp/food-rs/internal/service/custommeal"
	meallogservice "fyp/food-rs/internal/service/meallog"
	preferenceservice "fyp/food-rs/internal/service/preference"
	"fyp/food-rs/types/model"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

type integrationObjectURLResolver struct{}

func (integrationObjectURLResolver) Resolve(objectKey string) string {
	return objectKey
}

var _ = Describe("Meal log integration", func() {
	newMealLogService := func(tx *gorm.DB) interfaces.MealLogService {
		customMealRepo := postgres.NewCustomMealPostgresRepository(tx)
		catalogRepo := postgres.NewCatalogPostgresRepository(tx)
		mealLogRepo := postgres.NewMealLogPostgresRepository(tx)
		preferenceRepo := postgres.NewPreferencePostgresRepository(tx)
		userRepo := postgres.NewUserPostgresRepository(tx)

		customMealSvc := custommealservice.NewService(customMealRepo)
		catalogSvc := catalogservice.NewService(catalogRepo, integrationObjectURLResolver{})
		preferenceSvc := preferenceservice.NewService(preferenceRepo, userRepo)

		return meallogservice.NewService(mealLogRepo, customMealSvc, catalogSvc, preferenceSvc)
	}

	floatPtr := func(value float64) *float64 {
		return &value
	}

	createTestUser := func(tx *gorm.DB) model.User {
		user := model.User{
			Name:         "Meal Log User",
			Email:        fmt.Sprintf("integration-%s@example.com", uuid.NewString()),
			PasswordHash: "hashed-password",
		}

		Expect(tx.Create(&user).Error).NotTo(HaveOccurred())
		Expect(user.ID).NotTo(BeZero())

		return user
	}

	createPrebuiltMeal := func(tx *gorm.DB) model.PrebuiltMeal {
		meal := model.PrebuiltMeal{
			SourceCode:         "integration",
			SourceRecordID:     uuid.NewString(),
			Name:               "Integration Chicken Rice",
			CategoryCodes:      model.StringArray{"malaysian", "rice_dishes", "poultry"},
			ServingDescription: "1 plate",
			Calories:           floatPtr(620),
			ProteinG:           floatPtr(32),
			CarbsG:             floatPtr(72),
			FatG:               floatPtr(18),
		}

		Expect(tx.Create(&meal).Error).NotTo(HaveOccurred())
		Expect(meal.ID).NotTo(BeZero())

		return meal
	}

	createCustomMeal := func(tx *gorm.DB, userID uuid.UUID) *interfaces.CustomMealResponse {
		ctx := GinkgoT().Context()
		customMealRepo := postgres.NewCustomMealPostgresRepository(tx)
		customMealSvc := custommealservice.NewService(customMealRepo)

		meal, err := customMealSvc.Create(ctx, userID, interfaces.CustomMealInput{
			Name:                   "Integration Protein Bowl",
			Price:                  14.5,
			Calories:               540,
			FatG:                   12,
			ProteinG:               38,
			CarbsG:                 58,
			State:                  "Kuala Lumpur",
			District:               "Brickfields",
			RestaurantName:         "Integration Cafe",
			DietaryRestrictionTags: []string{"halal"},
			MealCategoryTags:       []string{"western", "poultry"},
			ImageURL:               "https://example.test/protein-bowl.jpg",
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(meal).NotTo(BeNil())

		return meal
	}

	parseLogID := func(response *interfaces.MealLogResponse) uuid.UUID {
		id, err := uuid.Parse(response.ID)
		Expect(err).NotTo(HaveOccurred())
		return id
	}

	countActiveMealLogs := func(tx *gorm.DB, logID uuid.UUID) int64 {
		var count int64
		Expect(tx.Model(&model.MealLog{}).
			Where("id = ? AND deleted_at IS NULL", logID).
			Count(&count).Error).
			NotTo(HaveOccurred())

		return count
	}

	Describe("Create meal log using prebuilt meal", func() {
		It("should retrieve prebuilt meal details and stores the log correctly", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			prebuiltMeal := createPrebuiltMeal(tx)
			svc := newMealLogService(tx)

			eatenAt := time.Date(2025, 7, 10, 12, 30, 0, 0, time.UTC)

			result, err := svc.Create(ctx, user.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMeal.ID.String(),
				Price:    11.5,
				EatenAt:  eatenAt,
				MealType: "lunch",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.PrebuiltMealID).NotTo(BeNil())
			Expect(*result.PrebuiltMealID).To(Equal(prebuiltMeal.ID.String()))
			Expect(result.CustomMealItemID).To(BeNil())
			Expect(result.MealName).To(Equal("Integration Chicken Rice"))
			Expect(result.Price).To(Equal(11.5))
			Expect(result.EatenAt).To(BeTemporally("==", eatenAt))
			Expect(result.MealType).To(Equal("lunch"))
			Expect(result.Calories).To(Equal(620.0))
			Expect(result.ProteinG).To(Equal(32.0))
			Expect(result.CarbsG).To(Equal(72.0))
			Expect(result.FatG).To(Equal(18.0))
			Expect(result.MealCategory).To(ConsistOf("malaysian", "rice_dishes", "poultry"))
		})
	})

	Describe("Create meal log using custom meal", func() {
		It("should retrieve custom meal details and stores the log correctly", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			customMeal := createCustomMeal(tx, user.ID)
			svc := newMealLogService(tx)

			eatenAt := time.Date(2025, 7, 11, 19, 0, 0, 0, time.UTC)

			result, err := svc.Create(ctx, user.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourceCustom,
				MealID:   customMeal.ID,
				Price:    14.5,
				EatenAt:  eatenAt,
				MealType: "dinner",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.CustomMealItemID).NotTo(BeNil())
			Expect(*result.CustomMealItemID).To(Equal(customMeal.ID))
			Expect(result.PrebuiltMealID).To(BeNil())
			Expect(result.MealName).To(Equal("Integration Protein Bowl"))
			Expect(result.Price).To(Equal(14.5))
			Expect(result.MealType).To(Equal("dinner"))
			Expect(result.Calories).To(Equal(540.0))
			Expect(result.ProteinG).To(Equal(38.0))
			Expect(result.CarbsG).To(Equal(58.0))
			Expect(result.FatG).To(Equal(12.0))
			Expect(result.MealCategory).To(ConsistOf("western", "poultry"))
		})
	})

	Describe("Retrieve meal logs for selected month", func() {
		It("should return only logs within the selected month", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			prebuiltMeal := createPrebuiltMeal(tx)
			svc := newMealLogService(tx)

			julyFirst := time.Date(2025, 7, 3, 8, 0, 0, 0, time.UTC)
			julySecond := time.Date(2025, 7, 20, 13, 0, 0, 0, time.UTC)
			august := time.Date(2025, 8, 1, 12, 0, 0, 0, time.UTC)

			_, err := svc.Create(ctx, user.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMeal.ID.String(),
				Price:    10,
				EatenAt:  julyFirst,
				MealType: "breakfast",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = svc.Create(ctx, user.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMeal.ID.String(),
				Price:    12,
				EatenAt:  julySecond,
				MealType: "lunch",
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = svc.Create(ctx, user.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMeal.ID.String(),
				Price:    15,
				EatenAt:  august,
				MealType: "lunch",
			})
			Expect(err).NotTo(HaveOccurred())

			result, err := svc.GetMonth(ctx, user.ID, "2025-07")

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.Summary.Month).To(Equal("2025-07"))
			Expect(result.Summary.TotalMealsEaten).To(Equal(2))
			Expect(result.Summary.TotalSpent).To(Equal(22.0))
			Expect(result.Items).To(HaveLen(2))
			Expect(result.Items[0].EatenAt.Month()).To(Equal(time.July))
			Expect(result.Items[1].EatenAt.Month()).To(Equal(time.July))
		})
	})

	Describe("Update owned meal log", func() {
		It("should update the meal log successfully", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			prebuiltMeal := createPrebuiltMeal(tx)
			svc := newMealLogService(tx)

			created, err := svc.Create(ctx, user.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMeal.ID.String(),
				Price:    11.5,
				EatenAt:  time.Date(2025, 7, 10, 12, 30, 0, 0, time.UTC),
				MealType: "lunch",
			})
			Expect(err).NotTo(HaveOccurred())

			updatedAt := time.Date(2025, 7, 10, 20, 0, 0, 0, time.UTC)

			updated, err := svc.Update(ctx, user.ID, parseLogID(created), interfaces.MealLogUpdateInput{
				Price:    16.25,
				EatenAt:  updatedAt,
				MealType: "dinner",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(updated).NotTo(BeNil())
			Expect(updated.Price).To(Equal(16.25))
			Expect(updated.EatenAt).To(BeTemporally("==", updatedAt))
			Expect(updated.MealType).To(Equal("dinner"))
			Expect(updated.MealName).To(Equal("Integration Chicken Rice"))
		})
	})

	Describe("Reject update of another user's meal log", func() {
		It("should deny the update and leaves the log unchanged", func() {
			tx, ctx := beginIntegrationTx()
			owner := createTestUser(tx)
			otherUser := createTestUser(tx)
			prebuiltMeal := createPrebuiltMeal(tx)
			svc := newMealLogService(tx)

			created, err := svc.Create(ctx, owner.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMeal.ID.String(),
				Price:    11.5,
				EatenAt:  time.Date(2025, 7, 10, 12, 30, 0, 0, time.UTC),
				MealType: "lunch",
			})
			Expect(err).NotTo(HaveOccurred())

			logID := parseLogID(created)

			updated, err := svc.Update(ctx, otherUser.ID, logID, interfaces.MealLogUpdateInput{
				Price:    99,
				EatenAt:  time.Date(2025, 7, 10, 20, 0, 0, 0, time.UTC),
				MealType: "dinner",
			})

			Expect(updated).To(BeNil())
			Expect(err).To(HaveOccurred())

			var stored model.MealLog
			Expect(tx.First(&stored, "id = ?", logID).Error).NotTo(HaveOccurred())
			Expect(stored.UserID).To(Equal(owner.ID))
			Expect(stored.Price).To(Equal(11.5))
			Expect(stored.MealType).To(Equal("lunch"))
		})
	})

	Describe("Delete owned meal log", func() {
		It("should remove the meal log successfully", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			prebuiltMeal := createPrebuiltMeal(tx)
			svc := newMealLogService(tx)

			created, err := svc.Create(ctx, user.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMeal.ID.String(),
				Price:    11.5,
				EatenAt:  time.Date(2025, 7, 10, 12, 30, 0, 0, time.UTC),
				MealType: "lunch",
			})
			Expect(err).NotTo(HaveOccurred())

			logID := parseLogID(created)

			err = svc.Delete(ctx, user.ID, logID)

			Expect(err).NotTo(HaveOccurred())
			Expect(countActiveMealLogs(tx, logID)).To(Equal(int64(0)))
		})
	})

	Describe("Reject delete of another user's meal log", func() {
		It("should deny the delete and keeps the log active", func() {
			tx, ctx := beginIntegrationTx()
			owner := createTestUser(tx)
			otherUser := createTestUser(tx)
			prebuiltMeal := createPrebuiltMeal(tx)
			svc := newMealLogService(tx)

			created, err := svc.Create(ctx, owner.ID, interfaces.MealLogInput{
				Source:   interfaces.MealLogSourcePrebuilt,
				MealID:   prebuiltMeal.ID.String(),
				Price:    11.5,
				EatenAt:  time.Date(2025, 7, 10, 12, 30, 0, 0, time.UTC),
				MealType: "lunch",
			})
			Expect(err).NotTo(HaveOccurred())

			logID := parseLogID(created)

			err = svc.Delete(ctx, otherUser.ID, logID)

			Expect(err).To(HaveOccurred())
			Expect(countActiveMealLogs(tx, logID)).To(Equal(int64(1)))
		})
	})
})
