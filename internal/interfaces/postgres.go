package interfaces

import (
	"context"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, data *model.User) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdateOnboardingStatus(ctx context.Context, id uuid.UUID, completed bool) error
}

type PreferenceRepository interface {
	FindByUserID(ctx context.Context, userID uuid.UUID) (*model.UserPreference, error)
	Upsert(ctx context.Context, preference *model.UserPreference) (*model.UserPreference, error)
	UpdateDataSharingConsent(ctx context.Context, userID uuid.UUID, consent bool) error
}

type CustomMealRepository interface {
	Create(ctx context.Context, meal *model.CustomMealItem) (*model.CustomMealItem, error)
	ListOwnedByUser(ctx context.Context, userID uuid.UUID, query string) ([]model.CustomMealItem, error)
	ListSharedFromOtherUsers(ctx context.Context, userID uuid.UUID, query string) ([]model.CustomMealItem, error)
	FindVisibleByID(ctx context.Context, userID uuid.UUID, customMealID uuid.UUID) (*model.CustomMealItem, error)
}
