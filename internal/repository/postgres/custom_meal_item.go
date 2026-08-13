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

// Create stores a custom meal and its associated tag rows in one transaction.
func (r *customMealRepository) Create(ctx context.Context, meal *model.CustomMealItem) (*model.CustomMealItem, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		dietaryRestrictionTags := meal.DietaryRestrictionTags
		mealCategoryTags := meal.MealCategoryTags

		meal.DietaryRestrictionTags = nil
		meal.MealCategoryTags = nil

		if err := tx.Create(meal).Error; err != nil {
			return err
		}

		for i := range dietaryRestrictionTags {
			dietaryRestrictionTags[i].CustomMealItemID = meal.ID
		}

		for i := range mealCategoryTags {
			mealCategoryTags[i].CustomMealItemID = meal.ID
		}

		if len(dietaryRestrictionTags) > 0 {
			if err := tx.Create(&dietaryRestrictionTags).Error; err != nil {
				return err
			}
		}

		if len(mealCategoryTags) > 0 {
			if err := tx.Create(&mealCategoryTags).Error; err != nil {
				return err
			}
		}

		meal.DietaryRestrictionTags = dietaryRestrictionTags
		meal.MealCategoryTags = mealCategoryTags

		return nil
	})

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

// UpdateOwned updates a custom meal only when it belongs to the specified user.
// Tag rows are replaced inside the same transaction so the meal and tags stay consistent.
func (r *customMealRepository) UpdateOwned(ctx context.Context, userID uuid.UUID, meal *model.CustomMealItem) (*model.CustomMealItem, error) {
	var updatedMeal model.CustomMealItem

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.CustomMealItem{}).
			Where("id = ? AND created_by = ?", meal.ID, userID).
			Updates(map[string]any{
				"name":            meal.Name,
				"price":           meal.Price,
				"calories":        meal.Calories,
				"fat_g":           meal.FatG,
				"protein_g":       meal.ProteinG,
				"carbs_g":         meal.CarbsG,
				"state":           meal.State,
				"district":        meal.District,
				"restaurant_name": meal.RestaurantName,
			})

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		if err := tx.Where("custom_meal_item_id = ?", meal.ID).
			Delete(&model.CustomMealDietaryRestrictionTag{}).Error; err != nil {
			return err
		}

		if err := tx.Where("custom_meal_item_id = ?", meal.ID).
			Delete(&model.CustomMealCategory{}).Error; err != nil {
			return err
		}

		for i := range meal.DietaryRestrictionTags {
			meal.DietaryRestrictionTags[i].CustomMealItemID = meal.ID
		}

		for i := range meal.MealCategoryTags {
			meal.MealCategoryTags[i].CustomMealItemID = meal.ID
		}

		if len(meal.DietaryRestrictionTags) > 0 {
			if err := tx.Create(&meal.DietaryRestrictionTags).Error; err != nil {
				return err
			}
		}

		if len(meal.MealCategoryTags) > 0 {
			if err := tx.Create(&meal.MealCategoryTags).Error; err != nil {
				return err
			}
		}

		return tx.
			Preload("DietaryRestrictionTags").
			Preload("MealCategoryTags").
			Where("id = ? AND created_by = ?", meal.ID, userID).
			First(&updatedMeal).Error
	})

	if err != nil {
		return nil, err
	}

	return &updatedMeal, nil
}

// DeleteOwned soft-deletes a custom meal only when it belongs to the specified
// user. Existing meal-log history should keep its own snapshots and should not
// depend on this source row after logging.
func (r *customMealRepository) DeleteOwned(ctx context.Context, userID uuid.UUID, customMealID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND created_by = ?", customMealID, userID).
		Delete(&model.CustomMealItem{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
