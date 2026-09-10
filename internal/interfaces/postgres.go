package interfaces

import (
	"context"
	"fyp/food-rs/types/model"
	"time"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, data *model.User) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	UpdateOnboardingStatus(ctx context.Context, id uuid.UUID, completed bool) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, refreshToken *model.RefreshToken) error
	Rotate(ctx context.Context, oldToken uuid.UUID, newToken *model.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*model.RefreshToken, error)
	RevokeByHash(ctx context.Context, hash string) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
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
	UpdateOwned(ctx context.Context, userID uuid.UUID, meal *model.CustomMealItem) (*model.CustomMealItem, error)
	DeleteOwned(ctx context.Context, userID uuid.UUID, customMealID uuid.UUID) error
}

type MealLogRepository interface {
	Create(ctx context.Context, mealLog *model.MealLog) (*model.MealLog, error)
	ListByUserAndMonth(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]model.MealLog, error)
	ListByUserAndRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]model.MealLog, error)

	FindByIDAndUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.MealLog, error)
	Update(ctx context.Context, mealLog *model.MealLog) (*model.MealLog, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}
