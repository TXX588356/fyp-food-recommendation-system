package interfaces

import (
	"context"

	"github.com/google/uuid"
)

type PreferenceInput struct {
	MainGoal            string   `json:"mainGoal"`
	MonthlyMealBudget   float64  `json:"monthlyMealBudget"`
	DataSharingConsent  *bool    `json:"dataSharingConsent"`
	HomeLocation        string   `json:"homeLocation"`
	WorkSchoolLocation  string   `json:"workSchoolLocation"`
	HealthConcerns      []string `json:"healthConcerns"`
	DietaryRestrictions []string `json:"dietaryRestrictions"`
	PreferredMealTags   []string `json:"preferredMealTags"`
}

type PreferenceResponse struct {
	MainGoal            string   `json:"mainGoal"`
	MonthlyMealBudget   float64  `json:"monthlyMealBudget"`
	DataSharingConsent  *bool    `json:"dataSharingConsent"`
	HomeLocation        string   `json:"homeLocation"`
	WorkSchoolLocation  string   `json:"workSchoolLocation"`
	HealthConcerns      []string `json:"healthConcerns"`
	DietaryRestrictions []string `json:"dietaryRestrictions"`
	PreferredMealTags   []string `json:"preferredMealTags"`
}

type PreferenceService interface {
	CompleteOnboarding(ctx context.Context, userID uuid.UUID, input PreferenceInput) (*PreferenceResponse, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*PreferenceResponse, error)
	Update(ctx context.Context, userID uuid.UUID, input PreferenceInput) (*PreferenceResponse, error)
	UpdateDataSharingConsent(ctx context.Context, userID uuid.UUID, consent bool) error
}
