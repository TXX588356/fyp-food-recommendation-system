//go:build integration

package integration_test

import (
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	custommealservice "fyp/food-rs/internal/service/custommeal"
	preferenceservice "fyp/food-rs/internal/service/preference"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var _ = Describe("Custom meal integration", func() {
	newCustomMealService := func(tx *gorm.DB) interfaces.CustomMealService {
		customMealRepo := postgres.NewCustomMealPostgresRepository(tx)
		return custommealservice.NewService(customMealRepo)
	}

	newPreferenceService := func(tx *gorm.DB) interfaces.PreferenceService {
		preferenceRepo := postgres.NewPreferencePostgresRepository(tx)
		userRepo := postgres.NewUserPostgresRepository(tx)

		return preferenceservice.NewService(preferenceRepo, userRepo)
	}

	createTestUser := func(tx *gorm.DB, name string) model.User {
		user := model.User{
			Name:         name,
			Email:        fmt.Sprintf("integration-%s@example.com", uuid.NewString()),
			PasswordHash: "hashed-password",
		}

		Expect(tx.Create(&user).Error).NotTo(HaveOccurred())
		Expect(user.ID).NotTo(BeZero())

		return user
	}

	completePreferences := func(tx *gorm.DB, userID uuid.UUID, consent bool) {
		ctx := GinkgoT().Context()
		preferenceSvc := newPreferenceService(tx)

		_, err := preferenceSvc.CompleteOnboarding(ctx, userID, interfaces.PreferenceInput{
			MainGoal:            "eat_healthier",
			MonthlyMealBudget:   600,
			DataSharingConsent:  &consent,
			HomeLocation:        "Bukit Jalil",
			WorkSchoolLocation:  "KL Sentral",
			HealthConcerns:      []string{"high_blood_pressure"},
			DietaryRestrictions: []string{"halal"},
			PreferredMealTags:   []string{"malaysian", "rice_dishes"},
		})
		Expect(err).NotTo(HaveOccurred())
	}

	validCustomMealInput := func(name string) interfaces.CustomMealInput {
		return interfaces.CustomMealInput{
			Name:                   name,
			Price:                  12.5,
			Calories:               520,
			FatG:                   14,
			ProteinG:               31,
			CarbsG:                 62,
			State:                  "Kuala Lumpur",
			District:               "Brickfields",
			RestaurantName:         "Integration Cafe",
			DietaryRestrictionTags: []string{"halal"},
			MealCategoryTags:       []string{"malaysian", "rice_dishes"},
			ImageURL:               "https://example.test/custom-meal.jpg",
		}
	}

	findStoredCustomMeal := func(tx *gorm.DB, mealID string) model.CustomMealItem {
		id, err := uuid.Parse(mealID)
		Expect(err).NotTo(HaveOccurred())

		var meal model.CustomMealItem
		Expect(tx.
			Preload("DietaryRestrictionTags").
			Preload("MealCategoryTags").
			Where("id = ?", id).
			First(&meal).Error).
			NotTo(HaveOccurred())

		return meal
	}

	Describe("Create custom meal when data sharing is enabled", func() {
		It("should store the custom meal and makes it visible as shared to another user", func() {
			tx, ctx := beginIntegrationTx()
			owner := createTestUser(tx, "Sharing Enabled Owner")
			viewer := createTestUser(tx, "Shared Meal Viewer")
			completePreferences(tx, owner.ID, true)

			customMealSvc := newCustomMealService(tx)

			created, err := customMealSvc.Create(ctx, owner.ID, validCustomMealInput("Shared Nasi Lemak"))

			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Name).To(Equal("Shared Nasi Lemak"))
			Expect(created.IsOwner).To(BeTrue())
			Expect(created.IsShared).To(BeFalse())

			visibleToViewer, err := customMealSvc.FindVisibleByID(ctx, viewer.ID, uuid.MustParse(created.ID))

			Expect(err).NotTo(HaveOccurred())
			Expect(visibleToViewer).NotTo(BeNil())
			Expect(visibleToViewer.Name).To(Equal("Shared Nasi Lemak"))
			Expect(visibleToViewer.IsOwner).To(BeFalse())
			Expect(visibleToViewer.IsShared).To(BeTrue())
		})
	})

	Describe("Create custom meal when data sharing is disabled", func() {
		It("should store the custom meal but keeps it private from another user", func() {
			tx, ctx := beginIntegrationTx()
			owner := createTestUser(tx, "Sharing Disabled Owner")
			viewer := createTestUser(tx, "Private Meal Viewer")
			completePreferences(tx, owner.ID, false)

			customMealSvc := newCustomMealService(tx)

			created, err := customMealSvc.Create(ctx, owner.ID, validCustomMealInput("Private Chicken Rice"))

			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.Name).To(Equal("Private Chicken Rice"))
			Expect(created.IsOwner).To(BeTrue())
			Expect(created.IsShared).To(BeFalse())

			visibleToViewer, err := customMealSvc.FindVisibleByID(ctx, viewer.ID, uuid.MustParse(created.ID))

			Expect(visibleToViewer).To(BeNil())
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Retrieve user's own custom meal", func() {
		It("should return the authenticated user's custom meal correctly", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx, "Custom Meal Owner")
			completePreferences(tx, user.ID, false)

			customMealSvc := newCustomMealService(tx)

			created, err := customMealSvc.Create(ctx, user.ID, validCustomMealInput("Owner Protein Bowl"))
			Expect(err).NotTo(HaveOccurred())

			found, err := customMealSvc.FindVisibleByID(ctx, user.ID, uuid.MustParse(created.ID))

			Expect(err).NotTo(HaveOccurred())
			Expect(found).NotTo(BeNil())
			Expect(found.ID).To(Equal(created.ID))
			Expect(found.Name).To(Equal("Owner Protein Bowl"))
			Expect(found.Price).To(Equal(12.5))
			Expect(found.Calories).To(Equal(520.0))
			Expect(found.ProteinG).To(Equal(31.0))
			Expect(found.CarbsG).To(Equal(62.0))
			Expect(found.FatG).To(Equal(14.0))
			Expect(found.State).To(Equal("Kuala Lumpur"))
			Expect(found.District).To(Equal("Brickfields"))
			Expect(found.RestaurantName).To(Equal("Integration Cafe"))
			Expect(found.DietaryRestrictionTags).To(ConsistOf("halal"))
			Expect(found.MealCategoryTags).To(ConsistOf("malaysian", "rice_dishes"))
			Expect(found.IsOwner).To(BeTrue())
			Expect(found.IsShared).To(BeFalse())
		})
	})

	Describe("Search visible custom meals", func() {
		It("should return own meals and shared meals, but excludes private meals from other users", func() {
			tx, ctx := beginIntegrationTx()

			viewer := createTestUser(tx, "Visible Meal Viewer")
			sharedOwner := createTestUser(tx, "Shared Owner")
			privateOwner := createTestUser(tx, "Private Owner")

			completePreferences(tx, viewer.ID, false)
			completePreferences(tx, sharedOwner.ID, true)
			completePreferences(tx, privateOwner.ID, false)

			customMealSvc := newCustomMealService(tx)

			ownMeal, err := customMealSvc.Create(ctx, viewer.ID, validCustomMealInput("Search Own Meal"))
			Expect(err).NotTo(HaveOccurred())

			sharedMeal, err := customMealSvc.Create(ctx, sharedOwner.ID, validCustomMealInput("Search Shared Meal"))
			Expect(err).NotTo(HaveOccurred())

			privateMeal, err := customMealSvc.Create(ctx, privateOwner.ID, validCustomMealInput("Search Private Meal"))
			Expect(err).NotTo(HaveOccurred())

			results, err := customMealSvc.ListVisible(ctx, viewer.ID, "Search")

			Expect(err).NotTo(HaveOccurred())

			ids := make([]string, 0, len(results))
			names := make([]string, 0, len(results))
			sharedFlags := map[string]bool{}
			ownerFlags := map[string]bool{}

			for _, meal := range results {
				ids = append(ids, meal.ID)
				names = append(names, meal.Name)
				sharedFlags[meal.ID] = meal.IsShared
				ownerFlags[meal.ID] = meal.IsOwner
			}

			Expect(ids).To(ContainElement(ownMeal.ID))
			Expect(ids).To(ContainElement(sharedMeal.ID))
			Expect(ids).NotTo(ContainElement(privateMeal.ID))

			Expect(names).To(ContainElement("Search Own Meal"))
			Expect(names).To(ContainElement("Search Shared Meal"))
			Expect(names).NotTo(ContainElement("Search Private Meal"))

			Expect(ownerFlags[ownMeal.ID]).To(BeTrue())
			Expect(sharedFlags[ownMeal.ID]).To(BeFalse())

			Expect(ownerFlags[sharedMeal.ID]).To(BeFalse())
			Expect(sharedFlags[sharedMeal.ID]).To(BeTrue())
		})
	})

	Describe("Create custom meal with location", func() {
		It("should store meal and related location information correctly", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx, "Location Meal Owner")
			completePreferences(tx, user.ID, true)

			customMealSvc := newCustomMealService(tx)

			input := validCustomMealInput("Location Curry Rice")
			input.State = "Selangor"
			input.District = "Petaling Jaya"
			input.RestaurantName = "PJ Integration Restaurant"

			created, err := customMealSvc.Create(ctx, user.ID, input)

			Expect(err).NotTo(HaveOccurred())
			Expect(created).NotTo(BeNil())
			Expect(created.State).To(Equal("Selangor"))
			Expect(created.District).To(Equal("Petaling Jaya"))
			Expect(created.RestaurantName).To(Equal("PJ Integration Restaurant"))

			stored := findStoredCustomMeal(tx, created.ID)

			Expect(stored.Name).To(Equal("Location Curry Rice"))
			Expect(stored.State).To(Equal("Selangor"))
			Expect(stored.District).To(Equal("Petaling Jaya"))
			Expect(stored.RestaurantName).To(Equal("PJ Integration Restaurant"))
			Expect(stored.CreatedBy).To(Equal(user.ID))
			Expect(stored.DietaryRestrictionTags).To(HaveLen(1))
			Expect(stored.DietaryRestrictionTags[0].DietaryRestrictionTag).To(Equal("halal"))
			Expect(stored.MealCategoryTags).To(HaveLen(2))
		})
	})
})
