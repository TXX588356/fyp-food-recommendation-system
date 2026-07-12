package postgres

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mealLogRepository struct {
	db *gorm.DB
}

func NewMealLogPostgresRepository(db *gorm.DB) interfaces.MealLogRepository {
	return &mealLogRepository{db: db}
}

func (r *mealLogRepository) Create(ctx context.Context, mealLog *model.MealLog) (*model.MealLog, error) {
	if err := r.db.WithContext(ctx).Create(mealLog).Error; err != nil {
		return nil, err
	}

	return mealLog, nil
}

func (r *mealLogRepository) ListByUserAndMonth(ctx context.Context, userID uuid.UUID, start time.Time, end time.Time) ([]model.MealLog, error) {
	var logs []model.MealLog

	err := r.db.WithContext(ctx).
		Where("user_id = ? AND eaten_at >= ? AND eaten_at < ? AND deleted_at IS NULL", userID, start, end).
		Order("eaten_at DESC").
		Find(&logs).Error

	return logs, err
}

func (r *mealLogRepository) ListByUserAndRange(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]model.MealLog, error) {
	var logs []model.MealLog

	err := mealLogsByUserAndRange(r.db.WithContext(ctx), userID, start, end).Find(&logs).Error

	return logs, err
}

func (r *mealLogRepository) FindByIDAndUser(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.MealLog, error) {
	var log model.MealLog

	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		First(&log).Error

	if err != nil {
		return nil, err
	}

	return &log, nil
}

func (r *mealLogRepository) Update(ctx context.Context, mealLog *model.MealLog) (*model.MealLog, error) {
	if err := r.db.WithContext(ctx).Save(mealLog).Error; err != nil {
		return nil, err
	}

	return mealLog, nil
}

func (r *mealLogRepository) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.MealLog{}).
		Error
}

func mealLogsByUserAndRange(db *gorm.DB, userID uuid.UUID, start, end time.Time) *gorm.DB {
	return db.
		Where("user_id = ? AND eaten_at >= ? AND eaten_at < ? AND deleted_at IS NULL", userID, start, end).
		Order("eaten_at DESC")
}
