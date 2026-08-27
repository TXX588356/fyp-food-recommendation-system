//go:build integration

package integration_test

import (
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/repository/postgres"
	authservice "fyp/food-rs/internal/service/auth"
	preferenceservice "fyp/food-rs/internal/service/preference"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var _ = Describe("Preference integration", func() {
	const jwtSecret = "integration-test-secret"

	newPreferenceService := func(tx *gorm.DB) interfaces.PreferenceService {
		preferenceRepo := postgres.NewPreferencePostgresRepository(tx)
		userRepo := postgres.NewUserPostgresRepository(tx)

		return preferenceservice.NewService(preferenceRepo, userRepo)
	}

	createTestUser := func(tx *gorm.DB) model.User {
		ctx := GinkgoT().Context()

		userRepo := postgres.NewUserPostgresRepository(tx)
		refreshTokenRepo := postgres.NewRefreshTokenPostgresRepository(tx)
		authSvc := authservice.NewService(userRepo, refreshTokenRepo, jwtSecret)

		email := fmt.Sprintf("integration-%s@gmail.com", uuid.NewString())

		result, err := authSvc.Register(ctx, interfaces.RegisterInput{
			Name:     "Preference User",
			Email:    email,
			Password: "password123",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())

		var user model.User
		Expect(tx.Where("email = ?", email).First(&user).Error).
			NotTo(HaveOccurred())

		return user
	}

	findStoredPreference := func(tx *gorm.DB, userID uuid.UUID) model.UserPreference {
		var preference model.UserPreference

		Expect(tx.
			Preload("HealthConcerns").
			Preload("DietaryRestrictions").
			Preload("MealPreferences").
			Where("user_id = ?", userID).
			First(&preference).Error).
			NotTo(HaveOccurred())

		return preference
	}

	healthConcerns := func(preference model.UserPreference) []string {
		values := make([]string, 0, len(preference.HealthConcerns))
		for _, item := range preference.HealthConcerns {
			values = append(values, item.Concern)
		}
		return values
	}

	dietaryRestrictions := func(preference model.UserPreference) []string {
		values := make([]string, 0, len(preference.DietaryRestrictions))
		for _, item := range preference.DietaryRestrictions {
			values = append(values, item.Restriction)
		}
		return values
	}

	preferredMealTags := func(preference model.UserPreference) []string {
		values := make([]string, 0, len(preference.MealPreferences))
		for _, item := range preference.MealPreferences {
			values = append(values, item.PreferenceTag)
		}
		return values
	}

	Describe("Complete onboarding with valid preferences", func() {
		It("should store preferences and marks the user as onboarded", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			svc := newPreferenceService(tx)

			consent := true

			result, err := svc.CompleteOnboarding(ctx, user.ID, interfaces.PreferenceInput{
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
			Expect(result).NotTo(BeNil())
			Expect(result.MainGoal).To(Equal("eat_healthier"))
			Expect(result.MonthlyMealBudget).To(Equal(600.0))
			Expect(result.DataSharingConsent).NotTo(BeNil())
			Expect(*result.DataSharingConsent).To(BeTrue())

			var storedUser model.User
			Expect(tx.First(&storedUser, "id = ?", user.ID).Error).
				NotTo(HaveOccurred())
			Expect(storedUser.HasCompletedOnboarding).To(BeTrue())

			storedPreference := findStoredPreference(tx, user.ID)
			Expect(storedPreference.MainGoal).To(Equal("eat_healthier"))
			Expect(storedPreference.HomeLocation).To(Equal("Bukit Jalil"))
			Expect(storedPreference.WorkSchoolLocation).To(Equal("KL Sentral"))
			Expect(healthConcerns(storedPreference)).To(ConsistOf("high_blood_pressure"))
			Expect(dietaryRestrictions(storedPreference)).To(ConsistOf("halal"))
			Expect(preferredMealTags(storedPreference)).To(ConsistOf("malaysian", "rice_dishes"))
		})
	})

	Describe("Retrieve saved preferences", func() {
		It("should return stored preference data correctly", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			svc := newPreferenceService(tx)

			consent := true

			_, err := svc.CompleteOnboarding(ctx, user.ID, interfaces.PreferenceInput{
				MainGoal:            "muscle_gain",
				MonthlyMealBudget:   750,
				DataSharingConsent:  &consent,
				HomeLocation:        "Cheras",
				WorkSchoolLocation:  "Mid Valley",
				HealthConcerns:      []string{"diabetes", "gout"},
				DietaryRestrictions: []string{"non_beef"},
				PreferredMealTags:   []string{"chinese", "noodle_dishes"},
			})
			Expect(err).NotTo(HaveOccurred())

			result, err := svc.GetByUserID(ctx, user.ID)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).NotTo(BeNil())
			Expect(result.MainGoal).To(Equal("muscle_gain"))
			Expect(result.MonthlyMealBudget).To(Equal(750.0))
			Expect(result.HomeLocation).To(Equal("Cheras"))
			Expect(result.WorkSchoolLocation).To(Equal("Mid Valley"))
			Expect(result.HealthConcerns).To(ConsistOf("diabetes", "gout"))
			Expect(result.DietaryRestrictions).To(ConsistOf("non_beef"))
			Expect(result.PreferredMealTags).To(ConsistOf("chinese", "noodle_dishes"))
		})
	})

	Describe("Update existing preferences", func() {
		It("should persist updated preference values in PostgreSQL", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			svc := newPreferenceService(tx)

			consent := true

			_, err := svc.CompleteOnboarding(ctx, user.ID, interfaces.PreferenceInput{
				MainGoal:            "eat_healthier",
				MonthlyMealBudget:   600,
				DataSharingConsent:  &consent,
				HomeLocation:        "Bukit Jalil",
				WorkSchoolLocation:  "KL Sentral",
				HealthConcerns:      []string{"high_blood_pressure"},
				DietaryRestrictions: []string{"halal"},
				PreferredMealTags:   []string{"malaysian"},
			})
			Expect(err).NotTo(HaveOccurred())

			updatedConsent := false

			result, err := svc.Update(ctx, user.ID, interfaces.PreferenceInput{
				MainGoal:            "quick_recommendation",
				MonthlyMealBudget:   450,
				DataSharingConsent:  &updatedConsent,
				HomeLocation:        "Subang Jaya",
				WorkSchoolLocation:  "Bangsar",
				HealthConcerns:      []string{"diabetes"},
				DietaryRestrictions: []string{"low_sugar"},
				PreferredMealTags:   []string{"japanese", "soups"},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.MainGoal).To(Equal("quick_recommendation"))
			Expect(result.MonthlyMealBudget).To(Equal(450.0))
			Expect(*result.DataSharingConsent).To(BeFalse())

			storedPreference := findStoredPreference(tx, user.ID)
			Expect(storedPreference.MainGoal).To(Equal("quick_recommendation"))
			Expect(storedPreference.MonthlyMealBudget).To(Equal(450.0))
			Expect(storedPreference.HomeLocation).To(Equal("Subang Jaya"))
			Expect(storedPreference.WorkSchoolLocation).To(Equal("Bangsar"))
		})
	})

	Describe("Update dietary restrictions and health concerns", func() {
		It("should replace existing relation rows with the updated values", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			svc := newPreferenceService(tx)

			consent := true

			_, err := svc.CompleteOnboarding(ctx, user.ID, interfaces.PreferenceInput{
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

			_, err = svc.Update(ctx, user.ID, interfaces.PreferenceInput{
				MainGoal:            "eat_healthier",
				MonthlyMealBudget:   600,
				DataSharingConsent:  &consent,
				HomeLocation:        "Bukit Jalil",
				WorkSchoolLocation:  "KL Sentral",
				HealthConcerns:      []string{"diabetes", "gout"},
				DietaryRestrictions: []string{"vegetarian", "low_salt"},
				PreferredMealTags:   []string{"vegetables", "tofu_soy"},
			})
			Expect(err).NotTo(HaveOccurred())

			storedPreference := findStoredPreference(tx, user.ID)

			Expect(healthConcerns(storedPreference)).
				To(ConsistOf("diabetes", "gout"))
			Expect(dietaryRestrictions(storedPreference)).
				To(ConsistOf("vegetarian", "low_salt"))
			Expect(preferredMealTags(storedPreference)).
				To(ConsistOf("vegetables", "tofu_soy"))

			Expect(healthConcerns(storedPreference)).
				NotTo(ContainElement("high_blood_pressure"))
			Expect(dietaryRestrictions(storedPreference)).
				NotTo(ContainElement("halal"))
			Expect(preferredMealTags(storedPreference)).
				NotTo(ContainElement("malaysian"))
		})
	})

	Describe("Update monthly budget and locations", func() {
		It("should store the new budget and locations correctly", func() {
			tx, ctx := beginIntegrationTx()
			user := createTestUser(tx)
			svc := newPreferenceService(tx)

			consent := true

			_, err := svc.CompleteOnboarding(ctx, user.ID, interfaces.PreferenceInput{
				MainGoal:            "eat_healthier",
				MonthlyMealBudget:   600,
				DataSharingConsent:  &consent,
				HomeLocation:        "Bukit Jalil",
				WorkSchoolLocation:  "KL Sentral",
				HealthConcerns:      []string{"high_blood_pressure"},
				DietaryRestrictions: []string{"halal"},
				PreferredMealTags:   []string{"malaysian"},
			})
			Expect(err).NotTo(HaveOccurred())

			_, err = svc.Update(ctx, user.ID, interfaces.PreferenceInput{
				MainGoal:            "eat_healthier",
				MonthlyMealBudget:   900,
				DataSharingConsent:  &consent,
				HomeLocation:        "Petaling Jaya",
				WorkSchoolLocation:  "Cyberjaya",
				HealthConcerns:      []string{"high_blood_pressure"},
				DietaryRestrictions: []string{"halal"},
				PreferredMealTags:   []string{"malaysian"},
			})
			Expect(err).NotTo(HaveOccurred())

			storedPreference := findStoredPreference(tx, user.ID)

			Expect(storedPreference.MonthlyMealBudget).To(Equal(900.0))
			Expect(storedPreference.HomeLocation).To(Equal("Petaling Jaya"))
			Expect(storedPreference.WorkSchoolLocation).To(Equal("Cyberjaya"))
		})
	})
})
