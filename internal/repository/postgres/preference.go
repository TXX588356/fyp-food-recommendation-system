package postgres

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type preferenceRepository struct {
	db *gorm.DB
}

func NewPreferencePostgresRepository(db *gorm.DB) interfaces.PreferenceRepository {
	return &preferenceRepository{db: db}
}

func (r *preferenceRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*model.UserPreference, error) {
	var userPreference model.UserPreference

	err := r.db.WithContext(ctx).
		Preload("HealthConcerns").
		Preload("DietaryRestrictions").
		Preload("MealPreferences").
		Where("user_id = ?", userID).
		First(&userPreference).Error
	return &userPreference, err
}

// Transactional since writing to four tables
func (r *preferenceRepository) Upsert(ctx context.Context, preference *model.UserPreference) (*model.UserPreference, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"main_goal",
				"monthly_meal_budget",
				"data_sharing_consent",
				"home_location",
				"work_school_location",
				"updated_at",
			}),
			// GORM can try to create associations when creating the parent. Since manual delete/recreate child rows is done afterward, make the parent upsert omit associations
		}).Omit("HealthConcerns", "DietaryRestrictions", "MealPreferences").
			Create(preference).Error; err != nil {
			return err
		}

		if err := tx.Where("user_id = ?", preference.UserID).Delete(&model.UserHealthConcern{}).Error; err != nil {
			return err
		}
		if len(preference.HealthConcerns) > 0 {
			if err := tx.Create(&preference.HealthConcerns).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("user_id = ?", preference.UserID).Delete(&model.UserDietaryRestriction{}).Error; err != nil {
			return err
		}
		if len(preference.DietaryRestrictions) > 0 {
			if err := tx.Create(&preference.DietaryRestrictions).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("user_id = ?", preference.UserID).Delete(&model.UserMealPreference{}).Error; err != nil {
			return err
		}
		if len(preference.MealPreferences) > 0 {
			if err := tx.Create(&preference.MealPreferences).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return r.FindByUserID(ctx, preference.UserID)
}

func (r *preferenceRepository) UpdateDataSharingConsent(ctx context.Context, userID uuid.UUID, consent bool) error {
	return r.db.WithContext(ctx).
		Model(&model.UserPreference{}).
		Where("user_id = ?", userID).
		Updates(map[string]any{
			"data_sharing_consent": consent,
			"updated_at":           gorm.Expr("now()"),
		}).Error
}
