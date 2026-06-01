package preference

import (
	"context"
	"errors"
	"fmt"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type service struct {
	preferenceRepo interfaces.PreferenceRepository
	userRepo       interfaces.UserRepository //to update user's field: has_completed_onboarding
}

func NewService(preferenceRepo interfaces.PreferenceRepository, userRepo interfaces.UserRepository) interfaces.PreferenceService {
	return &service{
		preferenceRepo: preferenceRepo,
		userRepo:       userRepo,
	}
}

func (s *service) CompleteOnboarding(ctx context.Context, userID uuid.UUID, input interfaces.PreferenceInput) (*interfaces.PreferenceResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	if user.HasCompletedOnboarding {
		return nil, errors.New("user has already completed onboarding")
	}

	err = validatePreferenceInput(input)
	if err != nil {
		return nil, err
	}

	newPreference := buildPreferenceModel(input, userID)

	savedPreference, err := s.preferenceRepo.Upsert(ctx, newPreference)
	if err != nil {
		return nil, err
	}

	if err = s.userRepo.UpdateOnboardingStatus(ctx, userID, true); err != nil {
		return nil, err
	}

	return buildPreferenceResponse(savedPreference), nil
}

// Return current logged-in users' saved preferences
func (s *service) GetByUserID(ctx context.Context, userID uuid.UUID) (*interfaces.PreferenceResponse, error) {
	_, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	userPreference, err := s.preferenceRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return buildPreferenceResponse(userPreference), nil
}

func (s *service) Update(ctx context.Context, userID uuid.UUID, input interfaces.PreferenceInput) (*interfaces.PreferenceResponse, error) {
	_, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	err = validatePreferenceInput(input)
	if err != nil {
		return nil, err
	}

	newPreference := buildPreferenceModel(input, userID)

	savedPreference, err := s.preferenceRepo.Upsert(ctx, newPreference)
	if err != nil {
		return nil, err
	}

	return buildPreferenceResponse(savedPreference), nil

}

func (s *service) UpdateDataSharingConsent(ctx context.Context, userID uuid.UUID, consent bool) error {
	_, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	if err = s.preferenceRepo.UpdateDataSharingConsent(ctx, userID, consent); err != nil {
		return err
	}
	return nil

}

func validatePreferenceInput(input interfaces.PreferenceInput) error {
	if input.MonthlyMealBudget <= 0 {
		return errors.New("monthly meal budget must be greater than 0")
	}

	if strings.TrimSpace(input.HomeLocation) == "" {
		return errors.New("home location is required")
	}

	if strings.TrimSpace(input.WorkSchoolLocation) == "" {
		return errors.New("work/school location is required")
	}

	if err := validateMainGoal(input.MainGoal); err != nil {
		return err
	}
	if err := validateHealthConcerns(input.HealthConcerns); err != nil {
		return err
	}

	if err := validateDietaryRestrictions(input.DietaryRestrictions); err != nil {
		return err
	}

	if err := validateMealPreferences(input.PreferredMealTags); err != nil {
		return err
	}

	if err := validateMealPreferencesAgainstDietaryRestrictions(input.PreferredMealTags, input.DietaryRestrictions); err != nil {
		return err
	}

	return nil
}

var allowedMainGoals = map[string]bool{
	"eat_healthier":        true,
	"muscle_gain":          true,
	"quick_recommendation": true,
}

var allowedDietaryRestrictions = map[string]bool{
	"halal":        true,
	"non_beef":     true,
	"vegetarian":   true,
	"vegan":        true,
	"seafood_free": true,
	"nut_free":     true,
	"low_salt":     true,
	"low_sugar":    true,
	"low_fat":      true,
}

var allowedHealthConcerns = map[string]bool{
	"high_blood_pressure": true,
	"gout":                true,
	"diabetes":            true,
}

var allowedMealPreferenceTags = map[string]bool{
	"american":       true,
	"basics":         true,
	"breakfast":      true,
	"chinese":        true,
	"condiments":     true,
	"desserts":       true,
	"drinks":         true,
	"french":         true,
	"fruits":         true,
	"grains":         true,
	"greek":          true,
	"healthy":        true,
	"indian":         true,
	"italian":        true,
	"japanese":       true,
	"korean":         true,
	"kuih":           true,
	"legumes":        true,
	"meat":           true,
	"mexican":        true,
	"middle_eastern": true,
	"noodles":        true,
	"nuts":           true,
	"proteins":       true,
	"rice":           true,
	"roti":           true,
	"seafood":        true,
	"seeds":          true,
	"snacks":         true,
	"soups":          true,
	"spanish":        true,
	"thai":           true,
	"vegetables":     true,
	"vietnamese":     true,
	"western":        true,
}

var restrictedMealPreferenceTags = map[string]map[string]bool{
	"seafood_free": {"seafood": true},
	"nut_free":     {"nuts": true, "seeds": true},
	"vegetarian":   {"meat": true, "seafood": true},
	"vegan":        {"meat": true, "seafood": true},
}

func validateMainGoal(goal string) error {
	goal = strings.TrimSpace(goal)

	if goal == "" {
		return errors.New("main goal cannot be empty")
	}

	if !allowedMainGoals[goal] {
		return fmt.Errorf("unsupported main goal: %s", goal)
	}
	return nil
}

func validateHealthConcerns(concerns []string) error {
	for _, concern := range concerns {
		concern = strings.TrimSpace(concern)

		if concern == "" {
			return errors.New("health concern cannot be empty")
		}

		if !allowedHealthConcerns[concern] {
			return fmt.Errorf("unsupported health concern: %s", concern)
		}
	}
	return nil

}

func validateDietaryRestrictions(restrictions []string) error {
	for _, restriction := range restrictions {
		restriction = strings.TrimSpace(restriction)

		if restriction == "" {
			return errors.New("dietary restriction cannot be empty")
		}

		if !allowedDietaryRestrictions[restriction] {
			return fmt.Errorf("unsupported dietary restriction: %s", restriction)
		}
	}
	return nil
}

func validateMealPreferences(tags []string) error {
	if len(tags) == 0 {
		return errors.New("at least one meal preference tag is required")
	}

	for _, tag := range tags {
		tag = strings.TrimSpace(tag)

		if tag == "" {
			return errors.New("meal preference tag cannot be empty")
		}

		if !allowedMealPreferenceTags[tag] {
			return fmt.Errorf("unsupported meal preference tag: %s", tag)
		}
	}
	return nil
}

func validateMealPreferencesAgainstDietaryRestrictions(tags []string, restrictions []string) error {
	for _, restriction := range restrictions {
		for _, tag := range tags {
			if restrictedMealPreferenceTags[restriction][tag] {
				return fmt.Errorf("meal preference tag %s conflicts with dietary restriction %s", tag, restriction)
			}
		}
	}

	return nil
}

func buildPreferenceResponse(userPreference *model.UserPreference) *interfaces.PreferenceResponse {
	healthConcerns := make([]string, 0, len(userPreference.HealthConcerns))
	for _, concern := range userPreference.HealthConcerns {
		healthConcerns = append(healthConcerns, concern.Concern)
	}

	dietaryRestrictions := make([]string, 0, len(userPreference.DietaryRestrictions))
	for _, restriction := range userPreference.DietaryRestrictions {
		dietaryRestrictions = append(dietaryRestrictions, restriction.Restriction)
	}

	preferredMealTags := make([]string, 0, len(userPreference.MealPreferences))
	for _, mealPreference := range userPreference.MealPreferences {
		preferredMealTags = append(preferredMealTags, mealPreference.PreferenceTag)
	}

	// Convert DB model to reponse DTO
	return &interfaces.PreferenceResponse{
		MainGoal:            userPreference.MainGoal,
		MonthlyMealBudget:   userPreference.MonthlyMealBudget,
		DataSharingConsent:  userPreference.DataSharingConsent,
		HomeLocation:        userPreference.HomeLocation,
		WorkSchoolLocation:  userPreference.WorkSchoolLocation,
		HealthConcerns:      healthConcerns,
		DietaryRestrictions: dietaryRestrictions,
		PreferredMealTags:   preferredMealTags,
	}
}

func buildPreferenceModel(input interfaces.PreferenceInput, userID uuid.UUID) *model.UserPreference {
	healthConcerns := make([]model.UserHealthConcern, 0, len(input.HealthConcerns))
	for _, concern := range input.HealthConcerns { // from string to model.UserHealthConcern
		healthConcerns = append(healthConcerns, model.UserHealthConcern{
			UserID:  userID,
			Concern: concern,
		})
	}

	dietaryRestrictions := make([]model.UserDietaryRestriction, 0, len(input.DietaryRestrictions))
	for _, restriction := range input.DietaryRestrictions {
		dietaryRestrictions = append(dietaryRestrictions, model.UserDietaryRestriction{
			UserID:      userID,
			Restriction: restriction,
		})
	}

	mealPreferences := make([]model.UserMealPreference, 0, len(input.PreferredMealTags))

	for _, tag := range input.PreferredMealTags {
		mealPreferences = append(mealPreferences, model.UserMealPreference{
			UserID:        userID,
			PreferenceTag: tag,
		})
	}

	newPreference := &model.UserPreference{
		UserID:              userID,
		MainGoal:            input.MainGoal,
		MonthlyMealBudget:   input.MonthlyMealBudget,
		DataSharingConsent:  input.DataSharingConsent,
		HomeLocation:        input.HomeLocation,
		WorkSchoolLocation:  input.WorkSchoolLocation,
		HealthConcerns:      healthConcerns,
		DietaryRestrictions: dietaryRestrictions,
		MealPreferences:     mealPreferences,
	}

	return newPreference
}
