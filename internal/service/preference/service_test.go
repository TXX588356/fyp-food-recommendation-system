package preference

import (
	"context"
	"errors"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/internal/mocks"
	"fyp/food-rs/types/model"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestPreferenceService(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Preference Service Suite")
}

var _ = Describe("meal preference validation", func() {

	var (
		mockUserRepo       *mocks.UserRepository
		mockPreferenceRepo *mocks.PreferenceRepository
		svc                interfaces.PreferenceService
	)

	BeforeEach(func() {
		mockUserRepo = mocks.NewUserRepository(GinkgoT())
		mockPreferenceRepo = mocks.NewPreferenceRepository(GinkgoT())

		svc = NewService(mockPreferenceRepo, mockUserRepo)
	})

	validPreferenceInput := func() interfaces.PreferenceInput {
		return interfaces.PreferenceInput{
			MainGoal:           "eat_healthier",
			MonthlyMealBudget:  1000,
			HomeLocation:       "Bangsar",
			WorkSchoolLocation: "Petaling Jaya",
			PreferredMealTags: []string{
				"malaysian",
			},
		}
	}

	It("should check dietary restrictions using current category codes", func() {
		err := validateMealPreferencesAgainstDietaryRestrictions([]string{"nuts_seeds"}, []string{"nut_free"})

		Expect(err).To(HaveOccurred())
	})

	It("should reject invalid monthly meal budget", func() {
		preferenceInput := interfaces.PreferenceInput{
			MainGoal:          "eat_healthier",
			MonthlyMealBudget: 0,
		}

		err := validatePreferenceInput(preferenceInput)

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("monthly meal budget must be greater than 0"))
	})

	It("should reject invalid main goal", func() {
		ctx := context.Background()
		userID := uuid.New()

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{
				ID:                     userID,
				HasCompletedOnboarding: false,
			}, nil)

		preferenceInput := interfaces.PreferenceInput{
			MainGoal:           "lose_weight",
			MonthlyMealBudget:  1000,
			HomeLocation:       "random places",
			WorkSchoolLocation: "Petaling Jaya",
			PreferredMealTags: []string{
				"malaysian",
			},
		}

		result, err := svc.CompleteOnboarding(ctx, userID, preferenceInput)

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("unsupported main goal: lose_weight"))
		Expect(result).To(BeNil())
	})

	It("should reject invalid health concern", func() {
		preferenceInput := interfaces.PreferenceInput{
			MainGoal:           "eat_healthier",
			MonthlyMealBudget:  1000,
			HealthConcerns:     []string{"idk"},
			HomeLocation:       "Bangsar",
			WorkSchoolLocation: "Petaling Jaya",
			PreferredMealTags: []string{
				"malaysian",
			},
		}

		err := validatePreferenceInput(preferenceInput)

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("unsupported health concern: idk"))
	})

	It("should reject invalid dietary restriction", func() {
		preferenceInput := interfaces.PreferenceInput{
			MainGoal:            "eat_healthier",
			MonthlyMealBudget:   1000,
			HealthConcerns:      []string{},
			HomeLocation:        "Bangsar",
			WorkSchoolLocation:  "Petaling Jaya",
			DietaryRestrictions: []string{"idk"},
			PreferredMealTags: []string{
				"malaysian",
			},
		}

		err := validatePreferenceInput(preferenceInput)

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("unsupported dietary restriction: idk"))
	})

	It("should reject invalid dietary restriction", func() {
		preferenceInput := interfaces.PreferenceInput{
			MainGoal:            "eat_healthier",
			MonthlyMealBudget:   1000,
			HealthConcerns:      []string{},
			HomeLocation:        "Bangsar",
			WorkSchoolLocation:  "Petaling Jaya",
			DietaryRestrictions: []string{"halal"},
			PreferredMealTags: []string{
				"idk",
			},
		}

		err := validatePreferenceInput(preferenceInput)

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError("unsupported meal preference tag: idk"))
	})

	It("should complete onboarding with valid preferences", func() {
		ctx := context.Background()
		userID := uuid.New()
		preferenceInput := validPreferenceInput()

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{
				ID:                     userID,
				HasCompletedOnboarding: false,
			}, nil).
			Once()

		mockPreferenceRepo.EXPECT().
			Upsert(ctx, mock.MatchedBy(func(preference *model.UserPreference) bool {
				return preference.UserID == userID &&
					preference.MainGoal == preferenceInput.MainGoal &&
					preference.MonthlyMealBudget == preferenceInput.MonthlyMealBudget &&
					preference.HomeLocation == preferenceInput.HomeLocation &&
					preference.WorkSchoolLocation == preferenceInput.WorkSchoolLocation &&
					len(preference.MealPreferences) == 1 &&
					preference.MealPreferences[0].PreferenceTag == "malaysian"
			})).
			RunAndReturn(func(_ context.Context, preference *model.UserPreference) (*model.UserPreference, error) {
				return preference, nil
			}).
			Once()

		mockUserRepo.EXPECT().
			UpdateOnboardingStatus(ctx, userID, true).
			Return(nil).
			Once()

		result, err := svc.CompleteOnboarding(ctx, userID, preferenceInput)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.MainGoal).To(Equal(preferenceInput.MainGoal))
		Expect(result.MonthlyMealBudget).To(Equal(preferenceInput.MonthlyMealBudget))
		Expect(result.PreferredMealTags).To(Equal([]string{"malaysian"}))
	})

	It("should return user not found when completing onboarding for a missing user", func() {
		ctx := context.Background()
		userID := uuid.New()

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(nil, gorm.ErrRecordNotFound).
			Once()

		result, err := svc.CompleteOnboarding(ctx, userID, validPreferenceInput())

		Expect(err).To(MatchError("user not found"))
		Expect(result).To(BeNil())
	})

	It("should reject onboarding when user has already completed onboarding", func() {
		ctx := context.Background()
		userID := uuid.New()

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{
				ID:                     userID,
				HasCompletedOnboarding: true,
			}, nil).
			Once()

		result, err := svc.CompleteOnboarding(ctx, userID, validPreferenceInput())

		Expect(err).To(MatchError("user has already completed onboarding"))
		Expect(result).To(BeNil())
	})

	It("should return error when saving onboarding preferences fails", func() {
		ctx := context.Background()
		userID := uuid.New()
		saveErr := errors.New("save failed")

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{
				ID:                     userID,
				HasCompletedOnboarding: false,
			}, nil).
			Once()

		mockPreferenceRepo.EXPECT().
			Upsert(ctx, mock.AnythingOfType("*model.UserPreference")).
			Return(nil, saveErr).
			Once()

		result, err := svc.CompleteOnboarding(ctx, userID, validPreferenceInput())

		Expect(err).To(MatchError(saveErr))
		Expect(result).To(BeNil())
	})

	It("should return error when updating onboarding status fails", func() {
		ctx := context.Background()
		userID := uuid.New()
		statusErr := errors.New("status update failed")

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{
				ID:                     userID,
				HasCompletedOnboarding: false,
			}, nil).
			Once()

		mockPreferenceRepo.EXPECT().
			Upsert(ctx, mock.AnythingOfType("*model.UserPreference")).
			RunAndReturn(func(_ context.Context, preference *model.UserPreference) (*model.UserPreference, error) {
				return preference, nil
			}).
			Once()

		mockUserRepo.EXPECT().
			UpdateOnboardingStatus(ctx, userID, true).
			Return(statusErr).
			Once()

		result, err := svc.CompleteOnboarding(ctx, userID, validPreferenceInput())

		Expect(err).To(MatchError(statusErr))
		Expect(result).To(BeNil())
	})

	It("should get preferences by user ID", func() {
		ctx := context.Background()
		userID := uuid.New()

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{ID: userID}, nil).
			Once()

		mockPreferenceRepo.EXPECT().
			FindByUserID(ctx, userID).
			Return(&model.UserPreference{
				UserID:             userID,
				MainGoal:           "eat_healthier",
				MonthlyMealBudget:  1000,
				HomeLocation:       "Bangsar",
				WorkSchoolLocation: "Petaling Jaya",
				HealthConcerns: []model.UserHealthConcern{
					{UserID: userID, Concern: "diabetes"},
				},
				DietaryRestrictions: []model.UserDietaryRestriction{
					{UserID: userID, Restriction: "halal"},
				},
				MealPreferences: []model.UserMealPreference{
					{UserID: userID, PreferenceTag: "malaysian"},
				},
			}, nil).
			Once()

		result, err := svc.GetByUserID(ctx, userID)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.HealthConcerns).To(Equal([]string{"diabetes"}))
		Expect(result.DietaryRestrictions).To(Equal([]string{"halal"}))
		Expect(result.PreferredMealTags).To(Equal([]string{"malaysian"}))
	})

	It("should update preferences with valid input", func() {
		ctx := context.Background()
		userID := uuid.New()
		preferenceInput := validPreferenceInput()

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{ID: userID}, nil).
			Once()

		mockPreferenceRepo.EXPECT().
			Upsert(ctx, mock.AnythingOfType("*model.UserPreference")).
			RunAndReturn(func(_ context.Context, preference *model.UserPreference) (*model.UserPreference, error) {
				Expect(preference.UserID).To(Equal(userID))
				Expect(preference.MainGoal).To(Equal(preferenceInput.MainGoal))
				return preference, nil
			}).
			Once()

		result, err := svc.Update(ctx, userID, preferenceInput)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).NotTo(BeNil())
		Expect(result.MainGoal).To(Equal(preferenceInput.MainGoal))
	})

	It("should update data sharing consent", func() {
		ctx := context.Background()
		userID := uuid.New()

		mockUserRepo.EXPECT().
			FindByID(ctx, userID).
			Return(&model.User{ID: userID}, nil).
			Once()

		mockPreferenceRepo.EXPECT().
			UpdateDataSharingConsent(ctx, userID, true).
			Return(nil).
			Once()

		err := svc.UpdateDataSharingConsent(ctx, userID, true)

		Expect(err).NotTo(HaveOccurred())
	})
})
