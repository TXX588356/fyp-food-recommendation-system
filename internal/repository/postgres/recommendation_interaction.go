package postgres

import (
	"context"
	"fyp/food-rs/internal/interfaces"
	"fyp/food-rs/types/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type recommendationInteractionRepository struct {
	db *gorm.DB
}

func NewRecommendationInteractionPostgresRepository(db *gorm.DB) interfaces.RecommendationInteractionRepository {
	return &recommendationInteractionRepository{db: db}
}

func (r *recommendationInteractionRepository) CreateImpressions(ctx context.Context, interactions []model.RecommendationInteraction) error {
	if len(interactions) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Create(&interactions).Error
}

func (r *recommendationInteractionRepository) MarkClicked(ctx context.Context, recommendationID uuid.UUID, userID uuid.UUID, mealSource string, mealID uuid.UUID, clickedAt time.Time) error {
	return r.updateInteraction(ctx, recommendationID, userID, mealSource, mealID, map[string]any{
		"clicked_at": clickedAt,
		"updated_at": gorm.Expr("now()"),
	})
}

func (r *recommendationInteractionRepository) MarkSelected(ctx context.Context, recommendationID uuid.UUID, userID uuid.UUID, mealSource string, mealID uuid.UUID, selectedAt time.Time) error {
	return r.updateInteraction(ctx, recommendationID, userID, mealSource, mealID, map[string]any{
		"selected_at": selectedAt,
		"updated_at":  gorm.Expr("now()"),
	})
}

func (r *recommendationInteractionRepository) MarkLogged(ctx context.Context, recommendationID uuid.UUID, userID uuid.UUID, mealSource string, mealID uuid.UUID, loggedAt time.Time) error {
	return r.updateInteraction(ctx, recommendationID, userID, mealSource, mealID, map[string]any{
		"logged_at":  loggedAt,
		"updated_at": gorm.Expr("now()"),
	})
}

func (r *recommendationInteractionRepository) SetRating(ctx context.Context, recommendationID uuid.UUID, userID uuid.UUID, mealSource string, mealID uuid.UUID, rating int) error {
	return r.updateInteraction(ctx, recommendationID, userID, mealSource, mealID, map[string]any{
		"rating":     rating,
		"updated_at": gorm.Expr("now()"),
	})
}

func (r *recommendationInteractionRepository) updateInteraction(ctx context.Context, recommendationID uuid.UUID, userID uuid.UUID, mealSource string, mealID uuid.UUID, updates map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&model.RecommendationInteraction{}).
		Where("recommendation_id = ? AND user_id = ? AND meal_source = ? AND meal_id = ?", recommendationID, userID, mealSource, mealID).
		Updates(updates).
		Error
}
