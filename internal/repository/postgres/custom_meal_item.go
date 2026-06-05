package postgres

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type customMealRepository struct {
	db *gorm.DB
}

// NewCustomMealPostgresRepository creates a Postgres-backed custom meal repository.
func NewCustomMealPostgresRepository(db *gorm.DB) interfaces.CustomMealRepository {
	return &customMealRepository{db: db}
}

// Create stores a custom meal and its associated tag rows.
func (r *customMealRepository) Create(ctx context.Context, meal *model.CustomMealItem) (*model.CustomMealItem, error) {
	err := r.db.WithContext(ctx).Create(meal).Error

	return meal, err
}

// ListOwnedByUser returns custom meals created by the specified user.
// When query is provided, results are filtered by meal name.
func (r *customMealRepository) ListOwnedByUser(ctx context.Context, userID uuid.UUID, query string) ([]model.CustomMealItem, error) {
	var meals []model.CustomMealItem

	db := r.db.WithContext(ctx).
		Preload("DietaryRestrictionTags").
		Preload("MealCategoryTags").
		Where("created_by = ?", userID)

	if query != "" {
		db = db.Where("LOWER(name) LIKE LOWER(?)", "%"+query+"%")
	}

	if err := db.Find(&meals).Error; err != nil {
		return nil, err
	}

	return meals, nil
}

// ListSharedFromOtherUsers returns custom meals created by other users whose
// owners have enabled data sharing. When query is provided, results are filtered by meal name.
func (r *customMealRepository) ListSharedFromOtherUsers(ctx context.Context, userID uuid.UUID, query string) ([]model.CustomMealItem, error) {
	var meals []model.CustomMealItem

	db := r.db.WithContext(ctx).
		Preload("DietaryRestrictionTags").
		Preload("MealCategoryTags").
		Joins("JOIN user_preferences ON user_preferences.user_id = custom_meal_items.created_by").
		Where("custom_meal_items.created_by <> ?", userID). // search for meals not created by current user
		Where("user_preferences.data_sharing_consent = ?", true)

	if query != "" {
		db = db.Where("LOWER(custom_meal_items.name) LIKE LOWER(?)", "%"+query+"%")
	}

	if err := db.Find(&meals).Error; err != nil {
		return nil, err
	}

	return meals, nil
}

// FindVisibleByID returns a custom meal if the user owns it or if the owner has
// enabled data sharing.
func (r *customMealRepository) FindVisibleByID(ctx context.Context, userID uuid.UUID, customMealID uuid.UUID) (*model.CustomMealItem, error) {
	var meal model.CustomMealItem

	err := r.db.WithContext(ctx).
		Preload("DietaryRestrictionTags").
		Preload("MealCategoryTags").
		Joins("LEFT JOIN user_preferences ON user_preferences.user_id = custom_meal_items.created_by").
		Where("custom_meal_items.id = ?", customMealID).
		// Only current user that owns the custom meal OR the meal owner's data sharing consent is true
		Where(r.db.Where("custom_meal_items.created_by = ?", userID).
			Or("user_preferences.data_sharing_consent = ?", true)).
		First(&meal).Error

	if err != nil {
		return nil, err
	}

	return &meal, nil
}
